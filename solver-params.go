// This file presents an interface to SAPI types and functions related to
// solver parameters.

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"strings"
	"unsafe"
)

// A SolverParameterAnswerMode indicates the format in which we want the solver
// to return solutions.
type SolverParameterAnswerMode int

// These are answer modes a solver can accept.
const (
	AnswerModeHistogram SolverParameterAnswerMode = C.SAPI_ANSWER_MODE_HISTOGRAM
	AnswerModeRaw                                 = C.SAPI_ANSWER_MODE_RAW
)

// SolverParameters is presented as an interface so the caller does not need to
// use different data structures for the different solver types (quantum or the
// various software solvers).
type SolverParameters interface {
	SetAnnealingTime(us int)
	SetAutoScale(y bool)
	SetAnswerMode(m SolverParameterAnswerMode)
	SetBeta(b float64)
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

// SetAnswerMode specifies the form in which we want to see the solver's output.
func (p *SwSampleSolverParameters) SetAnswerMode(m SolverParameterAnswerMode) {
	p.sssp.answer_mode = C.sapi_SolverParameterAnswerMode(m)
}

// SetBeta specifies the Boltzmann distribution parameter.
func (p *SwSampleSolverParameters) SetBeta(b float64) {
	p.sssp.beta = C.double(b)
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

// SetAnswerMode specifies the form in which we want to see the solver's output.
func (p *SwOptimizeSolverParameters) SetAnswerMode(m SolverParameterAnswerMode) {
	p.sosp.answer_mode = C.sapi_SolverParameterAnswerMode(m)
}

// SetBeta specifies the Boltzmann distribution parameter.
func (p *SwOptimizeSolverParameters) SetBeta(b float64) {
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

// SetAnswerMode specifies the form in which we want to see the solver's output.
func (p *SwHeuristicSolverParameters) SetAnswerMode(m SolverParameterAnswerMode) {
}

// SetBeta specifies the Boltzmann distribution parameter.
func (p *SwHeuristicSolverParameters) SetBeta(b float64) {
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

// SetAnswerMode specifies the form in which we want to see the solver's output.
func (p *QuantumSolverParameters) SetAnswerMode(m SolverParameterAnswerMode) {
	p.qsp.answer_mode = C.sapi_SolverParameterAnswerMode(m)
}

// SetBeta specifies the Boltzmann distribution parameter.
func (p *QuantumSolverParameters) SetBeta(b float64) {
	p.qsp.beta = C.double(b)
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
		return NewSwOptimizeSolverParameters()
	case strings.HasSuffix(s.Name, "-sw_sample"):
		return NewSwSampleSolverParameters()
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
