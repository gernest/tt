package cluster

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	raftbadger "github.com/BBVA/raft-badger"
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
	basePath := filepath.Join(dataPath, nodeID)
	store, err := raftbadger.NewBadgerStore(filepath.Join(basePath, "db"))
	if err != nil {
		return nil, err
	}
	fss, err := raft.NewFileSnapshotStore(
		filepath.Join(basePath, "snapshots"), 3, os.Stderr,
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
