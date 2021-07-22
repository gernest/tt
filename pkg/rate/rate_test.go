package rate

import (
	"testing"
	"time"
)

func TestRate(t *testing.T) {
	t.Run("set restriction", func(t *testing.T) {
		limit := 15
		r, err := New(
			t.TempDir(),
			time.Second,
			uint32(limit),
			100,
			100,
		)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		key := []byte("foo")
		for i := 0; i < limit; i++ {
			if err := r.Take(key); err != nil {
				t.Fatalf("limit should not have been reached step:%d error: %v", i, err)
			}
		}
		err = r.Take(key)
		if !IsForbidden(err) {
			t.Errorf("expected %v got %v", ErrLimited, err)
		}
	})
}
