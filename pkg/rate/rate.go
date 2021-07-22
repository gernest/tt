// Package rate is an implementation of sliding window rate limiter based on
// badger db
package rate

import (
	"errors"
	"time"

	"github.com/dgraph-io/badger/v3"
)

var ErrLimited = errors.New("rate: limited")

func IsForbidden(err error) bool {
	return errors.Is(err, ErrLimited)
}

type Rate struct {
	db     *badger.DB
	window time.Duration
	limit  uint32
	span   uint32
}

func (r *Rate) Close() error {
	return r.db.Close()
}

func New(path string,
	window time.Duration,
	limit uint32,
	numberOfBuckets uint32,
	versions int,
) (*Rate, error) {
	o := badger.DefaultOptions(path)
	o.Logger = nil
	o.NumVersionsToKeep = versions
	db, err := badger.Open(o)
	if err != nil {
		return nil, err
	}
	windowSec := uint32(window.Seconds())
	span := windowSec / numberOfBuckets
	if windowSec%numberOfBuckets > 0 {
		span++
	}
	return &Rate{
		db:     db,
		span:   span,
		limit:  limit,
		window: window,
	}, nil
}

func (r *Rate) TakeAt(ts time.Time, key []byte) error {
	return r.db.Update(func(txn *badger.Txn) error {
		now := timestamp(ts)
		it := txn.NewKeyIterator(key, badger.IteratorOptions{
			AllVersions: true,
		})
		defer it.Close()
		limit := r.limit
		var total uint32
		var lastTs uint64
		for it.Rewind(); it.Valid(); it.Next() {
			e := it.Item()
			if e.IsDeletedOrExpired() {
				continue
			}
			total += 1
			lastTs = e.ExpiresAt()
		}
		if total < limit {
			// limit has not been reached yet
			{
				// normalize buckets
				e := badger.NewEntry(key, []byte{})
				e.ExpiresAt = now - (now % uint64(r.span))
				if err := txn.SetEntry(e); err != nil {
					return err
				}
			}
			e := badger.NewEntry(key, []byte{})
			e.ExpiresAt = timestamp(ts.Add(r.window))
			return txn.SetEntry(e)
		}
		ets := toTime(lastTs).Sub(ts)
		e := badger.NewEntry(key, []byte{})
		e.ExpiresAt = timestamp(ts.Add(ets))
		if err := txn.SetEntry(e); err != nil {
			return err
		}
		return ErrLimited
	})
}

func timestamp(ts time.Time) uint64 {
	return uint64(ts.Unix())
}

func toTime(ts uint64) time.Time {
	return time.Unix(int64(ts), 0)
}

// Take returns nil if key hasn't exceeded its limit
func (r *Rate) Take(key []byte) error {
	return r.TakeAt(time.Now(), key)
}
