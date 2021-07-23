// Copyright 2017 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package store

import (
	"container/list"
	"errors"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/golang/protobuf/proto"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/open-policy-agent/opa/storage"
)

type dbKey string

const (
	prefix dbKey = "/policies/"
	data   dbKey = "/data/"
)

func (k dbKey) Key(x string) []byte {
	return []byte(string(k) + x)
}

func (k dbKey) Strip(x string) string {
	return strings.TrimPrefix(x, string(k))
}

// transaction implements the low-level read/write operations on the in-memory
// store and contains the state required for pending transactions.
//
// For write transactions, the struct contains a logical set of updates
// performed by write operations in the transaction. Each write operation
// compacts the set such that two updates never overlap:
//
// - If new update path is a prefix of existing update path, existing update is
// removed, new update is added.
//
// - If existing update path is a prefix of new update path, existing update is
// modified.
//
// - Otherwise, new update is added.
//
// Read transactions do not require any special handling and simply passthrough
// to the underlying store. Read transactions do not support upgrade.
type transaction struct {
	xid      uint64
	write    bool
	stale    bool
	db       *store
	txn      *badger.Txn
	updates  *list.List
	policies map[string]policyUpdate
	context  *storage.Context
}

type policyUpdate struct {
	value  []byte
	remove bool
}

func newTransaction(xid uint64,
	write bool,
	context *storage.Context,
	db *store,
	txn *badger.Txn,
) *transaction {
	return &transaction{
		xid:      xid,
		write:    write,
		db:       db,
		txn:      txn,
		policies: map[string]policyUpdate{},
		updates:  list.New(),
		context:  context,
	}
}

func (txn *transaction) ID() uint64 {
	return txn.xid
}

func (txn *transaction) Write(op storage.PatchOp, path storage.Path, in interface{}) error {
	value := Parse(in)
	if !txn.write {
		return &storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: "data write during read transaction",
		}
	}

	if len(path) == 0 {
		return txn.updateRoot(op, value)
	}

	for curr := txn.updates.Front(); curr != nil; {
		update := curr.Value.(*update)

		// Check if new update masks existing update exactly. In this case, the
		// existing update can be removed and no other updates have to be
		// visited (because no two updates overlap.)
		if update.path.Equal(path) {
			if update.remove {
				if op != storage.AddOp {
					return notFoundError(path)
				}
			}
			txn.updates.Remove(curr)
			break
		}

		// Check if new update masks existing update. In this case, the
		// existing update has to be removed but other updates may overlap, so
		// we must continue.
		if update.path.HasPrefix(path) {
			remove := curr
			curr = curr.Next()
			txn.updates.Remove(remove)
			continue
		}

		// Check if new update modifies existing update. In this case, the
		// existing update is mutated.
		if path.HasPrefix(update.path) {
			if update.remove {
				return notFoundError(path)
			}
			suffix := path[len(update.path):]
			newUpdate, err := newUpdate(update.value, op, suffix, 0, value)
			if err != nil {
				return err
			}
			update.value = newUpdate.Apply(update.value)
			return nil
		}

		curr = curr.Next()
	}

	update, err := newUpdate(txn.db.data, op, path, 0, value)
	if err != nil {
		return err
	}

	txn.updates.PushFront(update)
	return nil
}

func (txn *transaction) updateRoot(op storage.PatchOp, value *Data) error {
	if op == storage.RemoveOp {
		return invalidPatchError(rootCannotBeRemovedMsg)
	}
	if s := value.GetStructValue(); s == nil {
		return invalidPatchError(rootMustBeObjectMsg)
	}
	txn.updates.Init()
	txn.updates.PushFront(&update{
		path:   storage.Path{},
		remove: false,
		value:  value,
	})
	return nil
}

func (txn *transaction) Commit() (result storage.TriggerEvent) {
	result.Context = txn.context
	for curr := txn.updates.Front(); curr != nil; curr = curr.Next() {
		action := curr.Value.(*update)
		updated := action.Apply(txn.db.data)
		txn.db.data = updated

		result.Data = append(result.Data, storage.DataEvent{
			Path:    action.path,
			Data:    action.value,
			Removed: action.remove,
		})
	}
	for id, update := range txn.policies {
		if update.remove {
			txn.txn.Delete(prefix.Key(id))
		} else {
			txn.txn.Set(prefix.Key(id), update.value)
		}

		result.Policy = append(result.Policy, storage.PolicyEvent{
			ID:      id,
			Data:    update.value,
			Removed: update.remove,
		})
	}
	return result
}

func (txn *transaction) Read(path storage.Path) (interface{}, error) {

	if !txn.write {
		return ptr(txn.db.data, path)
	}

	merge := []*update{}

	for curr := txn.updates.Front(); curr != nil; curr = curr.Next() {

		update := curr.Value.(*update)

		if path.HasPrefix(update.path) {
			if update.remove {
				return nil, notFoundError(path)
			}
			return ptr(update.value, path[len(update.path):])
		}

		if update.path.HasPrefix(path) {
			merge = append(merge, update)
		}
	}

	data, err := ptr(txn.db.data, path)

	if err != nil {
		return nil, err
	}

	if len(merge) == 0 {
		return data, nil
	}

	cpy := deepCopy(data)

	for _, update := range merge {
		cpy = update.Relative(path).Apply(cpy)
	}

	return cpy, nil
}

func deepCopy(v *Data) *Data {
	return proto.Clone(v).(*Data)
}

func (txn *transaction) ListPolicies() []string {
	var ids []string
	o := badger.DefaultIteratorOptions
	o.Prefix = []byte(prefix)
	it := txn.txn.NewIterator(o)
	for it.Rewind(); it.ValidForPrefix(o.Prefix); it.Next() {
		k := it.Item().Key()
		ids = append(ids, prefix.Strip(string(k)))
	}
	for id, update := range txn.policies {
		if !update.remove {
			ids = append(ids, id)
		}
	}
	return ids
}

func (txn *transaction) GetPolicy(id string) ([]byte, error) {
	if update, ok := txn.policies[id]; ok {
		if !update.remove {
			return update.value, nil
		}
		return nil, notFoundErrorf("policy id %q", id)
	}
	v, err := txn.txn.Get(prefix.Key(id))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, notFoundErrorf("policy id %q", id)
		}
		return nil, err
	}
	return v.ValueCopy(nil)
}

func (txn *transaction) UpsertPolicy(id string, bs []byte) error {
	if !txn.write {
		return &storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: "policy write during read transaction",
		}
	}
	txn.policies[id] = policyUpdate{bs, false}
	return nil
}

func (txn *transaction) DeletePolicy(id string) error {
	if !txn.write {
		return &storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: "policy write during read transaction",
		}
	}
	txn.policies[id] = policyUpdate{nil, true}
	return nil
}

// update contains state associated with an update to be applied to the
// in-memory data store.
type update struct {
	path   storage.Path // data path modified by update
	remove bool         // indicates whether update removes the value at path
	value  *Data        // value to add/replace at path (ignored if remove is true)
}

func newUpdate(data *Data, op storage.PatchOp, path storage.Path, idx int, value *Data) (*update, error) {

	switch data := data.Kind.(type) {
	case *_struct.Value_StructValue:
		return newUpdateObject(data.StructValue, op, path, idx, value)

	case *_struct.Value_ListValue:
		return newUpdateArray(data.ListValue.GetValues(), op, path, idx, value)

	case nil,
		*_struct.Value_BoolValue,
		*_struct.Value_NumberValue,
		*_struct.Value_StringValue:
		return nil, notFoundError(path)
	}

	return nil, &storage.Error{
		Code:    storage.InternalErr,
		Message: "invalid data value encountered",
	}
}

func listValue(data []*_struct.Value) *_struct.Value {
	return &_struct.Value{
		Kind: &_struct.Value_ListValue{
			ListValue: &_struct.ListValue{
				Values: data,
			},
		},
	}
}

func newUpdateArray(data []*_struct.Value, op storage.PatchOp, path storage.Path, idx int, value *Data) (*update, error) {
	if idx == len(path)-1 {
		if path[idx] == "-" {
			if op != storage.AddOp {
				return nil, invalidPatchError("%v: invalid patch path", path)
			}
			cpy := make([]*_struct.Value, len(data)+1)
			copy(cpy, data)
			cpy[len(data)] = value
			return &update{path[:len(path)-1], false, listValue(cpy)}, nil
		}

		pos, err := validateArrayIndex(data, path[idx], path)
		if err != nil {
			return nil, err
		}

		if op == storage.AddOp {
			cpy := make([]*_struct.Value, len(data)+1)
			copy(cpy[:pos], data[:pos])
			copy(cpy[pos+1:], data[pos:])
			cpy[pos] = value
			return &update{path[:len(path)-1], false, listValue(cpy)}, nil

		} else if op == storage.RemoveOp {
			cpy := make([]*_struct.Value, len(data)-1)
			copy(cpy[:pos], data[:pos])
			copy(cpy[pos:], data[pos+1:])
			return &update{path[:len(path)-1], false, listValue(cpy)}, nil

		} else {
			cpy := make([]*_struct.Value, len(data))
			copy(cpy, data)
			cpy[pos] = value
			return &update{path[:len(path)-1], false, listValue(cpy)}, nil
		}
	}

	pos, err := validateArrayIndex(data, path[idx], path)
	if err != nil {
		return nil, err
	}

	return newUpdate(data[pos], op, path, idx+1, value)
}

func newUpdateObject(data *_struct.Struct, op storage.PatchOp, path storage.Path, idx int, value *Data) (*update, error) {
	if idx == len(path)-1 {
		switch op {
		case storage.ReplaceOp, storage.RemoveOp:
			if _, ok := data.GetFields()[path[idx]]; !ok {
				return nil, notFoundError(path)
			}
		}
		return &update{path, op == storage.RemoveOp, value}, nil
	}

	if data, ok := data.GetFields()[path[idx]]; ok {
		return newUpdate(data, op, path, idx+1, value)
	}

	return nil, notFoundError(path)
}

func (u *update) Apply(data *Data) *Data {
	if len(u.path) == 0 {
		return u.value
	}
	parent, err := ptr(data, u.path[:len(u.path)-1])
	if err != nil {
		panic(err)
	}
	key := u.path[len(u.path)-1]
	if u.remove {
		deleteData(parent, key)
		return data
	}
	if s := parent.GetStructValue(); s != nil {
		s.GetFields()[key] = u.value
	}
	if ls := parent.GetListValue(); ls != nil {
		idx, err := strconv.Atoi(key)
		if err != nil {
			panic(err)
		}
		ls.Values[idx] = u.value
	}
	return data
}

func (u *update) Relative(path storage.Path) *update {
	cpy := *u
	cpy.path = cpy.path[len(path):]
	return &cpy
}
