// This file presents an interface to SAPI solver-related types and
// functions.

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
		return nil, newErrorf(C.SAPI_ERR_INVALID_PARAMETER, "Solver %q not found on connection %s", name, c.URL)
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

// An IsingRangeProperties indicates the acceptable ranges of h and J
// coefficients.
type IsingRangeProperties struct {
	HMin float64
	HMax float64
	JMin float64
	JMax float64
}

// A QuantumSolverProperties records the available qubits and couplers.
type QuantumSolverProperties struct {
	NumQubits int      // Total number of qubits, both working and non-working, in the processor
	Qubits    []int    // Working qubit indices
	Couplers  [][2]int // Working couplers in the processor
}

// An AnnealOffsetRange indicates the minimum and maximum values a specific
// anneal offset can accept.
type AnnealOffsetRange [2]float64

// An AnnealOffsetProperties encapsulates properties of per-qubit annealing
// offsets.
type AnnealOffsetProperties struct {
	Ranges   []AnnealOffsetRange // Ranges of valid anneal offset values, in normalized offset units, for each qubit
	Step     float64             // Quantization step size of anneal offset values in normalized units
	StepPhi0 float64             // Quantization step size in physical units (annealing flux bias units)
}

// SolverProperties represents a SAPI solver's properties.
type SolverProperties struct {
	props                 *C.sapi_SolverProperties // SAPI solver properties object
	SupportedProblemTypes []string                 // "qubo" and/or "ising"
	IsingRanges           *IsingRangeProperties    // Range of h and J coefficients
	QuantumProps          *QuantumSolverProperties // Properties of the quantum solver
	AnnealOffsets         *AnnealOffsetProperties  // Properties of the per-qubit annealing offsets
	Parameters            []string                 // Valid solver parameter names, sorted in ascending order
}

// convertQSPs converts quantum solver properties from C to Go.
func convertQSPs(p *C.sapi_SolverProperties) *QuantumSolverProperties {
	// Do nothing if we have nothing to do.
	qs := p.quantum_solver
	if qs == nil {
		return nil
	}

	// Convert the qubit count from C to Go.
	numQubits := int(qs.num_qubits)

	// Convert the qubit list from C to Go.
	qubits := cIntsToGo(qs.qubits, int(qs.qubits_len))

	// Convert the coupler list from C to Go.
	nc := qs.couplers_len
	couplers := make([][2]int, nc)
	cPtr := (*[1 << 30]C.sapi_Coupler)(unsafe.Pointer(qs.couplers))[:nc:nc]
	for i := range couplers {
		couplers[i] = [2]int{
			int(cPtr[i].q1),
			int(cPtr[i].q2),
		}
	}

	// Store all of the above in the qProps struct.
	return &QuantumSolverProperties{
		NumQubits: numQubits,
		Qubits:    qubits,
		Couplers:  couplers,
	}
}

// convertAOPs converts annealing offset properties from C to Go.
func convertAOPs(p *C.sapi_SolverProperties) *AnnealOffsetProperties {
	// Do nothing if we have nothing to do.
	ao := p.anneal_offset
	if ao == nil {
		return nil
	}

	// Convert the anneal offset ranges.
	nr := int(ao.ranges_len)
	ranges := make([]AnnealOffsetRange, nr)
	rPtr := (*[1 << 30]C.sapi_AnnealOffsetRange)(unsafe.Pointer(ao.ranges))[:nr:nr]
	for i, r := range rPtr {
		ranges[i] = [2]float64{float64(r.min), float64(r.max)}
	}

	// Return the set of properties.
	return &AnnealOffsetProperties{
		Ranges:   ranges,
		Step:     float64(ao.step),
		StepPhi0: float64(ao.step_phi0),
	}
}

// GetProperties returns the properties associated with a SAPI solver.
func (s *Solver) GetProperties() *SolverProperties {
	// Acquire the solver's properties.
	p := C.sapi_getSolverProperties(s.solver)

	// Convert the supported problem types from C to Go.
	var spts []string
	if p.supported_problem_types != nil {
		spts = cStringsToGo(p.supported_problem_types.elements, int(p.supported_problem_types.len))
	}

	// Convert the Ising ranges from C to Go.
	var ranges *IsingRangeProperties
	if p.ising_ranges != nil {
		ranges = &IsingRangeProperties{
			HMin: float64(p.ising_ranges.h_min),
			HMax: float64(p.ising_ranges.h_max),
			JMin: float64(p.ising_ranges.j_min),
			JMax: float64(p.ising_ranges.j_max),
		}
	}

	// Convert the valid solver parameter names from C to Go.
	var params []string
	if p.parameters != nil {
		params = cStringsToGo(p.parameters.elements, int(p.parameters.len))
	}

	// Create and initialize a Go solvers properties object and return it.
	propObj := &SolverProperties{
		props: p,
		SupportedProblemTypes: spts,
		IsingRanges:           ranges,
		QuantumProps:          convertQSPs(p),
		AnnealOffsets:         convertAOPs(p),
		Parameters:            params,
	}
	return propObj
}

// HardwareAdjacency returns the adjacency matrix for the solver's underlying
// topology.
func (s *Solver) HardwareAdjacency() (Problem, error) {
	var cProb *C.sapi_Problem
	if ret := C.sapi_getHardwareAdjacency(s.solver, &cProb); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "Failed to query the %s solver's topology", s.Name)
	}
	defer C.sapi_freeProblem(cProb)
	return problemFromC(cProb), nil
}

// An IsingResult represents a solver's output in Ising-model form.
type IsingResult struct {
	Solutions   [][]int8  // Solutions found (±1 or 3 for "unused")
	Energies    []float64 // Energy of each solution
	Occurrences []int     // Tally of occurrences of each solution
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
	occurs := cIntsToGo(result.num_occurrences, ns)

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
	if ret := C.sapi_solveIsing(s.solver, prob, params, &result, &cErr[0]); ret != C.SAPI_OK {
		return IsingResult{}, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}
	return convertIsingResultToGo(result)
}

// SolveQubo solves a QUBO problem.
func (s *Solver) SolveQubo(p Problem, sp SolverParameters) (IsingResult, error) {
	prob := p.toC()
	params := sp.ToC()
	var result *C.sapi_IsingResult
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_solveQubo(s.solver, prob, params, &result, &cErr[0]); ret != C.SAPI_OK {
		return IsingResult{}, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}
	return convertIsingResultToGo(result)
}
