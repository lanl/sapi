/*
Package sapi provides a Go interface to D-Wave's SAPI library.

Consider this very much a work in progress.  At the moment, it exposes
only a small subset of SAPI types and functions.  The intention is to
add more functionality as the need arises.
*/
package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"fmt"
	"unsafe"
)

// init initializes SAPI.
func init() {
	if C.sapi_globalInit() != C.SAPI_OK {
		panic("Failed to initialize SAPI")
	}
}

// Version returns the SAPI version number as a string.
func Version() string {
	return C.GoString(C.sapi_version())
}

// Code represents a SAPI error code
type Code int

// These are the SAPI error codes known at the time of this writing.
const (
	OK                  Code = C.SAPI_OK
	InvalidParameter         = C.SAPI_ERR_INVALID_PARAMETER
	SolveFailed              = C.SAPI_ERR_SOLVE_FAILED
	AuthenticationError      = C.SAPI_ERR_AUTHENTICATION
	NetworkError             = C.SAPI_ERR_NETWORK
	CommunicationError       = C.SAPI_ERR_COMMUNICATION
	AsyncNotDone             = C.SAPI_ERR_ASYNC_NOT_DONE
	ProblemCanceled          = C.SAPI_ERR_PROBLEM_CANCELLED
	NotInitialized           = C.SAPI_ERR_NO_INIT
	OutOfMemory              = C.SAPI_ERR_OUT_OF_MEMORY
)

// An Error encapsulates a SAPI code and its string representation.
type Error struct {
	N Code   // Numerical representation
	S string // Textual representation
}

// Error returns the textual representation of an Error.
func (e Error) Error() string {
	return e.S
}

// newErrorf creates a new Error struct from a SAPI return code and error
// string.
func newErrorf(c C.sapi_Code, format string, a ...interface{}) Error {
	return Error{
		N: Code(c),
		S: fmt.Sprintf(format, a),
	}
}

// cIntsToGo converts a C array of ints to a Go slice.
func cIntsToGo(cArray *C.int, n int) []int {
	a := make([]int, n)
	cPtr := (*[1 << 30]C.int)(unsafe.Pointer(cArray))[:n:n]
	for i, v := range cPtr {
		a[i] = int(v)
	}
	return a
}

// goIntsToC converts a Go slice of ints to a C array of ints.
func goIntsToC(xs []int) *C.int {
	n := C.size_t(len(xs))
	elts := C.malloc(C.sizeof_int * n)
	ePtr := (*[1 << 30]C.int)(elts)[:n:n]
	for i, x := range xs {
		ePtr[i] = C.int(x)
	}
	return (*C.int)(elts)
}

// cStringsToGo converts a C array of strings to a Go slice.
func cStringsToGo(cArray **C.char, n int) []string {
	a := make([]string, n)
	cPtr := (*[1 << 30]*C.char)(unsafe.Pointer(cArray))[:n:n]
	for i, v := range cPtr {
		a[i] = C.GoString(v)
	}
	return a
}

// cInt8MatrixToGo converts a flattened 2-D matrix of C ints to a Go slice of
// slices of int8s.
func cInt8MatrixToGo(cArray *C.int, nr, nc int) [][]int8 {
	aPtr := (*[1 << 30]C.int)(unsafe.Pointer(cArray))[:nr*nc : nr*nc]
	array := make([][]int8, nr)
	for i := range array {
		array[i] = make([]int8, nc)
		for j := range array[i] {
			array[i][j] = int8(aPtr[i*nc+j])
		}
	}
	return array
}

// int8MatrixtoC converts a Go slice of slices of int8s to a flattened 2-D
// matrix of C ints.
func int8MatrixtoC(array [][]int8) *C.int {
	nr := len(array)
	nc := len(array[0])
	cArray := C.malloc(C.sizeof_int * C.size_t(nr*nc))
	aPtr := (*[1 << 30]C.int)(unsafe.Pointer(cArray))[:nr*nc : nr*nc]
	i := 0
	for _, row := range array {
		for _, v := range row {
			aPtr[i] = C.int(v)
			i++
		}
	}
	return (*C.int)(cArray)
}
