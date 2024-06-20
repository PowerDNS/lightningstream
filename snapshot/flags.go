package snapshot

import "github.com/PowerDNS/lightningstream/lmdbenv/header"

func (kv *KV) MaskedFlags() header.Flags {
	return header.Flags(kv.Flags).Masked()
}
