package proxy

import (
	"fmt"

	"github.com/gernest/tt/api"
)

func BindToHostPort(b *api.Bind, defaultHostPort string) string {
	if b == nil {
		return defaultHostPort
	}
	var h string
	switch e := b.To.(type) {
	case *api.Bind_Port:
		h = fmt.Sprintf(":%d", e.Port)
	case *api.Bind_HostPort:
		h = e.HostPort
	}
	if h != "" {
		return h
	}
	return defaultHostPort
}
