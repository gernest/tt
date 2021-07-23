package store

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/open-policy-agent/opa/storage"
)

type Data = _struct.Value

func ptr2(data *Data, path storage.Path) (interface{}, error) {
	return ptr(data, path)
}

func ptr(data *Data, path storage.Path) (*Data, error) {
	node := data
	for i := range path {
		key := path[i]
		switch curr := node.Kind.(type) {
		case *_struct.Value_StructValue:
			var ok bool
			if node, ok = curr.StructValue.GetFields()[key]; !ok {
				return nil, notFoundError(path)
			}
		case *_struct.Value_ListValue:
			pos, err := validateArrayIndex(curr.ListValue.Values, key, path)
			if err != nil {
				return nil, err
			}
			node = curr.ListValue.Values[pos]
		default:
			return nil, notFoundError(path)
		}
	}

	return node, nil
}

func validateArrayIndex(arr []*_struct.Value, s string, path storage.Path) (int, error) {
	idx, err := strconv.Atoi(s)
	if err != nil {
		return 0, notFoundErrorHint(path, arrayIndexTypeMsg)
	}
	if idx < 0 || idx >= len(arr) {
		return 0, notFoundErrorHint(path, outOfRangeMsg)
	}
	return idx, nil
}

func deleteData(data *Data, key string) {
	if s := data.GetStructValue(); s != nil {
		delete(s.GetFields(), key)
	}
}

var um jsonpb.Unmarshaler

func Parse(in interface{}) *Data {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(buf)
	var data _struct.Value
	um.Unmarshal(&buf, &data)
	return &data
}

func Marshal(d *Data) ([]byte, error) {
	return proto.Marshal(d)
}
