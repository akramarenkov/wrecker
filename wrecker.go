// Library with a Wrecker which corresponds to the io.ReadWriter interface and provides
// completes read and/or write operations with an error when reaching the limits on
// completed calls and/or the size of processed data.
package wrecker

import (
	"io"
)

// Options of the created Wrecker instance.
type Opts struct {
	// Error value that will be returned when reaching the limits. If not specified,
	// io.ErrUnexpectedEOF will be used
	Error error
	// Limit on the number of completed Read calls. An error will be returned when
	// attempting to make a ReadCallsLimit+1 call and on subsequent attempts. A
	// negative value indicates that there are no limit
	ReadCallsLimit int
	// Limit on the size of read data. An error will be returned when attempting to
	// read in total more than ReadSizeLimit data and on subsequent attempts. A
	// negative value indicates that there are no limit
	ReadSizeLimit int
	// Underlying io.ReadWriter whose corresponding methods will be called until the
	// limits are reached. May not be specified, in which case the Read/Write methods
	// will return a zero error and the amount of processed data equal to the amount
	// of input data
	ReadWriter io.ReadWriter
	// Limit on the number of completed Write calls. An error will be returned when
	// attempting to make a WriteCallsLimit+1 call and on subsequent attempts. A
	// negative value indicates that there are no limit
	WriteCallsLimit int
	// Limit on the size of write data. An error will be returned when attempting to
	// write in total more than WriteSizeLimit data and on subsequent attempts. A
	// negative value indicates that there are no limit
	WriteSizeLimit int
}

func (opts Opts) normalize() Opts {
	if opts.Error == nil {
		opts.Error = io.ErrUnexpectedEOF
	}

	return opts
}

type counters struct {
	completedCalls int
	processedSize  int
}

// Completes read and/or write operations with an error when reaching the limits on
// completed calls and/or the size of processed data.
type Wrecker struct {
	opts Opts

	read  counters
	write counters
}

// Creates Wrecker instance.
func New(opts Opts) *Wrecker {
	wrc := &Wrecker{
		opts: opts.normalize(),

		read: counters{
			completedCalls: opts.ReadCallsLimit,
			processedSize:  opts.ReadSizeLimit,
		},
		write: counters{
			completedCalls: opts.WriteCallsLimit,
			processedSize:  opts.WriteSizeLimit,
		},
	}

	return wrc
}

func (wrc *Wrecker) Read(data []byte) (int, error) {
	if wrc.readCallsLimitIsReached() {
		return 0, wrc.opts.Error
	}

	if wrc.readSizeLimitIsReached(data) {
		return 0, wrc.opts.Error
	}

	if wrc.opts.ReadWriter == nil {
		return len(data), nil
	}

	return wrc.opts.ReadWriter.Read(data)
}

func (wrc *Wrecker) readCallsLimitIsReached() bool {
	if wrc.opts.ReadCallsLimit < 0 {
		return false
	}

	if wrc.read.completedCalls == 0 {
		return true
	}

	wrc.read.completedCalls--

	return false
}

func (wrc *Wrecker) readSizeLimitIsReached(data []byte) bool {
	if wrc.opts.ReadSizeLimit < 0 {
		return false
	}

	if wrc.read.processedSize <= 0 {
		return true
	}

	// cannot be overflowed under these conditions
	wrc.read.processedSize -= len(data)

	return wrc.read.processedSize < 0
}

func (wrc *Wrecker) Write(data []byte) (int, error) {
	if wrc.writeCallsLimitIsReached() {
		return 0, wrc.opts.Error
	}

	if wrc.writeSizeLimitIsReached(data) {
		return 0, wrc.opts.Error
	}

	if wrc.opts.ReadWriter == nil {
		return len(data), nil
	}

	return wrc.opts.ReadWriter.Write(data)
}

func (wrc *Wrecker) writeCallsLimitIsReached() bool {
	if wrc.opts.WriteCallsLimit < 0 {
		return false
	}

	if wrc.write.completedCalls == 0 {
		return true
	}

	wrc.write.completedCalls--

	return false
}

func (wrc *Wrecker) writeSizeLimitIsReached(data []byte) bool {
	if wrc.opts.WriteSizeLimit < 0 {
		return false
	}

	if wrc.write.processedSize <= 0 {
		return true
	}

	// cannot be overflowed under these conditions
	wrc.write.processedSize -= len(data)

	return wrc.write.processedSize < 0
}
