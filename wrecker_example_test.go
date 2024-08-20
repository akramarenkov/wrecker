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

	if _, err := wrecker.Write(data); err != nil {
		fmt.Println(err)
	}

	fmt.Println(buffer.String() == string(data))

	if _, err := wrecker.Write(data); err != nil {
		fmt.Println(err)
	}

	fmt.Println(buffer.String() == string(data)+string(data))

	if _, err := wrecker.Write(data); err != nil {
		fmt.Println(err)
	}

	fmt.Println(buffer.String() == string(data)+string(data))

	payload := make([]byte, len(data))

	if _, err := wrecker.Read(payload); err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(payload) == string(data))

	payload = make([]byte, len(data))

	if _, err := wrecker.Read(payload); err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(payload) == string(data))

	// Output:
	// true
	// true
	// limit is reached
	// true
	// true
	// limit is reached
	// false
}
