package pubsub

import (
	"fmt"
	"github.com/klauspost/compress/zstd"
)

// compactor is an instance of zstandard compression algorithm
type compactor struct {
	zEncodr *zstd.Encoder
	zDecodr *zstd.Decoder
}

func newCompactor() (*compactor, error) {
	zstdEncoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, fmt.Errorf(`creating zstd encoder failed - %v`, err)
	}

	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf(`creating zstd decoder failed - %v`, err)
	}

	return &compactor{zEncodr: zstdEncoder, zDecodr: zstdDecoder}, nil
}
