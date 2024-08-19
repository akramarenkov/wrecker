// Calculates the reaching of limits on the number of completed calls and the size
// of processed data.
package limiter

import "math"

// Calculates the reaching of limits on the number of completed calls and the size
// of processed data.
type Limiter struct {
	// Limit on the number of completed calls. A positive conclusion about reaching the
	// limit will be returned when attempting to make a callsLimit+1 call and on
	// subsequent attempts. A negative value indicates that there are no limit
	callsLimit int
	// Limit on the size of processed data. A positive conclusion about reaching the
	// limit will be returned when attempting to process in total more than sizeLimit
	// data and on subsequent attempts. A negative value indicates that there are no
	// limit
	sizeLimit int

	// Completed calls
	calls int
	// Size of processed data
	size int
}

// Creates Limiter instance.
//
// callsLimit - limit on the number of completed calls. A positive conclusion about
// reaching the limit will be returned when attempting to make a callsLimit+1 call and
// on subsequent attempts. A negative value indicates that there are no limit.
//
// sizeLimit - limit on the size of processed data. A positive conclusion about
// reaching the limit will be returned when attempting to process more than sizeLimit
// data and on subsequent attempts. A negative value indicates that there are no limit.
func New(callsLimit int, sizeLimit int) *Limiter {
	lmt := &Limiter{
		callsLimit: callsLimit,
		sizeLimit:  sizeLimit,
	}

	return lmt
}

// Returns a conclusion about reaching the limits on the number of completed calls
// and/or the size of processed data.
func (lmt *Limiter) IsReached(size int) bool {
	if lmt.isCallsReached() {
		return true
	}

	if lmt.sizeLimit < 0 {
		return false
	}

	if lmt.size >= lmt.sizeLimit {
		return true
	}

	increased := lmt.size + size

	// increased overflowed, size value is set to the maximum for the int type to return
	// true on subsequent calls
	if increased < lmt.size {
		lmt.size = math.MaxInt
		return true
	}

	// size is increased for checks on subsequent calls
	lmt.size = increased

	return lmt.size > lmt.sizeLimit
}

func (lmt *Limiter) isCallsReached() bool {
	if lmt.callsLimit < 0 {
		return false
	}

	if lmt.calls == lmt.callsLimit {
		return true
	}

	lmt.calls++

	return false
}
