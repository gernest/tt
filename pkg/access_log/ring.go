// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ring implements operations on circular lists.
package accesslog

import "context"

type RingOptions struct {
	InSize  int
	OutSize int
}

type Ring struct {
	In  <-chan *Entry
	Out chan *Entry
}

func (r *Ring) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-r.In:
			if !ok {
				// che channel was closed
				return
			}
			select {
			case r.Out <- e:
			default:
				e.Release()
			}
		}
	}
}
