// This file presents an interface to SAPI problem-related types and functions.

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// A ProblemEntry represents a single coefficient in a problem to submit to a
// solver.  If I=J, the ProblemEntry represents a linear term.  Otherwise, it
// represents a quadratic term.
type ProblemEntry struct {
	I     int
	J     int
	Value float64
}

// A Problem is a list of ProblemEntry coefficients.
type Problem []ProblemEntry

// toC converts a Go Problem to a C sapi_Problem.
func (p Problem) toC() *C.sapi_Problem {
	// Convert each ProblemEntry in turn.
	cProblem := &C.sapi_Problem{}
	cProblem.len = C.size_t(len(p))
	elts := C.malloc(C.sizeof_sapi_ProblemEntry * cProblem.len)
	ePtr := (*[1 << 30]C.sapi_ProblemEntry)(elts)[:len(p):len(p)]
	for i, pe := range p {
		ePtr[i].i = C.int(pe.I)
		ePtr[i].j = C.int(pe.J)
		ePtr[i].value = C.double(pe.Value)
	}
	cProblem.elements = (*C.sapi_ProblemEntry)(elts)

	// Free the memory we allocated when the object is GC'd.
	runtime.SetFinalizer(cProblem, func(cp *C.sapi_Problem) {
		C.free(unsafe.Pointer(cp.elements))
		cp.elements = nil
	})
	return cProblem
}
