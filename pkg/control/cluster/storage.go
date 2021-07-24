package cluster

import (
	"context"
	"time"

	"github.com/gernest/tt/api"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
)

var _ api.StorageServer = (*DB)(nil)

type DB struct {
	api.UnimplementedStorageServer
	raft    *raft.Raft
	timeout time.Duration
}

func (db *DB) Set(ctx context.Context, in *api.Store_SetRequest) (*api.Store_SetRequest, error) {
	_, err := db.send(&api.Raft_Log{
		Entry: &api.Raft_Log_KeyValue{
			KeyValue: &api.Raft_KeyValue{
				Action: api.Raft_KeyValue_SET,
				Context: &api.Raft_KeyValue_Context{
					Key:       in.Key,
					Value:     in.Value,
					ExpiresAt: in.ExpiresAt,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &api.Store_SetRequest{}, nil
}

func (db *DB) send(entry *api.Raft_Log) (interface{}, error) {
	m, err := proto.Marshal(entry)
	if err != nil {
		return nil, err
	}
	a := db.raft.Apply(m, db.timeout)
	if a.Error() != nil {
		return nil, err
	}
	res := a.Response()
	if e, ok := res.(error); ok {
		return nil, e
	}
	return res, nil
}

func (db *DB) Get(ctx context.Context, in *api.Store_GetRequest) (*api.Store_GetResponse, error) {
	v, err := db.send(&api.Raft_Log{
		Entry: &api.Raft_Log_KeyValue{
			KeyValue: &api.Raft_KeyValue{
				Action: api.Raft_KeyValue_GET,
				Context: &api.Raft_KeyValue_Context{
					Key: in.Key,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	n := v.(*api.Raft_KeyValue_Context)
	return &api.Store_GetResponse{
		Value:     n.Value,
		ExpiresAt: n.ExpiresAt,
	}, nil
}
