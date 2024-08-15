package wrecker

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	erroneousIterationsExcess = 3
)

func TestWrecker(t *testing.T) {
	for limit := range 1 << 8 {
		testWreckerCalls(t, limit)
	}

	for limit := range 1 << 14 {
		testWreckerSize(t, 1024, limit)
	}
}

func testWreckerCalls(t *testing.T, callsLimit int) {
	unerringIterations := callsLimit
	require.GreaterOrEqual(t, unerringIterations, 0, "calls limit: %v", callsLimit)

	erroneousIterations := erroneousIterationsExcess * (unerringIterations + 1)
	require.Positive(t, erroneousIterations, "calls limit: %v", callsLimit)

	opts := Opts{
		Error:           io.ErrUnexpectedEOF,
		ReadCallsLimit:  callsLimit,
		ReadSizeLimit:   -1,
		WriteCallsLimit: callsLimit,
		WriteSizeLimit:  -1,
	}

	wrecker := New(opts)

	for range unerringIterations {
		_, err := wrecker.Write(nil)
		require.NoError(t, err, "calls limit: %v", callsLimit)
	}

	for range erroneousIterations {
		_, err := wrecker.Write(nil)
		require.Error(t, err, "calls limit: %v", callsLimit)
		require.Equal(t, opts.Error, err, "calls limit: %v", callsLimit)
	}

	for range unerringIterations {
		_, err := wrecker.Read(nil)
		require.NoError(t, err, "calls limit: %v", callsLimit)
	}

	for range erroneousIterations {
		_, err := wrecker.Read(nil)
		require.Error(t, err, "calls limit: %v", callsLimit)
		require.Equal(t, opts.Error, err, "calls limit: %v", callsLimit)
	}
}

func testWreckerSize(t *testing.T, blockSize int, sizeLimit int) {
	unerringIterations := sizeLimit / blockSize
	require.GreaterOrEqual(
		t,
		unerringIterations,
		0,
		"block size: %v, size limit: %v",
		blockSize,
		sizeLimit,
	)

	erroneousIterations := erroneousIterationsExcess * (unerringIterations + 1)
	require.Positive(
		t,
		erroneousIterations,
		"block size: %v, size limit: %v",
		blockSize,
		sizeLimit,
	)

	opts := Opts{
		Error:           io.ErrUnexpectedEOF,
		ReadCallsLimit:  -1,
		ReadSizeLimit:   sizeLimit,
		ReadWriter:      bytes.NewBuffer(nil),
		WriteCallsLimit: -1,
		WriteSizeLimit:  sizeLimit,
	}

	wrecker := New(opts)

	writeBlock := make([]byte, blockSize)
	writeBlock[0] = 1
	writeBlock[len(writeBlock)-1] = 1

	for range unerringIterations {
		_, err := wrecker.Write(writeBlock)
		require.NoError(
			t,
			err,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)
	}

	for range erroneousIterations {
		_, err := wrecker.Write(writeBlock)
		require.Error(
			t,
			err,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)

		require.Equal(
			t,
			opts.Error,
			err,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)
	}

	for range unerringIterations {
		readBlock := make([]byte, blockSize)

		_, err := wrecker.Read(readBlock)
		require.NoError(
			t,
			err,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)

		require.Equal(
			t,
			writeBlock,
			readBlock,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)
	}

	for range erroneousIterations {
		readBlock := make([]byte, blockSize)

		_, err := wrecker.Read(readBlock)
		require.Error(
			t,
			err,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)

		require.Equal(
			t,
			opts.Error,
			err,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)

		require.NotEqual(
			t,
			writeBlock,
			readBlock,
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)
	}
}
