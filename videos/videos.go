//go:generate go run github.com/shuLhan/go-bindata/cmd/go-bindata -pkg videos -nocompress src/...
package videos

import "bytes"

func Globe() *bytes.Reader {
	return bytes.NewReader(MustAsset("src/globe.ivf"))
}

func Pion() *bytes.Reader {
	return bytes.NewReader(MustAsset("src/pion.ivf"))
}
