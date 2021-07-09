package xhttp

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/gernest/tt/api"
	"github.com/golang/protobuf/jsonpb"
)

func TestGenerateTestConfig(t *testing.T) {
	generateSample(sample0)
}

func generateSample(routes ...*api.Route) {
	var m jsonpb.Marshaler
	m.Indent = "  "
	var o bytes.Buffer
	for _, v := range routes {
		o.Reset()
		m.Marshal(&o, v)
		ioutil.WriteFile(filepath.Join("samples", v.Name)+".json", o.Bytes(), 0600)
	}
}

var sample0 = &api.Route{
	Name:     "httpbin",
	Protocol: api.Protocol_HTTP,
	LoadBalance: []*api.WeightedAddr{
		{
			Addr: &api.Address{
				Address: "https://httpbin.org",
			},
		},
	},
	Middlewares: &api.Middleware_List{
		List: []*api.Middleware{
			{
				Match: &api.Middleware_StripPathPrefix_{
					StripPathPrefix: &api.Middleware_StripPathPrefix{
						Prefix: "/httpbin/",
					},
				},
			},
		},
	},
	Rule: &api.Rule{
		Match: &api.Rule_Http{
			Http: &api.Rule_HTTP{
				Match: &api.Rule_HTTP_Path{
					Path: &api.Rule_StringMatch{
						Match: &api.Rule_StringMatch_Prefix{
							Prefix: "/httpbin",
						},
					},
				},
			},
		},
	},
}