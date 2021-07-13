package cluster

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	raftbadger "github.com/BBVA/raft-badger"
	transport "github.com/Jille/raft-grpc-transport"
	"github.com/dgraph-io/badger/v3"
	"github.com/gernest/tt/pkg/zlg"
	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
)

func NewRaft(
	ctx context.Context,
	bootstrap bool,
	nodeID string,
	nodeAddr string,
	fsm raft.FSM,
	dataPath string,
) (*raft.Raft, *transport.Manager, error) {
	c := raft.DefaultConfig()
	c.LocalID = raft.ServerID(nodeID)
	basePath := filepath.Join(dataPath, nodeID)
	store, err := raftbadger.NewBadgerStore(filepath.Join(basePath, "db"))
	if err != nil {
		return nil, nil, err
	}
	fss, err := raft.NewFileSnapshotStore(
		filepath.Join(basePath, "snapshots"), 3, os.Stderr,
	)
	if err != nil {
		return nil, nil, err
	}
	tm := transport.New(raft.ServerAddress(nodeAddr), []grpc.DialOption{grpc.WithInsecure()})
	r, err := raft.NewRaft(c, fsm, store, store, fss, tm.Transport())
	if err != nil {
		return nil, nil, fmt.Errorf("raft.NewRaft: %v", err)
	}
	if bootstrap {
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(nodeID),
					Address:  raft.ServerAddress(nodeAddr),
				},
			},
		}
		f := r.BootstrapCluster(cfg)
		if err := f.Error(); err != nil {
			return nil, nil, fmt.Errorf("raft.Raft.BootstrapCluster: %v", err)
		}
	}
	return r, tm, nil
}

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
	return nil
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
