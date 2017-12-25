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
	"time"
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

// toC converts an IsingRangeProperties to a C sapi_IsingRangeProperties.
func (irp *IsingRangeProperties) toC() *C.sapi_IsingRangeProperties {
	var cIrp C.sapi_IsingRangeProperties
	cIrp.h_min = C.double(irp.HMin)
	cIrp.h_max = C.double(irp.HMax)
	cIrp.j_min = C.double(irp.JMin)
	cIrp.j_max = C.double(irp.JMax)
	return &cIrp
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

// Properties returns the properties associated with a SAPI solver.
func (s *Solver) Properties() *SolverProperties {
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

// A Timing tracks where solving time was spent.  Fields deprecated by SAPI 2.4
// are not included here.
type Timing struct {
	QpuAccessTime              time.Duration // Total time in the QPU
	QpuProgrammingTime         time.Duration // Time to program the QPU
	QpuSamplingTime            time.Duration // Total time for R samples, where R is the number of reads/samples
	QpuAnnealTimePerSample     time.Duration // Time for one anneal
	QpuReadoutTimePerSample    time.Duration // Time for one read
	QpuDelayTimePerSample      time.Duration // Rethermalization time between anneals
	TotalPostprocessingTime    time.Duration // Total time spent in postprocessing, including energy calculations and histogramming
	PostprocessingOverheadTime time.Duration // Part of the total postprocessing time that is not concurrent with the QPU
}

// An IsingResult represents a solver's output in Ising-model form.
type IsingResult struct {
	Solutions   [][]int8  // Solutions found (±1 or 3 for "unused")
	Energies    []float64 // Energy of each solution
	Occurrences []int     // Tally of occurrences of each solution
	Timing      Timing    // Solver timing breakdown
}

// convertIsingResultToGo is a helper function for SolveIsing and SolveQubo
// that converts the returned C.sapi_IsingResult structure to a Go-friendly
// format.
func convertIsingResultToGo(result *C.sapi_IsingResult) (IsingResult, error) {
	// Convert the resulting solutions from C to Go.
	ns := int(result.num_solutions)
	sl := int(result.solution_len)
	solns := cInt8MatrixToGo(result.solutions, ns, sl)

	// Convert the resulting energies from C to Go.
	ePtr := (*[1 << 30]C.double)(unsafe.Pointer(result.energies))[:ns:ns]
	energies := make([]float64, ns)
	for i, v := range ePtr {
		energies[i] = float64(v)
	}

	// Convert the resulting tallies from C to Go.
	var occurs []int
	if result.num_occurrences != nil {
		occurs = cIntsToGo(result.num_occurrences, ns)
	}

	// Convert the timing data from C to Go.
	toDur := func(us C.longlong) time.Duration {
		return time.Duration(us) * time.Microsecond
	}
	cTime := result.timing
	times := Timing{
		QpuAccessTime:              toDur(cTime.qpu_access_time),
		QpuProgrammingTime:         toDur(cTime.qpu_programming_time),
		QpuSamplingTime:            toDur(cTime.qpu_sampling_time),
		QpuAnnealTimePerSample:     toDur(cTime.qpu_anneal_time_per_sample),
		QpuReadoutTimePerSample:    toDur(cTime.qpu_readout_time_per_sample),
		QpuDelayTimePerSample:      toDur(cTime.qpu_delay_time_per_sample),
		TotalPostprocessingTime:    toDur(cTime.total_post_processing_time),
		PostprocessingOverheadTime: toDur(cTime.post_processing_overhead_time),
	}

	// Free the C data and return the Go result.
	C.sapi_freeIsingResult(result)
	ir := IsingResult{
		Solutions:   solns,
		Energies:    energies,
		Occurrences: occurs,
		Timing:      times,
	}
	return ir, nil
}

// SolveIsing solves an Ising-model problem.
func (s *Solver) SolveIsing(p Problem, sp SolverParameters) (IsingResult, error) {
	prob := p.toC()
	params := sp.ToCSolverParameters()
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
	params := sp.ToCSolverParameters()
	var result *C.sapi_IsingResult
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_solveQubo(s.solver, prob, params, &result, &cErr[0]); ret != C.SAPI_OK {
		return IsingResult{}, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}
	return convertIsingResultToGo(result)
}

// A SubmittedProblem represents a problem submitted asynchronously to a solver.
type SubmittedProblem struct {
	sp *C.sapi_SubmittedProblem
}

// AsyncSolveIsing submits an Ising-model problem to a solver but does not wait
// for it to complete.
func (s *Solver) AsyncSolveIsing(p Problem, sp SolverParameters) (*SubmittedProblem, error) {
	// Submit the problem.
	prob := p.toC()
	params := sp.ToCSolverParameters()
	var cSub *C.sapi_SubmittedProblem
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_asyncSolveIsing(s.solver, prob, params, &cSub, &cErr[0]); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}
	sub := &SubmittedProblem{sp: cSub}

	// Free the problem when it gets GC'd away.
	runtime.SetFinalizer(sub, func(sub *SubmittedProblem) {
		C.sapi_freeSubmittedProblem(sub.sp)
	})
	return sub, nil
}

// AsyncSolveQubo submits a QUBO problem to a solver but does not wait for it
// to complete.
func (s *Solver) AsyncSolveQubo(p Problem, sp SolverParameters) (*SubmittedProblem, error) {
	// Submit the problem.
	prob := p.toC()
	params := sp.ToCSolverParameters()
	var cSub *C.sapi_SubmittedProblem
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_asyncSolveQubo(s.solver, prob, params, &cSub, &cErr[0]); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}
	sub := &SubmittedProblem{sp: cSub}

	// Free the problem when it gets GC'd away.
	runtime.SetFinalizer(sub, func(sub *SubmittedProblem) {
		C.sapi_freeSubmittedProblem(sub.sp)
	})
	return sub, nil
}

// A SubmittedState represents the state of an asynchronously submitted problem.
type SubmittedState int

// These are the values a SubmittedState can accept.
const (
	StateSubmitting SubmittedState = C.SAPI_STATE_SUBMITTING // Problem is still being submitted
	StateSubmitted                 = C.SAPI_STATE_SUBMITTED  // Problem has been submitted but isn't done yet
	StateDone                      = C.SAPI_STATE_DONE       // Problem is done (completed, failed, or canceled)
	StateRetrying                  = C.SAPI_STATE_RETRYING   // Network communication error occurred but submission/polling is being retried
	StateFailed                    = C.SAPI_STATE_FAILED     // Network communication error occurred while submitting the problem or checking its status
)

// A RemoteStatus represents the status of a problem as reported by the server.
type RemoteStatus int

// These are the values a RemoteStatus can accept.
const (
	StatusUnknown    RemoteStatus = C.SAPI_STATUS_UNKNOWN     // No server response yet (still submitting)
	StatusPending                 = C.SAPI_STATUS_PENDING     // Problem is waiting in a queue
	StatusInProgress              = C.SAPI_STATUS_IN_PROGRESS // Problem is being solved (or will be solved shortly)
	StatusCompleted               = C.SAPI_STATUS_COMPLETED   // Solving succeeded
	StatusFailed                  = C.SAPI_STATUS_FAILED      // Solving failed
	StatusCanceled                = C.SAPI_STATUS_CANCELED    // Problem cancelled by user
)

// A ProblemStatus represents the status of an asynchronously submitted
// problem.  This structure isn’t meaningful for problems running locally.
type ProblemStatus struct {
	ID            string         // Remote problem ID
	TimeReceived  time.Time      // Time at which the server received the problem
	TimeSolved    time.Time      // Time at which the problem was completed
	State         SubmittedState // State of the problem as seen by the client library
	LastGoodState SubmittedState // Last "good" value of state (i.e., not StateFailed or StateRetrying)
	RemoteStatus  RemoteStatus   // Status of the problem as reported by the server
	Error         Error          // Error type when in any kind of failed state
}

// Status returns the current status of an asynchronously submitted problem.
func (sp *SubmittedProblem) Status() (*ProblemStatus, error) {
	// Query the status.
	cSp := sp.sp
	var cPs C.sapi_ProblemStatus
	if ret := C.sapi_asyncStatus(cSp, &cPs); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "sapi_asyncStatus failed")
	}

	// Convert the status from C to Go.
	var err error
	var ps ProblemStatus
	ps.ID = C.GoString(&cPs.problem_id[0])
	ps.TimeReceived, err = time.Parse(time.RFC3339, C.GoString(&cPs.time_received[0]))
	if err != nil {
		return nil, err
	}
	ps.TimeSolved, err = time.Parse(time.RFC3339, C.GoString(&cPs.time_solved[0]))
	if err != nil {
		return nil, err
	}
	ps.State = SubmittedState(cPs.state)
	ps.LastGoodState = SubmittedState(cPs.last_good_state)
	ps.RemoteStatus = RemoteStatus(cPs.remote_status)
	if cPs.error_code != C.SAPI_OK {
		ps.Error = newErrorf(cPs.error_code, C.GoString(&cPs.error_message[0]))
	}
	return &ps, nil
}

// Done says whether a submitted problem has completed.
func (sp *SubmittedProblem) Done() bool {
	return C.sapi_asyncDone(sp.sp) != 0
}

// Cancel cancels a submitted problem.
func (sp *SubmittedProblem) Cancel() {
	C.sapi_cancelSubmittedProblem(sp.sp)
}

// Retry retries a submitted problem that encountered a network, communication,
// or authentication error.
func (sp *SubmittedProblem) Retry() {
	C.sapi_asyncRetry(sp.sp)
}

// AwaitCompletion waits for a submitted problem to complete.  It returns true
// if the problem completed, false if the specified timeout was reached.
func (sp *SubmittedProblem) AwaitCompletion(timeout time.Duration) bool {
	cTime := C.double(timeout.Seconds())
	ret := C.sapi_awaitCompletion(&sp.sp, 1, 1, cTime)
	return ret != 0
}
