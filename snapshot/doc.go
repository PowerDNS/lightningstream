/*
Package snapshot implements the snapshot protobuf serialisation code.

It implements custom serialisation and deserialisation code for performance.

## Background

Originally, this used gogo-protobuf to generate the serialisation code,
but that ended up allocating a []KV slice for all DBI entries, with inside
the KV a []byte for both key and value. Since one slices takes up 24 byte
just for its header, that is 48 bytes of overhead per entry.

For 6 million entries, that is theoretically 300 MB of overhead.

What we actually saw were allocations of over 2.7 GB in just the DBI code
under the following circumstances:

- main was 26 MB compressed, 175 MB uncompressed.
- shard was 34 GB compressed, 265 MB uncompressed.

In total that is 440 MB data uncompressed. It turns out that half of the
allocation was used by the code copying all key and value bytes.

	     flat  flat%   sum%        cum   cum%
	2047.55MB 56.26% 56.26%  2722.09MB 74.80%  github.com/PowerDNS/lightningstream/snapshot.(*DBI).Unmarshal
	 674.54MB 18.54% 98.45%   674.54MB 18.54%  github.com/PowerDNS/lightningstream/snapshot.(*KV).Unmarshal

Patching the generated code to not copy the data reduced the total memory use to
1.4 GB:

	1378.77MB 52.20% 52.20%  1378.77MB 52.20%  github.com/PowerDNS/lightningstream/snapshot.(*DBI).Unmarshal

That is still significantly more than the 440 MB we would expect. Part of it
is likely because the allocated slices are up to 2x larger than needed with
the append() growth algorithm, the rest is probably memory allocator overhead.

The solution to this was to stream the data instead of loading the full protobuf
into slices.

## Other protobuf options

At the moment of writing the status of Go protobuf libraries was as follows:

- GoGo protobuf was no longer maintained and deprecated
- Google protobuf insisted on deserializing into []*KV
- VTProtobuf used Google protobuf to generate the struct, thus ending up with []*KV
- CSProto mostly wrapped the above for easy use of mixed code in gRPC

The only library that supported the use of a custom Go type to handle DBIs
lazily was GoGo, but it was deprecated, and support for this did appear to have
many caveats.

A potential workaround was to use a standard lib for the outer protobuf, but then
define `repeated bytes databases = 3` instead of `repeated DBI databases = 3`, and
deserialize the entries on demand, since bytes and sub-messages have the same
wire format.

In the end we decided to just use custom code for all.
*/
package snapshot
