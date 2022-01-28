// +build tools

package tools

import (
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/gogo/protobuf/protoc-gen-gogofast"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
