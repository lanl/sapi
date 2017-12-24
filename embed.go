// This file provides functions for embedding problems in a topology.

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"unsafe"
)

// FindEmbeddingParameters encapsulate the parameters for FindEmbedding.
type FindEmbeddingParameters struct {
	cFep             C.sapi_FindEmbeddingParameters // C version of FindEmbeddingParameters
	FastEmbedding    bool                           // Try to get an embedding quickly, without worrying about chain length
	MaxNoImprovement int                            // Number of rounds of the algorithm to try from the current solution with no improvement
	UseRandomSeed    bool                           // Honor the RandomSeed field (below)
	RandomSeed       uint                           // Seed for the random number generator
	Timeout          float64                        // Give up after this many seconds
	Tries            int                            // Give up after this many retry attempts
	Verbose          bool                           // Output verbose information to standard output
}

// toC converts a Go FindEmbeddingParameters to a C
// sapi_FindEmbeddingParameters.
func (fep *FindEmbeddingParameters) toC() *C.sapi_FindEmbeddingParameters {
	bool2cint := map[bool]C.int{true: 1, false: 0}
	cFep := &fep.cFep
	cFep.fast_embedding = bool2cint[fep.FastEmbedding]
	cFep.max_no_improvement = C.int(fep.MaxNoImprovement)
	cFep.use_random_seed = bool2cint[fep.UseRandomSeed]
	cFep.random_seed = C.uint(fep.RandomSeed)
	cFep.timeout = C.double(fep.Timeout)
	cFep.tries = C.int(fep.Tries)
	cFep.verbose = bool2cint[fep.Verbose]
	return cFep
}

// findEmbeddingParametersFromC converts a C sapi_FindEmbeddingParameters to a
// Go FindEmbeddingParameters.
func findEmbeddingParametersFromC(cFep *C.sapi_FindEmbeddingParameters) *FindEmbeddingParameters {
	var fep FindEmbeddingParameters
	fep.cFep = *cFep
	fep.FastEmbedding = cFep.fast_embedding != 0
	fep.MaxNoImprovement = int(cFep.max_no_improvement)
	fep.UseRandomSeed = cFep.use_random_seed != 0
	fep.RandomSeed = uint(cFep.random_seed)
	fep.Timeout = float64(cFep.timeout)
	fep.Tries = int(cFep.tries)
	fep.Verbose = cFep.verbose != 0
	return &fep
}

// NewFindEmbeddingParameters returns a new FindEmbeddingParameters,
// initialized using a set of default parameters.
func NewFindEmbeddingParameters() *FindEmbeddingParameters {
	return findEmbeddingParametersFromC(&C.SAPI_FIND_EMBEDDING_DEFAULT_PARAMETERS)
}

// Embeddings indicates the logical variable e[i] that maps to physical qubit i
// (or -1 for no logical variable).
type Embeddings []int

// toC converts an Embeddings to a C sapi_Embeddings.
func (emb Embeddings) toC() *C.sapi_Embeddings {
	return &C.sapi_Embeddings{
		elements: goIntsToC(emb),
		len:      C.size_t(len(emb)),
	}
}

// FindEmbedding attempts to find an embedding of a Ising/QUBO problem in a
// graph. This function is entirely heuristic: failure to return an embedding
// does not prove that no embedding exists.
func FindEmbedding(pr, adj Problem, fep *FindEmbeddingParameters) (Embeddings, error) {
	// Find an embedding.
	cPr := pr.toC()
	cAdj := adj.toC()
	cFep := fep.toC()
	var cEmbed *C.sapi_Embeddings
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_findEmbedding(cPr, cAdj, cFep, &cEmbed, &cErr[0]); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}

	// Convert the embedding from C to Go.
	ne := int(cEmbed.len)
	embed := make(Embeddings, ne)
	ePtr := (*[1 << 30]C.int)(unsafe.Pointer(cEmbed.elements))[:ne:ne]
	for i, e := range ePtr {
		embed[i] = int(e)
	}
	C.sapi_freeEmbeddings(cEmbed)
	return embed, nil
}

// An EmbedProblemResult represents the result of an embedding of a problem in
// a physical topology.
type EmbedProblemResult struct {
	Prob Problem    // Embedded problem
	JC   Problem    // Chain edges (J values coupling vertices representing the same logical variable)
	Emb  Embeddings // Embeddings, possibly modified by cleaning or smearing
}

// EmbedProblem uses the result of FindEmbedding to embed a problem in the
// physical topology.
func EmbedProblem(pr Problem, emb Embeddings, adj Problem, clean, smear bool,
	ranges IsingRangeProperties) (*EmbedProblemResult, error) {
	// Convert each argument from Go to C.
	cPr := pr.toC()
	cAdj := adj.toC()
	cEmb := emb.toC()
	var cClean, cSmear C.int
	if clean {
		cClean = 1
	}
	if smear {
		cSmear = 1
	}
	cRanges := ranges.toC()

	// Invoke the C function.
	var cResult *C.sapi_EmbedProblemResult
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_embedProblem(cPr, cEmb, cAdj, cClean, cSmear, cRanges, &cResult, &cErr[0]); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}

	// Convert the result from C to Go.
	result := &EmbedProblemResult{
		Prob: problemFromC(&cResult.problem),
		JC:   problemFromC(&cResult.jc),
		Emb:  cIntsToGo(cResult.embeddings.elements, int(cResult.embeddings.len)),
	}
	C.sapi_freeEmbedProblemResult(cResult)
	return result, nil
}

// BrokenChains specifies how broken chains should be handled.
type BrokenChains int

// These are the valid values for a BrokenChains variable.
const (
	BrokenChainsMinimizeEnergy BrokenChains = C.SAPI_BROKEN_CHAINS_MINIMIZE_ENERGY
	BrokenChainsVote                        = C.SAPI_BROKEN_CHAINS_VOTE
	BrokenChainsDiscard                     = C.SAPI_BROKEN_CHAINS_DISCARD
	BrokenChainsWeightedRandom              = C.SAPI_BROKEN_CHAINS_WEIGHTED_RANDOM
)

// UnembedAnswer maps an answer from using physical qubit numbers back to
// logical qubit numbers.
func UnembedAnswer(solns [][]int8, emb Embeddings, broken BrokenChains, prob Problem) ([][]int8, error) {
	// Convert each argument from Go to C.
	cSolns := int8MatrixtoC(solns)
	cEmb := emb.toC()
	cBroken := C.sapi_BrokenChains(broken)
	cProb := prob.toC()

	// Invoke the C function.
	nv := prob.countQubits()
	cNew := (*C.int)(C.malloc(C.sizeof_int * C.size_t(len(solns)*nv)))
	var cNnew C.size_t
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	if ret := C.sapi_unembedAnswer(cSolns, C.size_t(len(solns[0])), C.size_t(len(solns)),
		cEmb, cBroken, cProb, cNew, &cNnew, &cErr[0]); ret != C.SAPI_OK {
		return nil, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}

	// Convert the result from C to Go.
	return cInt8MatrixToGo(cNew, int(cNnew), nv), nil
}
