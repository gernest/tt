package cluster

import (
	"io"

	"github.com/dgraph-io/badger/v3"
	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/zlg"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
)

// FSM implements FSM but stores data in badger
type FSM struct {
	db *badger.DB
}

func NewFSM(path string) (*FSM, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = &Badger{zlg.Logger.Named("raft-fsm").Sugar()}
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &FSM{db: db}, nil
}

func (fsm *FSM) Close() error {
	return fsm.db.Close()
}

var _ raft.FSM = (*FSM)(nil)

func (fsm *FSM) Apply(log *raft.Log) interface{} {
	e, err := fsm.entry(log)
	if err != nil {
		return err
	}
	if kv := e.GetKeyValue(); kv != nil {
		return fsm.kv(kv)
	}
	return nil
}

func (fsm *FSM) entry(log *raft.Log) (*api.Raft_Log, error) {
	var r api.Raft_Log
	err := proto.Unmarshal(log.Data, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (fsm *FSM) kv(e *api.Raft_KeyValue) interface{} {
	switch e.Action {
	case api.Raft_KeyValue_GET:
		var o api.Raft_KeyValue_Context
		err := fsm.db.View(func(txn *badger.Txn) error {
			i, err := txn.Get(e.Context.Key)
			if err != nil {
				return err
			}
			v, err := i.ValueCopy(nil)
			o.Value = v
			o.ExpiresAt = i.ExpiresAt()
			return nil
		})
		if err != nil {
			return err
		}
		return &o
	case api.Raft_KeyValue_SET:
		return fsm.db.Update(func(txn *badger.Txn) error {
			en := badger.NewEntry(e.Context.Key, e.Context.Value)
			if e.Context.ExpiresAt != 0 {
				en.ExpiresAt = e.Context.ExpiresAt
			}
			return txn.SetEntry(en)
		})
	default:
		return nil
	}
}

func (fsm *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return fsm, nil
}

func (fsm *FSM) Persist(sink raft.SnapshotSink) error {
	if _, err := fsm.db.Backup(sink, 0); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (fsm *FSM) Release() {}

func (fsm *FSM) Restore(r io.ReadCloser) error {
	return fsm.db.Load(r, 5)
}
