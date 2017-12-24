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

// problemFromC converts a C sapi_Problem to a Go Problem.
func problemFromC(csp *C.sapi_Problem) Problem {
	npe := int(csp.len)
	prob := make(Problem, npe)
	pePtr := (*[1 << 30]C.sapi_ProblemEntry)(unsafe.Pointer(csp.elements))[:npe:npe]
	for i, pe := range pePtr {
		prob[i] = ProblemEntry{
			I:     int(pe.i),
			J:     int(pe.j),
			Value: float64(pe.value),
		}
	}
	return prob
}

// countQubits returns a tally of the number of unique qubits referenced by a
// Problem.
func (p Problem) countQubits() int {
	seen := make(map[int]struct{}, len(p))
	for _, pe := range p {
		seen[pe.I] = struct{}{}
		seen[pe.J] = struct{}{}
	}
	return len(seen)
}

// ChimeraAdjacency constructs the adjacency matrix for an arbitrary Chimera
// graph.
func ChimeraAdjacency(m, n, l int) (Problem, error) {
	var cProb *C.sapi_Problem
	if ret := C.sapi_getChimeraAdjacency(C.int(m), C.int(n), C.int(l), &cProb); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "Failed to construct a {%d, %d, %d} Chimera graph", m, n, l)
	}
	defer C.sapi_freeProblem(cProb)
	return problemFromC(cProb), nil
}
