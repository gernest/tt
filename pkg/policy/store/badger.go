// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// package store implements an in-memory version of the policy engine's storage
// layer.
//
// The in-memory store is used as the default storage layer implementation. The
// in-memory store supports multi-reader/single-writer concurrency with
// rollback.
//
// Callers should assume the in-memory store does not make copies of written
// data. Once data is written to the in-memory store, it should not be modified
// (outside of calling Store.Write). Furthermore, data read from the in-memory
// store should be treated as read-only.
package store

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/dgraph-io/badger/v3"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/util"
)

// New returns an empty in-memory store.
func New(db *badger.DB, data *Data) storage.Store {
	if data == nil {
		data = &Data{}
	}
	return &store{
		data:     data,
		triggers: map[*handle]storage.TriggerConfig{},
		db:       db,
	}
}

// NewFromObject returns a new in-memory store from the supplied data object.
func NewFromObject(bdb *badger.DB, data map[string]interface{}) storage.Store {
	db := New(bdb, Parse(data))
	ctx := context.Background()
	txn, err := db.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		panic(err)
	}
	if err := db.Write(ctx, txn, storage.AddOp, storage.Path{}, data); err != nil {
		panic(err)
	}
	if err := db.Commit(ctx, txn); err != nil {
		panic(err)
	}
	return db
}

// NewFromReader returns a new in-memory store from a reader that produces a
// JSON serialized object. This function is for test purposes.
func NewFromReader(db *badger.DB, r io.Reader) storage.Store {
	d := util.NewJSONDecoder(r)
	var data map[string]interface{}
	if err := d.Decode(&data); err != nil {
		panic(err)
	}
	return NewFromObject(db, data)
}

type store struct {
	rmu      sync.RWMutex // reader-writer lock
	wmu      sync.Mutex   // writer lock
	data     *Data        // raw data
	db       *badger.DB
	triggers map[*handle]storage.TriggerConfig // registered triggers
}

type handle struct {
	db *store
}

func (db *store) NewTransaction(ctx context.Context, params ...storage.TransactionParams) (storage.Transaction, error) {
	var write bool
	var context *storage.Context
	if len(params) > 0 {
		write = params[0].Write
		context = params[0].Context
	}
	xtn := db.db.NewTransaction(write)
	xid := xtn.ReadTs()
	if write {
		db.wmu.Lock()
	} else {
		db.rmu.RLock()
	}
	return newTransaction(xid, write, context, db, xtn), nil
}

func (db *store) closeTransaction(txn *transaction) error {
	defer txn.Commit()
	if txn.write {
		b, err := Marshal(db.data)
		if err != nil {
			return err
		}
		if err := txn.txn.Set([]byte(data), b); err != nil {
			return err
		}
	}
	return nil
}

func (db *store) Commit(ctx context.Context, txn storage.Transaction) error {
	underlying, err := db.underlying(txn)
	if err != nil {
		return err
	}
	defer db.closeTransaction(underlying)

	if underlying.write {
		db.rmu.Lock()
		event := underlying.Commit()
		db.runOnCommitTriggers(ctx, txn, event)
		// Mark the transaction stale after executing triggers so they can
		// perform store operations if needed.
		underlying.stale = true
		db.rmu.Unlock()
		db.wmu.Unlock()
	} else {
		db.rmu.RUnlock()
	}
	return nil
}

func (db *store) Abort(ctx context.Context, txn storage.Transaction) {
	underlying, err := db.underlying(txn)
	if err != nil {
		panic(err)
	}
	underlying.stale = true
	if underlying.write {
		db.wmu.Unlock()
	} else {
		db.rmu.RUnlock()
	}
}

func (db *store) ListPolicies(_ context.Context, txn storage.Transaction) ([]string, error) {
	underlying, err := db.underlying(txn)
	if err != nil {
		return nil, err
	}
	return underlying.ListPolicies(), nil
}

func (db *store) GetPolicy(_ context.Context, txn storage.Transaction, id string) ([]byte, error) {
	underlying, err := db.underlying(txn)
	if err != nil {
		return nil, err
	}
	return underlying.GetPolicy(id)
}

func (db *store) UpsertPolicy(_ context.Context, txn storage.Transaction, id string, bs []byte) error {
	underlying, err := db.underlying(txn)
	if err != nil {
		return err
	}
	return underlying.UpsertPolicy(id, bs)
}

func (db *store) DeletePolicy(_ context.Context, txn storage.Transaction, id string) error {
	underlying, err := db.underlying(txn)
	if err != nil {
		return err
	}
	if _, err := underlying.GetPolicy(id); err != nil {
		return err
	}
	return underlying.DeletePolicy(id)
}

func (db *store) Register(ctx context.Context, txn storage.Transaction, config storage.TriggerConfig) (storage.TriggerHandle, error) {
	underlying, err := db.underlying(txn)
	if err != nil {
		return nil, err
	}
	if !underlying.write {
		return nil, &storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: "triggers must be registered with a write transaction",
		}
	}
	h := &handle{db}
	db.triggers[h] = config
	return h, nil
}

func (db *store) Read(ctx context.Context, txn storage.Transaction, path storage.Path) (interface{}, error) {
	underlying, err := db.underlying(txn)
	if err != nil {
		return nil, err
	}
	return underlying.Read(path)
}

func (db *store) Write(ctx context.Context, txn storage.Transaction, op storage.PatchOp, path storage.Path, value interface{}) error {
	underlying, err := db.underlying(txn)
	if err != nil {
		return err
	}
	val := util.Reference(value)
	if err := util.RoundTrip(val); err != nil {
		return err
	}
	return underlying.Write(op, path, *val)
}

func (h *handle) Unregister(ctx context.Context, txn storage.Transaction) {
	underlying, err := h.db.underlying(txn)
	if err != nil {
		panic(err)
	}
	if !underlying.write {
		panic(&storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: "triggers must be unregistered with a write transaction",
		})
	}
	delete(h.db.triggers, h)
}

func (db *store) runOnCommitTriggers(ctx context.Context, txn storage.Transaction, event storage.TriggerEvent) {
	for _, t := range db.triggers {
		t.OnCommit(ctx, txn, event)
	}
}

func (db *store) underlying(txn storage.Transaction) (*transaction, error) {
	underlying, ok := txn.(*transaction)
	if !ok {
		return nil, &storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: fmt.Sprintf("unexpected transaction type %T", txn),
		}
	}
	if underlying.db != db {
		return nil, &storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: "unknown transaction",
		}
	}
	if underlying.stale {
		return nil, &storage.Error{
			Code:    storage.InvalidTransactionErr,
			Message: "stale transaction",
		}
	}
	return underlying, nil
}

var doesNotExistMsg = "document does not exist"
var rootMustBeObjectMsg = "root must be object"
var rootCannotBeRemovedMsg = "root cannot be removed"
var outOfRangeMsg = "array index out of range"
var arrayIndexTypeMsg = "array index must be integer"

func invalidPatchError(f string, a ...interface{}) *storage.Error {
	return &storage.Error{
		Code:    storage.InvalidPatchErr,
		Message: fmt.Sprintf(f, a...),
	}
}

func notFoundError(path storage.Path) *storage.Error {
	return notFoundErrorHint(path, doesNotExistMsg)
}

func notFoundErrorHint(path storage.Path, hint string) *storage.Error {
	return notFoundErrorf("%v: %v", path.String(), hint)
}

func notFoundErrorf(f string, a ...interface{}) *storage.Error {
	msg := fmt.Sprintf(f, a...)
	return &storage.Error{
		Code:    storage.NotFoundErr,
		Message: msg,
	}
}
