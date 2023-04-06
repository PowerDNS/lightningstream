package snapshot

// Update wraps a Snapshot and NameInfo
type Update struct {
	Snapshot *Snapshot
	NameInfo NameInfo
	OnClose  func(u *Update)
}

func (u *Update) Close() {
	if u.OnClose != nil {
		u.OnClose(u)
	}
	u.OnClose = nil
	u.Snapshot = nil
}
