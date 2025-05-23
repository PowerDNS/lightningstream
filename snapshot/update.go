package snapshot

import "github.com/c2h5oh/datasize"

// Update wraps a Snapshot and NameInfo
type Update struct {
	Snapshot *Snapshot
	NameInfo NameInfo
	BlobSize datasize.ByteSize
	OnClose  func(u *Update)
}

func (u *Update) Close() {
	if u.OnClose != nil {
		u.OnClose(u)
	}
	u.OnClose = nil
	u.Snapshot = nil
}
