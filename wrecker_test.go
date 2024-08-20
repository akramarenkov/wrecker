package wrecker

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	erroneousIterationsExcess = 10
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

	erroneousIterations := unerringIterations + erroneousIterationsExcess
	require.Positive(t, erroneousIterations, "calls limit: %v", callsLimit)

	opts := Opts{
		Error:           io.ErrClosedPipe,
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

func testWreckerSize(t *testing.T, blockSize int, readSizeLimit int) {
	readUnerringIterations := readSizeLimit / blockSize
	require.GreaterOrEqual(
		t,
		readUnerringIterations,
		0,
		"block size: %v, read size limit: %v",
		blockSize,
		readSizeLimit,
	)

	readErroneousIterations := readUnerringIterations + erroneousIterationsExcess
	require.Positive(
		t,
		readErroneousIterations,
		"block size: %v, read size limit: %v",
		blockSize,
		readSizeLimit,
	)

	// to make sure that read errors and inequality of the read data block to the
	// written data block are caused by Wrecker
	writeSizeLimit := readErroneousIterations * blockSize
	writeUnerringIterations := readErroneousIterations
	writeErroneousIterations := writeUnerringIterations + erroneousIterationsExcess

	opts := Opts{
		Error:           io.ErrClosedPipe,
		ReadCallsLimit:  -1,
		ReadSizeLimit:   readSizeLimit,
		ReadWriter:      bytes.NewBuffer(nil),
		WriteCallsLimit: -1,
		WriteSizeLimit:  writeSizeLimit,
	}

	wrecker := New(opts)

	writeBlock := make([]byte, blockSize)

	for id := range writeBlock {
		writeBlock[id] = 1
	}

	for range writeUnerringIterations {
		_, err := wrecker.Write(writeBlock)
		require.NoError(
			t,
			err,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)
	}

	for range writeErroneousIterations {
		_, err := wrecker.Write(writeBlock)
		require.Error(
			t,
			err,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)

		require.Equal(
			t,
			opts.Error,
			err,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)
	}

	for range readUnerringIterations {
		readBlock := make([]byte, blockSize)

		_, err := wrecker.Read(readBlock)
		require.NoError(
			t,
			err,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)

		require.Equal(
			t,
			writeBlock,
			readBlock,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)
	}

	for range readErroneousIterations {
		readBlock := make([]byte, blockSize)

		_, err := wrecker.Read(readBlock)
		require.Error(
			t,
			err,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)

		require.Equal(
			t,
			opts.Error,
			err,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)

		require.NotEqual(
			t,
			writeBlock,
			readBlock,
			"block size: %v, read size limit: %v",
			blockSize,
			readSizeLimit,
		)
	}
}

func TestWreckerUnspecifiedError(t *testing.T) {
	wrecker := New(Opts{})

	_, err := wrecker.Write(nil)
	require.Error(t, err)
	require.Equal(t, io.ErrUnexpectedEOF, err)

	_, err = wrecker.Read(nil)
	require.Error(t, err)
	require.Equal(t, io.ErrUnexpectedEOF, err)
}
