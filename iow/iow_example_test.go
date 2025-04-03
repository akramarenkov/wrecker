package iow_test

import (
	"bytes"
	"errors"
	"fmt"
	"slices"

	"github.com/akramarenkov/wrecker/iow"
)

var ErrLimitReached = errors.New("limit is reached")

func ExampleWrecker() {
	data := []byte("some data")

	buffer := bytes.NewBuffer(nil)

	opts := iow.Opts{
		Error:           ErrLimitReached,
		ReadCallsLimit:  1,
		ReadSizeLimit:   2 * len(data),
		Underlying:      buffer,
		WriteCallsLimit: 3,
		WriteSizeLimit:  2 * len(data),
	}

	wrecker := iow.New(opts)

	_, err := wrecker.Write(data)
	fmt.Println(err)
	fmt.Println(slices.Equal(buffer.Bytes(), data))
	fmt.Println()

	_, err = wrecker.Write(data)
	fmt.Println(err)
	fmt.Println(slices.Equal(buffer.Bytes(), slices.Concat(data, data)))
	fmt.Println()

	_, err = wrecker.Write(data)
	fmt.Println(err)
	fmt.Println(slices.Equal(buffer.Bytes(), slices.Concat(data, data)))
	fmt.Println()

	received := make([]byte, len(data))

	_, err = wrecker.Read(received)
	fmt.Println(err)
	fmt.Println(slices.Equal(received, data))
	fmt.Println()

	received2 := make([]byte, len(data))

	_, err = wrecker.Read(received2)
	fmt.Println(err)
	fmt.Println(slices.Equal(received2, data))
	// Output:
	// <nil>
	// true
	//
	// <nil>
	// true
	//
	// limit is reached
	// true
	//
	// <nil>
	// true
	//
	// limit is reached
	// false
}
