package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	raftbadger "github.com/BBVA/raft-badger"
	transport "github.com/Jille/raft-grpc-transport"
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
