// This file presents an interface to SAPI solver-related types and functions.
// Functions related to asynchronous execution are in this file; functions
// related to synchronous execution are in solver.go

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"runtime"
	"time"
)

// A SubmittedProblem represents a problem submitted asynchronously to a solver.
type SubmittedProblem struct {
	cSp *C.sapi_SubmittedProblem
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
	sub := &SubmittedProblem{cSp: cSub}

	// Free the problem when it gets GC'd away.
	runtime.SetFinalizer(sub, func(sub *SubmittedProblem) {
		C.sapi_freeSubmittedProblem(sub.cSp)
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
	sub := &SubmittedProblem{cSp: cSub}

	// Free the problem when it gets GC'd away.
	runtime.SetFinalizer(sub, func(sub *SubmittedProblem) {
		C.sapi_freeSubmittedProblem(sub.cSp)
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
	cSp := sp.cSp
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

// Done says whether an asynchronously submitted problem has completed.
func (sp *SubmittedProblem) Done() bool {
	return C.sapi_asyncDone(sp.cSp) != 0
}

// Cancel cancels an asynchronously submitted problem.
func (sp *SubmittedProblem) Cancel() {
	C.sapi_cancelSubmittedProblem(sp.cSp)
}

// Retry retries an asynchronously submitted problem that encountered a
// network, communication, or authentication error.
func (sp *SubmittedProblem) Retry() {
	C.sapi_asyncRetry(sp.cSp)
}

// AwaitCompletion waits for an asynchronously submitted problem to complete.
// It returns true if the problem completed, false if the specified timeout was
// reached.
func (sp *SubmittedProblem) AwaitCompletion(timeout time.Duration) bool {
	cTime := C.double(timeout.Seconds())
	ret := C.sapi_awaitCompletion(&sp.cSp, 1, 1, cTime)
	return ret != 0
}

// AwaitCompletion waits for multiple asynchronously submitted problems to
// complete.  It returns true if a minimum number of problems completed, false
// if the specified timeout was reached first.  For a single submitted problem,
// SubmittedProblem.AwaitCompletion may be more convenient.
func AwaitCompletion(sps []*SubmittedProblem, minDone int, timeout time.Duration) bool {
	// Create a list of C sapi_SubmittedProblem pointers.
	cSps := make([]*C.sapi_SubmittedProblem, len(sps))
	for i, s := range sps {
		cSps[i] = s.cSp
	}

	// Invoke the C function.
	cTime := C.double(timeout.Seconds())
	ret := C.sapi_awaitCompletion(&cSps[0], C.size_t(len(sps)), C.size_t(minDone), cTime)
	return ret != 0
}

// Result returns the result of asynchronously submitted problem.
func (sp *SubmittedProblem) Result() (IsingResult, error) {
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	var result *C.sapi_IsingResult
	if ret := C.sapi_asyncResult(sp.cSp, &result, &cErr[0]); ret != C.SAPI_OK {
		return IsingResult{}, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}
	return convertIsingResultToGo(result)
}
