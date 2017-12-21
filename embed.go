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
	FastEmbedding    bool    // Try to get an embedding quickly, without worrying about chain length
	MaxNoImprovement int     // Number of rounds of the algorithm to try from the current solution with no improvement
	UseRandomSeed    bool    // Honor the RandomSeed field (below)
	RandomSeed       uint    // Seed for the random number generator
	Timeout          float64 // Give up after this many seconds
	Tries            int     // Give up after this many retry attempts
	Verbose          bool    // Output verbose information to standard output
	notUsed          int     // Not used, but prevents the caller from bypassing NewFindEmbeddingParameters
}

// toC converts a Go FindEmbeddingParameters to a C
// sapi_FindEmbeddingParameters.
func (fep *FindEmbeddingParameters) toC() *C.sapi_FindEmbeddingParameters {
	var cFep C.sapi_FindEmbeddingParameters
	bool2cint := map[bool]C.int{true: 1, false: 0}
	cFep.fast_embedding = bool2cint[fep.FastEmbedding]
	cFep.max_no_improvement = C.int(fep.MaxNoImprovement)
	cFep.use_random_seed = bool2cint[fep.UseRandomSeed]
	cFep.random_seed = C.uint(fep.RandomSeed)
	cFep.timeout = C.double(fep.Timeout)
	cFep.tries = C.int(fep.Tries)
	cFep.verbose = bool2cint[fep.Verbose]
	return &cFep
}

// findEmbeddingParametersFromC converts a C sapi_FindEmbeddingParameters to a
// Go FindEmbeddingParameters.
func findEmbeddingParametersFromC(cFep *C.sapi_FindEmbeddingParameters) *FindEmbeddingParameters {
	var fep FindEmbeddingParameters
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
	return embed, nil
}

// An EmbedProblemResult represents the result of an embedding of a problem in
// a physical topology.
type EmbedProblemResult struct {
	Prob Problem    // Embedded original problem
	JC   Problem    // Chain edges (J values coupling vertices representing the same logical variable)
	Emb  Embeddings // Original embeddings, possibly modified by cleaning or smearing
}

// EmbedProblem uses the result of FindEmbedding to embed a problem in the
// physical topology.
func EmbedProblem(pr Problem, emb Embeddings, adj Problem, clean, smear bool,
	ranges IsingRangeProperties) (*EmbedProblemResult, error) {
	// Convert each argument from C to Go.
	cPr := pr.toC()
	cAdj := pr.toC()
	cEmb := &C.sapi_Embeddings{
		elements: goIntsToC(emb),
		len:      C.size_t(len(emb)),
	}
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
	return result, nil
}
