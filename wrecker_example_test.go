package wrecker_test

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/akramarenkov/wrecker"
)

var (
	ErrLimitReached = errors.New("limit is reached")
)

func ExampleWrecker() {
	data := []byte("some data")

	buffer := bytes.NewBuffer(nil)

	opts := wrecker.Opts{
		Error:           ErrLimitReached,
		ReadCallsLimit:  1,
		ReadSizeLimit:   2 * len(data),
		ReadWriter:      buffer,
		WriteCallsLimit: 3,
		WriteSizeLimit:  2 * len(data),
	}

	wrecker := wrecker.New(opts)

	_, err := wrecker.Write(data)
	fmt.Println(err)
	fmt.Println(buffer.String() == string(data))
	fmt.Println()

	_, err = wrecker.Write(data)
	fmt.Println(err)
	fmt.Println(buffer.String() == string(data)+string(data))
	fmt.Println()

	_, err = wrecker.Write(data)
	fmt.Println(err)
	fmt.Println(buffer.String() == string(data)+string(data))
	fmt.Println()

	payload := make([]byte, len(data))

	_, err = wrecker.Read(payload)
	fmt.Println(err)
	fmt.Println(string(payload) == string(data))
	fmt.Println()

	payload = make([]byte, len(data))

	_, err = wrecker.Read(payload)
	fmt.Println(err)
	fmt.Println(string(payload) == string(data))

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
