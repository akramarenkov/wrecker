// Library with a Wrecker which corresponds to the io.ReadWriter interface and provides
// completes read and/or write operations with an error after reaching the limits on
// completed calls and/or the size of processed data.
package wrecker

import (
	"io"

	"github.com/akramarenkov/wrecker/internal/limiter"
)

// Options of the created Wrecker instance.
type Opts struct {
	// Error value that will be returned after reaching the limits
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
	// limits are reached
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

// Completes read and/or write operations with an error after reaching the limits on
// completed calls and/or the size of processed data.
type Wrecker struct {
	opts Opts

	read  *limiter.Limiter
	write *limiter.Limiter
}

// Creates Wrecker instance.
func New(opts Opts) *Wrecker {
	wrc := &Wrecker{
		opts: opts,

		read:  limiter.New(opts.ReadCallsLimit, opts.ReadSizeLimit),
		write: limiter.New(opts.WriteCallsLimit, opts.WriteSizeLimit),
	}

	return wrc
}

func (wrc *Wrecker) Read(data []byte) (int, error) {
	if wrc.read.IsReached(len(data)) {
		return 0, wrc.opts.Error
	}

	if wrc.opts.ReadWriter == nil {
		return len(data), nil
	}

	return wrc.opts.ReadWriter.Read(data)
}

func (wrc *Wrecker) Write(data []byte) (int, error) {
	if wrc.write.IsReached(len(data)) {
		return 0, wrc.opts.Error
	}

	if wrc.opts.ReadWriter == nil {
		return len(data), nil
	}

	return wrc.opts.ReadWriter.Write(data)
}
