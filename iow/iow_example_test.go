package iow_test

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/akramarenkov/wrecker/iow"
)

func ExampleWrecker() {
	data := []byte("some data")

	buffer := bytes.NewBuffer(nil)

	opts := iow.Opts{
		ReadCallsLimit:  1,
		ReadSizeLimit:   2 * len(data),
		Underlying:      buffer,
		WriteCallsLimit: 3,
		WriteSizeLimit:  2 * len(data),
	}

	wrecker := iow.New(opts)

	_, err := wrecker.Write(data)
	fmt.Println("First write error is nil:", err == nil)
	fmt.Println(
		"Is buffer contains one data element:",
		slices.Equal(buffer.Bytes(), data),
	)
	fmt.Println()

	_, err = wrecker.Write(data)
	fmt.Println("Second write error is nil:", err == nil)
	fmt.Println(
		"Is buffer contains two data element:",
		slices.Equal(buffer.Bytes(),
			slices.Concat(data, data)),
	)
	fmt.Println()

	_, err = wrecker.Write(data)
	fmt.Println("Third write error is nil:", err == nil)
	fmt.Println(
		"Is buffer contains two data element:",
		slices.Equal(buffer.Bytes(),
			slices.Concat(data, data)),
	)
	fmt.Println()

	received := make([]byte, len(data))

	_, err = wrecker.Read(received)
	fmt.Println("First read error is nil:", err == nil)
	fmt.Println(
		"Is received data is equal to one data element:",
		slices.Equal(received, data),
	)
	fmt.Println()

	received2 := make([]byte, len(data))

	_, err = wrecker.Read(received2)
	fmt.Println("Second read error is nil:", err == nil)
	fmt.Println(
		"Is received data is equal to one data element:",
		slices.Equal(received2, data),
	)
	// Output:
	// First write error is nil: true
	// Is buffer contains one data element: true
	//
	// Second write error is nil: true
	// Is buffer contains two data element: true
	//
	// Third write error is nil: false
	// Is buffer contains two data element: true
	//
	// First read error is nil: true
	// Is received data is equal to one data element: true
	//
	// Second read error is nil: false
	// Is received data is equal to one data element: false
}
