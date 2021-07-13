package cluster

import (
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
)

var _ badger.Logger = (*Badger)(nil)

type Badger struct {
	*zap.SugaredLogger
}

func (b *Badger) Warningf(msg string, args ...interface{}) {
	b.Warnf(msg, args...)
}
