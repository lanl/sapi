// This file presents an interface to SAPI solver-related types and
// functions.

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// A Solver represents a SAPI solver.
type Solver struct {
	solver *C.sapi_Solver // SAPI solver object
	Name   string         // Solver name
	Conn   *Connection    // Connection with which this solver is associated
}

// GetSolver returns a solver associated with a given connection.
func (c *Connection) GetSolver(name string) (*Solver, error) {
	// Access a solver by name.
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	s := C.sapi_getSolver(c.conn, cName)
	if s == nil {
		return nil, fmt.Errorf("Solver %q not found on connection %s", name, c.URL)
	}
	solverObj := &Solver{
		solver: s,
		Name:   name,
		Conn:   c,
	}

	// Free the solver when it gets GC'd away.
	runtime.SetFinalizer(solverObj, func(s *Solver) {
		if s.solver != nil {
			C.sapi_freeSolver(s.solver)
			s.solver = nil
		}
	})
	return solverObj, nil
}

// SolverProperties represents a SAPI solver's properties.
type SolverProperties struct {
	props                 *C.sapi_SolverProperties // SAPI solver properties object
	SupportedProblemTypes []string                 // "qubo" and/or "ising"
	NumQubits             int                      // Total number of qubits, both working and non-working, in the processor
	Qubits                []int                    // Working qubit indices
	Couplers              [][2]int                 // Working couplers in the processor
}

// GetProperties returns the properties associated with a SAPI solver.
func (s *Solver) GetProperties() *SolverProperties {
	// Acquire the solver's properties.
	p := C.sapi_getSolverProperties(s.solver)

	// Convert the supported problem types from C to Go.
	var spts []string
	if p.supported_problem_types != nil {
		nSpts := p.supported_problem_types.len
		spts = make([]string, nSpts)
		sptsPtr := (*[1 << 30]*C.char)(unsafe.Pointer(p.supported_problem_types.elements))[:nSpts:nSpts]
		for i := range spts {
			spts[i] = C.GoString(sptsPtr[i])
		}
	}

	// Convert the quantum solver properties from C to Go.
	var numQubits int
	var qubits []int
	var couplers [][2]int
	if p.quantum_solver != nil {
		// Convert the qubit count from C to Go.
		numQubits = int(p.quantum_solver.num_qubits)

		// Convert the qubit list from C to Go.
		nq := p.quantum_solver.qubits_len
		qubits = make([]int, nq)
		qPtr := (*[1 << 30]C.int)(unsafe.Pointer(p.quantum_solver.qubits))[:nq:nq]
		for i := range qubits {
			qubits[i] = int(qPtr[i])
		}

		// Convert the coupler list from C to Go.
		nc := p.quantum_solver.couplers_len
		couplers = make([][2]int, nc)
		cPtr := (*[1 << 30]C.sapi_Coupler)(unsafe.Pointer(p.quantum_solver.couplers))[:nc:nc]
		for i := range couplers {
			couplers[i] = [2]int{
				int(cPtr[i].q1),
				int(cPtr[i].q2),
			}
		}
	}

	// Create and initialize a Go solvers properties object and return it.
	propObj := &SolverProperties{
		props: p,
		SupportedProblemTypes: spts,
		NumQubits:             numQubits,
		Qubits:                qubits,
		Couplers:              couplers,
	}
	return propObj
}

// convertIsingResultToGo is a helper function for SolveIsing and SolveQubo
// that converts the returned C.sapi_IsingResult structure to a Go-friendly
// format.
func convertIsingResultToGo(result *C.sapi_IsingResult) (IsingResult, error) {
	// Convert the resulting solutions from C to Go.
	ns := int(result.num_solutions)
	sl := int(result.solution_len)
	sPtr := (*[1 << 30]C.int)(unsafe.Pointer(result.solutions))[:ns*sl : ns*sl]
	solns := make([][]int8, ns)
	for i := range solns {
		solns[i] = make([]int8, sl)
		for j := range solns[i] {
			solns[i][j] = int8(sPtr[i*sl+j])
		}
	}

	// Convert the resulting energies from C to Go.
	ePtr := (*[1 << 30]C.double)(unsafe.Pointer(result.energies))[:ns:ns]
	energies := make([]float64, ns)
	for i, v := range ePtr {
		energies[i] = float64(v)
	}

	// Convert the resulting tallies from C to Go.
	oPtr := (*[1 << 30]C.int)(unsafe.Pointer(result.num_occurrences))[:ns:ns]
	occurs := make([]int, ns)
	for i, v := range oPtr {
		occurs[i] = int(v)
	}

	// Free the C data and return the Go result.
	C.sapi_freeIsingResult(result)
	ir := IsingResult{
		Solutions:   solns,
		Energies:    energies,
		Occurrences: occurs,
	}
	return ir, nil
}

// SolveIsing solves an Ising-model problem.
func (s *Solver) SolveIsing(p Problem, sp SolverParameters) (IsingResult, error) {
	prob := p.toC()
	params := sp.ToC()
	var result *C.sapi_IsingResult
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if C.sapi_solveIsing(s.solver, prob, params, &result, &cErr[0]) != C.SAPI_OK {
		return IsingResult{}, fmt.Errorf("%s", C.GoString(&cErr[0]))
	}
	return convertIsingResultToGo(result)
}

// SolveQubo solves a QUBO problem.
func (s *Solver) SolveQubo(p Problem, sp SolverParameters) (IsingResult, error) {
	prob := p.toC()
	params := sp.ToC()
	var result *C.sapi_IsingResult
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if C.sapi_solveQubo(s.solver, prob, params, &result, &cErr[0]) != C.SAPI_OK {
		return IsingResult{}, fmt.Errorf("%s", C.GoString(&cErr[0]))
	}
	return convertIsingResultToGo(result)
}
