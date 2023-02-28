package syncer

type Options struct {
	// ReceiveOnly prevents writing snapshots, we will only receive them
	ReceiveOnly bool
}
