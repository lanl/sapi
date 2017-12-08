// The sapi package provides a Go interface to D-Wave's SAPI library.
package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"fmt"
	"runtime"
	"strings"
	"unsafe"
)

// init initializes SAPI.
func init() {
	if C.sapi_globalInit() != C.SAPI_OK {
		panic("Failed to initialize SAPI")
	}
}

// A Connection represents a connection to a remote solver.
type Connection struct {
	conn  *C.sapi_Connection // SAPI connection object
	URL   string             // Connection name
	Token string             // Token to authenticate a user
	Proxy string             // Proxy URL
}

// LocalConnection returns a connection to the local solver.
func LocalConnection() *Connection {
	conn := C.sapi_localConnection()
	return &Connection{
		conn:  conn,
		URL:   "",
		Token: "",
		Proxy: "",
	}
}

// RemoteConnection establishes a connection to a remote solver.
func RemoteConnection(url, token, proxy string) (*Connection, error) {
	// Establish a connection.
	var conn *C.sapi_Connection
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))
	cProxy := C.CString(proxy)
	defer C.free(unsafe.Pointer(cProxy))
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	ret := C.sapi_remoteConnection(cURL, cToken, cProxy, &conn, &cErr[0])
	if ret != C.SAPI_OK {
		return nil, fmt.Errorf("%s", C.GoString(&cErr[0]))
	}
	connObj := &Connection{
		conn:  conn,
		URL:   url,
		Token: token,
		Proxy: proxy,
	}

	// Free the connection when it gets GC'd away.
	runtime.SetFinalizer(connObj, func(c *Connection) {
		if c.conn != nil {
			C.sapi_freeConnection(c.conn)
			c.conn = nil
		}
	})
	return connObj, nil
}

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

// SolverParameters is presented as an interface so the caller does not need to
// use different data structures for the different solver types (quantum or the
// various software solvers).
type SolverParameters interface {
	SetAnnealingTime(us int)
	SetAutoScale(y bool)
	SetNumReads(nr int)
	SetNumSpinReversals(sr int)
	ToC() *C.sapi_SolverParameters
}

// A SwSampleSolverParameters represents the parameters that can be passed to a
// sampling software solver.  It implements the SolverParameters interface.
type SwSampleSolverParameters struct {
	sssp C.sapi_SwSampleSolverParameters
}

// NewSwSampleSolverParameters returns a new SwSampleSolverParameters.
func NewSwSampleSolverParameters() *SwSampleSolverParameters {
	return &SwSampleSolverParameters{
		sssp: C.SAPI_SW_SAMPLE_SOLVER_DEFAULT_PARAMETERS,
	}
}

// SetAnnealingTime specifies the annealing time in microseconds (unused by
// this solver).
func (p *SwSampleSolverParameters) SetAnnealingTime(us int) {
}

// SetAutoScale specifies whether coefficients should be automatically scaled
// (unused by this solver).
func (p *SwSampleSolverParameters) SetAutoScale(y bool) {
}

// SetNumReads specifies the number of reads to take.
func (p *SwSampleSolverParameters) SetNumReads(nr int) {
	p.sssp.num_reads = C.int(nr)
}

// SetNumSpinReversals specifies the number of spin-reversal transformations to
// perform (unused by this solver).
func (p *SwSampleSolverParameters) SetNumSpinReversals(sr int) {
}

// ToC converts a SwSampleSolverParameters to a sapi_SolverParameters
func (p *SwSampleSolverParameters) ToC() *C.sapi_SolverParameters {
	return (*C.sapi_SolverParameters)(unsafe.Pointer(&p.sssp))
}

// A SwOptimizeSolverParameters represents the parameters that can be passed to
// an optimizing software solver.  It implements the SolverParameters
// interface.
type SwOptimizeSolverParameters struct {
	sosp C.sapi_SwOptimizeSolverParameters
}

// NewSwOptimizeSolverParameters returns a new SwOptimizeSolverParameters.
func NewSwOptimizeSolverParameters() *SwOptimizeSolverParameters {
	return &SwOptimizeSolverParameters{
		sosp: C.SAPI_SW_OPTIMIZE_SOLVER_DEFAULT_PARAMETERS,
	}
}

// SetAnnealingTime specifies the annealing time in microseconds (unused by
// this solver).
func (p *SwOptimizeSolverParameters) SetAnnealingTime(us int) {
}

// SetAutoScale specifies whether coefficients should be automatically scaled
// (unused by this solver).
func (p *SwOptimizeSolverParameters) SetAutoScale(y bool) {
}

// SetNumReads specifies the number of reads to take.
func (p *SwOptimizeSolverParameters) SetNumReads(nr int) {
	p.sosp.num_reads = C.int(nr)
}

// SetNumSpinReversals specifies the number of spin-reversal transformations to
// perform (unused by this solver).
func (p *SwOptimizeSolverParameters) SetNumSpinReversals(sr int) {
}

// ToC converts a SwOptimizeSolverParameters to a sapi_SolverParameters
func (p *SwOptimizeSolverParameters) ToC() *C.sapi_SolverParameters {
	return (*C.sapi_SolverParameters)(unsafe.Pointer(&p.sosp))
}

// A SwHeuristicSolverParameters represents the parameters that can be passed
// to a heuristic software solver.  It implements the SolverParameters
// interface.
type SwHeuristicSolverParameters struct {
	swsp C.sapi_SwHeuristicSolverParameters
}

// NewSwHeuristicSolverParameters returns a new SwHeuristicSolverParameters.
func NewSwHeuristicSolverParameters() *SwHeuristicSolverParameters {
	return &SwHeuristicSolverParameters{
		swsp: C.SAPI_SW_HEURISTIC_SOLVER_DEFAULT_PARAMETERS,
	}
}

// SetAnnealingTime specifies the annealing time in microseconds (unused by
// this solver).
func (p *SwHeuristicSolverParameters) SetAnnealingTime(us int) {
}

// SetAutoScale specifies whether coefficients should be automatically scaled
// (unused by this solver).
func (p *SwHeuristicSolverParameters) SetAutoScale(y bool) {
}

// SetNumReads specifies the number of reads to take (unused by this solver).
func (p *SwHeuristicSolverParameters) SetNumReads(nr int) {
}

// SetNumSpinReversals specifies the number of spin-reversal transformations to
// perform (unused by this solver).
func (p *SwHeuristicSolverParameters) SetNumSpinReversals(sr int) {
}

// ToC converts a SwHeuristicSolverParameters to a sapi_SolverParameters
func (p *SwHeuristicSolverParameters) ToC() *C.sapi_SolverParameters {
	return (*C.sapi_SolverParameters)(unsafe.Pointer(&p.swsp))
}

// A QuantumSolverParameters represents the parameters that can be passed to a
// quantum solver.  It implements the SolverParameters interface.
type QuantumSolverParameters struct {
	qsp C.sapi_QuantumSolverParameters
}

// NewQuantumSolverParameters returns a new QuantumSolverParameters.
func NewQuantumSolverParameters() *QuantumSolverParameters {
	return &QuantumSolverParameters{
		qsp: C.SAPI_QUANTUM_SOLVER_DEFAULT_PARAMETERS,
	}
}

// SetAnnealingTime specifies the annealing time in microseconds.
func (p *QuantumSolverParameters) SetAnnealingTime(us int) {
	p.qsp.annealing_time = C.int(us)
}

// SetAutoScale specifies whether coefficients should be automatically scaled.
func (p *QuantumSolverParameters) SetAutoScale(y bool) {
	if y {
		p.qsp.auto_scale = 1
	} else {
		p.qsp.auto_scale = 0
	}
}

// SetNumReads specifies the number of reads to take.
func (p *QuantumSolverParameters) SetNumReads(nr int) {
	p.qsp.num_reads = C.int(nr)
}

// SetNumSpinReversals specifies the number of spin-reversal transformations to
// perform.
func (p *QuantumSolverParameters) SetNumSpinReversals(sr int) {
	p.qsp.num_spin_reversal_transforms = C.int(sr)
}

// ToC converts a QuantumSolverParameters to a sapi_SolverParameters
func (p *QuantumSolverParameters) ToC() *C.sapi_SolverParameters {
	return (*C.sapi_SolverParameters)(unsafe.Pointer(&p.qsp))
}

// NewSolverParameters returns an appropriate SolverParameters for the solver
// type.
func (s *Solver) NewSolverParameters() SolverParameters {
	switch {
	case strings.HasSuffix(s.Name, "-sw_optimize"):
		return NewSwSampleSolverParameters()
	case strings.HasSuffix(s.Name, "-sw_sample"):
		return NewSwOptimizeSolverParameters()
	case strings.HasSuffix(s.Name, "-heuristic"):
		return NewSwHeuristicSolverParameters()
	default:
		return NewQuantumSolverParameters()
	}
}

// An IsingResult represents a solver's output in Ising-model form.
type IsingResult struct {
	Solutions   [][]int8  // Solutions found (±1 or 3 for "unused")
	Energies    []float64 // Energy of each solution
	Occurrences []int     // Tally of occurrences of each solution
}

// SolveIsing solves an Ising-model problem.
func (s *Solver) SolveIsing(p Problem, sp SolverParameters) (IsingResult, error) {
	// Submit the problem to the solver.
	prob := p.toC()
	params := sp.ToC()
	var result *C.sapi_IsingResult
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if C.sapi_solveIsing(s.solver, prob, params, &result, &cErr[0]) != C.SAPI_OK {
		return IsingResult{}, fmt.Errorf("%s", C.GoString(&cErr[0]))
	}

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
