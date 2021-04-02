package proxy

import "testing"

func TestSpeed(t *testing.T) {
	sample := []struct {
		s string
		g int64
	}{
		{"120kib/s", 122880},
	}
	for _, v := range sample {
		t.Run(v.s, func(t *testing.T) {
			l, err := Speed(v.s).Limit()
			if err != nil {
				t.Fatal(err)
			}
			n := int64(l)
			if n != v.g {
				t.Errorf("expected %d got %d", n, v.g)
			}
		})
	}
}
