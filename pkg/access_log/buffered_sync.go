package accesslog

type Batch []*Entry

type BufferedSync struct {
	batch Batch
}
