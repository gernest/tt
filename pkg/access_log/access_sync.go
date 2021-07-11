package accesslog

type Sync interface {
	// Sync process entry. It is up to sync to make sure it call (*Entry).Release
	// when it is done to free up resources
	Sync(*Entry)
	sync()
}

type BatchSync interface {
	SyncBatch(Batch)
	syncBatch()
}

type BlackHole struct{}

func (BlackHole) sync()           {}
func (BlackHole) Sync(e *Entry)   { e.Release() }
func (BlackHole) Record(e *Entry) { e.Release() }
