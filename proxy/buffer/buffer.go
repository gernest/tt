package buffer

import "github.com/valyala/bytebufferpool"

func Get() *bytebufferpool.ByteBuffer {
	return bytebufferpool.Get()
}

func Put(b *bytebufferpool.ByteBuffer) {
	bytebufferpool.Put(b)
}
