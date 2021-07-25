package cluster

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	raftbadger "github.com/BBVA/raft-badger"
	"github.com/dgraph-io/badger/v3"
	"github.com/gernest/tt/pkg/zlg"
	"github.com/hashicorp/raft"
)

func NewRaft(
	bootstrap bool,
	nodeID string,
	nodeAddr string,
	fsm raft.FSM,
	dataPath string,
) (*raft.Raft, error) {
	c := raft.DefaultConfig()
	c.LocalID = raft.ServerID(nodeID)
	raftLog := zlg.Logger.Named("raft")
	c.Logger = NewHCLogAdapter(raftLog)
	basePath := filepath.Join(dataPath, nodeID)
	storeOpts := badger.DefaultOptions(filepath.Join(basePath, "db"))
	storeOpts.Logger = &Badger{raftLog.Named("store").Sugar()}
	store, err := raftbadger.New(raftbadger.Options{
		Path:          filepath.Join(basePath, "db"),
		BadgerOptions: &storeOpts,
	})
	if err != nil {
		return nil, err
	}
	fss, err := raft.NewFileSnapshotStoreWithLogger(
		filepath.Join(basePath, "snapshots"), 3,
		NewHCLogAdapter(raftLog.Named("file-snapshot")),
	)
	if err != nil {
		return nil, err
	}
	nodeAddr = formatNodeAdr(nodeAddr)
	addr, err := net.ResolveTCPAddr("tcp", nodeAddr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(nodeAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}
	r, err := raft.NewRaft(c, fsm, store, store, fss, transport)
	if err != nil {
		return nil, fmt.Errorf("raft.NewRaft: %v", err)
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
			return nil, fmt.Errorf("raft.Raft.BootstrapCluster: %v", err)
		}
	}
	return r, nil
}

func formatNodeAdr(addr string) string {
	if addr[0] == ':' {
		return "localhost" + addr
	}
	return addr
}
