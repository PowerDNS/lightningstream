package snapshot

import "powerdns.com/platform/lightningstream/lmdbenv/header"

func (kv *KV) MaskedFlags() header.Flags {
	return header.Flags(kv.Flags).Masked()
}
