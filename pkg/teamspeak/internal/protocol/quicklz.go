package protocol

import (
	"errors"
	"sync"

	"github.com/Hiroko103/go-quicklz"
)

// QuickLZDecompressLevel1 uses github.com/Hiroko103/go-quicklz (GPL-3.0).
func QuickLZDecompressLevel1(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("quicklz: empty input")
	}
	qlz, err := getQuickLZ()
	if err != nil {
		return nil, err
	}
	size := quicklz.Size_decompressed(&src)
	if size <= 0 {
		return nil, errors.New("quicklz: invalid decompressed size")
	}
	dst := make([]byte, size)
	n, err := qlz.Decompress(&src, &dst)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}

var (
	qlzOnce sync.Once
	qlzInst *quicklz.Qlz
	qlzErr  error
)

func getQuickLZ() (*quicklz.Qlz, error) {
	qlzOnce.Do(func() {
		qlzInst, qlzErr = quicklz.New(quicklz.COMPRESSION_LEVEL_1, quicklz.STREAMING_BUFFER_0)
	})
	return qlzInst, qlzErr
}
