// Package sapi provides a Go interface to D-Wave's SAPI library.
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
