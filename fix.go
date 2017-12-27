// This file presents an interface to SAPI functions for simplifying
// optimization problems.

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"unsafe"
)

// FixVariablesMethod specifies how to identify values with a fixed value in
// all optimal solutions.
type FixVariablesMethod int

// These are the values a FixVariablesMethod accepts.
const (
	FixVariablesMethodOptimized FixVariablesMethod = C.SAPI_FIX_VARIABLES_METHOD_OPTIMIZED // Use both roof-duality and strongly connected components
	FixVariablesMethodStandard                     = C.SAPI_FIX_VARIABLES_METHOD_STANDARD  // Uses only roof duality
)

// A FixVariablesResult identifies variables that can be removed from a problem
// because their value is known a priori.
type FixVariablesResult struct {
	FixedVars  map[int]int8 // Map from a variable to its value
	Offset     float64      // Energy difference between the new and original problems
	NewProblem Problem      // Simplified problem, containing no fixed variables
}

// FixVariables identifies variables in a QUBO problem that have a fixed value
// in all optimal solutions and can therefore be elided from the problem that
// gets submitted to the solver.
func (p Problem) FixVariables(m FixVariablesMethod) (FixVariablesResult, error) {
	// Invoke the C function.
	cProb := p.toC()
	cMethod := C.sapi_FixVariablesMethod(m)
	var cResult *C.sapi_FixVariablesResult
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_fixVariables(cProb, cMethod, &cResult, &cErr[0]); ret != C.SAPI_OK {
		return FixVariablesResult{}, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}

	// Convert the result from C to Go.
	var fvr FixVariablesResult
	nf := int(cResult.fixed_variables_len)
	fvr.FixedVars = make(map[int]int8, nf)
	fPtr := (*[1 << 30]C.sapi_FixedVariable)(unsafe.Pointer(cResult.fixed_variables))[:nf:nf]
	for _, fv := range fPtr {
		fvr.FixedVars[int(fv._var)] = int8(fv.value)
	}
	fvr.Offset = float64(cResult.offset)
	fvr.NewProblem = problemFromC(&cResult.new_problem)

	// We no longer need the C version of the result.
	C.sapi_freeFixVariablesResult(cResult)
	return fvr, nil
}
