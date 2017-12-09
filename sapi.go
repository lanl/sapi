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
