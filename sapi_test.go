// This file provides various tests of the features of the sapi package.

package sapi_test

import (
	"github.com/lanl/sapi"
	"os"
	"strings"
	"testing"
	"time"
)

// localSolver represents the name of a local solver to connect to.
const localSolverName = "c4-sw_optimize"

// TestVersion tests that we can query the SAPI version string without
// crashing.
func TestVersion(t *testing.T) {
	v := sapi.Version()
	if v == "" {
		t.Fatal("Expected a non-empty SAPI version string")
	}
	t.Logf("Testing against SAPI version %s", v)
}

// TestChimeraAdjacency tests that we can generate an adjacency list for a
// Chimera.
func TestChimeraAdjacency(t *testing.T) {
	// Generate an adjacency list.
	const (
		M = 3 // Vertical
		N = 4 // Horizontal
		L = 5 // Intra-cell
	)
	adj, err := sapi.ChimeraAdjacency(M, N, L)
	if err != nil {
		t.Fatal(err)
	}

	// Remove one of each pair of symmetric connections.
	oldAdj := adj
	adj = make(sapi.Problem, 0, len(oldAdj)/2)
	for _, a := range oldAdj {
		if a.I < a.J {
			adj = append(adj, a)
		}
	}

	// Rather than check every connection we merely ensure that the list
	// contains the correct number of connections.
	expected := M*N*L*L + // Intra-cell
		L*(M-1)*N + // Up and down inter-cell
		L*M*(N-1) // Left and right inter-cell
	if len(adj) != expected {
		t.Logf("Chimera {%d, %d, %d} connections returned: %v", M, N, L, adj)
		t.Fatalf("Expected %d connections but saw %d", expected, len(adj))
	}
}

// getRemoteParams extracts from the environment the parameters needed for a
// remote connection.  If one of the URL, token, or solver name is not set, the
// function skips the current test.
func getRemoteParams(t *testing.T) (url, token string, proxy *string, solver string) {
	// Define a helper function that indicates a variable is mandatory.
	requireVar := func(k string) string {
		nm := "DW_INTERNAL__" + k
		v := os.Getenv(nm)
		if v == "" {
			t.Skipf("Environment variable %s is not set", nm)
		}
		return v
	}

	// Extract various variables from the environment and return them.
	url = requireVar("HTTPLINK")
	token = requireVar("TOKEN")
	if strp, found := os.LookupEnv("DW_INTERNAL__HTTPPROXY"); found {
		proxy = &strp
	}
	solver = requireVar("SOLVER")
	return
}

// TestLocalConnection ensures we can connect to a local simulator.
func TestLocalConnection(t *testing.T) {
	_ = sapi.LocalConnection()
}

// TestRemoteConnection ensures we can connect to a remote device.
func TestRemoteConnection(t *testing.T) {
	url, token, proxy, _ := getRemoteParams(t)
	_, err := sapi.RemoteConnection(url, token, proxy)
	if err != nil {
		t.Fatal(err)
	}
}

// TestLocalSolversExist ensures we have at least one local solver.
func TestLocalSolversExist(t *testing.T) {
	conn := sapi.LocalConnection()
	_, err := conn.Solver(localSolverName)
	if err != nil {
		t.Fatal(err)
	}
	sList, err := conn.Solvers()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Available solvers: \"%s\"", strings.Join(sList, `", "`))
	if len(sList) < 1 {
		t.Fatal("No solvers found")
	}
}

// TestRemoteSolversExist ensures we have at least one local solver.
func TestRemoteSolversExist(t *testing.T) {
	url, token, proxy, _ := getRemoteParams(t)
	conn, err := sapi.RemoteConnection(url, token, proxy)
	if err != nil {
		t.Fatal(err)
	}
	sList, err := conn.Solvers()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Available solvers: \"%s\"", strings.Join(sList, `", "`))
	if len(sList) < 1 {
		t.Fatalf("No solvers found on connection %s", url)
	}
}

// prepareLocal is a helper function that initializes a local connection and
// solver.
func prepareLocal(t *testing.T) (*sapi.Connection, *sapi.Solver) {
	conn := sapi.LocalConnection()
	solver, err := conn.Solver(localSolverName)
	if err != nil {
		t.Fatal(err)
	}
	return conn, solver
}

// TestLocalSolver ensures we can connect to a local solver.
func TestLocalSolver(t *testing.T) {
	prepareLocal(t)
}

// TestLocalHardwareAdjacency ensures we can query a local solver's topology.
func TestLocalHardwareAdjacency(t *testing.T) {
	_, solver := prepareLocal(t)
	adj, err := solver.HardwareAdjacency()
	if err != nil {
		t.Fatal(err)
	}
	if len(adj) == 0 {
		t.Fatalf("Received an empty adjacency graph for solver %s", localSolverName)
	}
}

// prepareRemote is a helper function that initializes a remote connection and
// solver.
func prepareRemote(t *testing.T) (*sapi.Connection, *sapi.Solver) {
	url, token, proxy, solverName := getRemoteParams(t)
	conn, err := sapi.RemoteConnection(url, token, proxy)
	if err != nil {
		t.Fatal(err)
	}
	solver, err := conn.Solver(solverName)
	if err != nil {
		t.Fatal(err)
	}
	return conn, solver
}

// TestRemoteSolver ensures we can connect to a remote solver.
func TestRemoteSolver(t *testing.T) {
	prepareRemote(t)
}

// TestRemoteHardwareAdjacency ensures we can query a remote solver's topology.
func TestRemoteHardwareAdjacency(t *testing.T) {
	_, solver := prepareRemote(t)
	adj, err := solver.HardwareAdjacency()
	if err != nil {
		t.Fatal(err)
	}
	if len(adj) == 0 {
		t.Fatalf("Received an empty adjacency graph for solver %s", localSolverName)
	}
}

// couplersToAdj constructs an adjacency list from a list of couplers.
func couplersToAdj(cs [][2]int) map[int]map[int]bool {
	adj := make(map[int]map[int]bool)
	for _, c := range cs {
		q0, q1 := c[0], c[1]
		if _, ok := adj[q0]; !ok {
			adj[q0] = make(map[int]bool, 8)
		}
		adj[q0][q1] = true
		if _, ok := adj[q1]; !ok {
			adj[q1] = make(map[int]bool, 8)
		}
		adj[q1][q0] = true
	}
	return adj
}

// findFourCycle finds a set of four distinct qubits with connections (0, 1),
// (1, 2), (2, 3), and (3, 0).
func findFourCycle(s *sapi.Solver) []int {
	// Search every set of four neighbors until we find a square.
	props := s.Properties()
	adj := couplersToAdj(props.QuantumProps.Couplers)
	for _, q0 := range props.QuantumProps.Qubits {
		for q1 := range adj[q0] {
			if q1 == q0 {
				continue
			}
			for q2 := range adj[q1] {
				if q2 == q1 || q2 == q0 {
					continue
				}
				for q3 := range adj[q2] {
					if q3 == q2 || q3 == q1 || q3 == q0 {
						continue
					}

					// Ensure we have a square.
					if adj[q0][q1] &&
						adj[q1][q2] &&
						adj[q2][q3] &&
						adj[q3][q0] {
						return []int{q0, q1, q2, q3}
					}
				}
			}
		}
	}
	return nil
}

// verifyAnd ensures that the results of an AND are correct.
func verifyAnd(t *testing.T, ising bool, square []int, ir sapi.IsingResult) {
	// Ensure that each solution is either correct or sits at high enough
	// energy that we know it's incorrect.
	var correctEnergy float64
	if ising {
		correctEnergy = -1.75
	} else {
		correctEnergy = 0.0
	}
	q0, q1, q2, q3 := square[0], square[1], square[2], square[3]
	s2b := map[int8]bool{-1: false, 0: false, +1: true}
	nSolns := 0
	for i, soln := range ir.Solutions {
		// Extract the AND inputs and output.
		a := s2b[soln[q0]]
		aAlt := s2b[soln[q1]]
		b := s2b[soln[q2]]
		y := s2b[soln[q3]]

		// Skip high-energy solutions.
		if ir.Energies[i] > correctEnergy {
			t.Logf("Ignoring high-energy (%.2f) solution %v=%v AND %v = %v",
				ir.Energies[i], a, aAlt, b, y)
			continue
		}

		// Ensure the solutions that should be valid are indeed so.
		if a != aAlt {
			t.Fatalf("Expected qubits %d and %d to be equal in solution %d", q0, q1, i+1)
		}
		if (a && b) != y {
			t.Fatalf("Saw %v AND %v = %v in solution %d", a, b, y, i+1)
		}
		nSolns++
	}
	if nSolns == 0 {
		t.Fatalf("Saw no valid solutions (and %d invalid ones)", len(ir.Solutions))
	}
}

// testAnd solves for all valid rows in an AND truth table, designed to fit
// within a Chimera graph.
func testAnd(t *testing.T, ising bool, solver *sapi.Solver,
	solverFunc func(sapi.Problem, sapi.SolverParameters) (sapi.IsingResult, error)) {
	// Find a set of qubits we can use.
	square := findFourCycle(solver)
	if square == nil {
		t.Fatalf("Failed to find a 4-cycle in the %s solver", localSolverName)
	}
	q0, q1, q2, q3 := square[0], square[1], square[2], square[3]

	// Construct a simple problem (an AND truth table).
	prob := make(sapi.Problem, 8)
	if ising {
		prob[0] = sapi.ProblemEntry{I: q0, J: q0, Value: -0.125}
		prob[1] = sapi.ProblemEntry{I: q1, J: q1, Value: -0.125}
		prob[2] = sapi.ProblemEntry{I: q2, J: q2, Value: -0.25}
		prob[3] = sapi.ProblemEntry{I: q3, J: q3, Value: 0.5}
		prob[4] = sapi.ProblemEntry{I: q0, J: q1, Value: -1.0}
		prob[5] = sapi.ProblemEntry{I: q1, J: q2, Value: 0.25}
		prob[6] = sapi.ProblemEntry{I: q2, J: q3, Value: -0.5}
		prob[7] = sapi.ProblemEntry{I: q3, J: q0, Value: -0.5}
	} else {
		prob[0] = sapi.ProblemEntry{I: q0, J: q0, Value: 2.75}
		prob[1] = sapi.ProblemEntry{I: q1, J: q1, Value: 1.25}
		prob[2] = sapi.ProblemEntry{I: q2, J: q2, Value: 0.0}
		prob[3] = sapi.ProblemEntry{I: q3, J: q3, Value: 3.0}
		prob[4] = sapi.ProblemEntry{I: q0, J: q1, Value: -4.0}
		prob[5] = sapi.ProblemEntry{I: q1, J: q2, Value: 1.0}
		prob[6] = sapi.ProblemEntry{I: q2, J: q3, Value: -2.0}
		prob[7] = sapi.ProblemEntry{I: q3, J: q0, Value: -2.0}
	}

	// Set the solver's NumReads parameter to a large value.
	sp := solver.NewSolverParameters()
	switch sp := sp.(type) {
	case *sapi.SwOptimizeSolverParameters:
		sp.NumReads = 1000
	case *sapi.SwSampleSolverParameters:
		sp.NumReads = 1000
	case *sapi.QuantumSolverParameters:
		sp.NumReads = 1000
	}

	// Solve the problem.
	ir, err := solverFunc(prob, sp)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that each solution is either correct or sits at high enough
	// energy that we know it's incorrect.
	verifyAnd(t, ising, square, ir)
}

// TestLocalSolveIsing ensures we can solve an Ising-model problem on a local
// solver.
func TestLocalSolveIsing(t *testing.T) {
	_, solver := prepareLocal(t)
	testAnd(t, true, solver, solver.SolveIsing)
}

// TestRemoteSolveIsing ensures we can solve an Ising-model problem on a remote
// solver.
func TestRemoteSolveIsing(t *testing.T) {
	_, solver := prepareRemote(t)
	testAnd(t, true, solver, solver.SolveIsing)
}

// TestLocalSolveQubo ensures we can solve an QUBO problem on a local solver.
func TestLocalSolveQubo(t *testing.T) {
	_, solver := prepareLocal(t)
	testAnd(t, false, solver, solver.SolveQubo)
}

// TestLocalSolveQubo ensures we can solve a QUBO problem on a remote solver.
func TestRemoteSolveQubo(t *testing.T) {
	_, solver := prepareRemote(t)
	testAnd(t, false, solver, solver.SolveQubo)
}

// TestLocalAsyncSolveIsing ensures we can asynchronously solve an Ising-model
// problem on a local solver.
func TestLocalAsyncSolveIsing(t *testing.T) {
	_, solver := prepareLocal(t)
	run := func(prob sapi.Problem, sp sapi.SolverParameters) (sapi.IsingResult, error) {
		sub, err := solver.AsyncSolveIsing(prob, sp)
		if err != nil {
			return sapi.IsingResult{}, err
		}
		for !sub.AwaitCompletion(3 * time.Second) {
		}
		return sub.Result()
	}
	testAnd(t, true, solver, run)
}

// TestRemoteAsyncSolveIsing ensures we can asynchronously solve an Ising-model
// problem on a remote solver.
func TestRemoteAsyncSolveIsing(t *testing.T) {
	_, solver := prepareRemote(t)
	run := func(prob sapi.Problem, sp sapi.SolverParameters) (sapi.IsingResult, error) {
		sub, err := solver.AsyncSolveIsing(prob, sp)
		if err != nil {
			return sapi.IsingResult{}, err
		}
		for !sub.AwaitCompletion(3 * time.Second) {
		}
		return sub.Result()
	}
	testAnd(t, true, solver, run)
}

// TestLocalAsyncSolveQubo ensures we can asynchronously solve a QUBO problem
// on a local solver.
func TestLocalAsyncSolveQubo(t *testing.T) {
	_, solver := prepareLocal(t)
	run := func(prob sapi.Problem, sp sapi.SolverParameters) (sapi.IsingResult, error) {
		sub, err := solver.AsyncSolveQubo(prob, sp)
		if err != nil {
			return sapi.IsingResult{}, err
		}
		for !sub.AwaitCompletion(3 * time.Second) {
		}
		return sub.Result()
	}
	testAnd(t, true, solver, run)
}

// TestRemoteAsyncSolveQubo ensures we can asynchronously solve a QUBO problem
// on a remote solver.
func TestRemoteAsyncSolveQubo(t *testing.T) {
	_, solver := prepareRemote(t)
	run := func(prob sapi.Problem, sp sapi.SolverParameters) (sapi.IsingResult, error) {
		sub, err := solver.AsyncSolveQubo(prob, sp)
		if err != nil {
			return sapi.IsingResult{}, err
		}
		for !sub.AwaitCompletion(3 * time.Second) {
		}
		return sub.Result()
	}
	testAnd(t, true, solver, run)
}

// testEmbedding ensures we can embed an XOR problem in a solver's topology,
// solve it, and get the correct answer.
func testEmbedding(t *testing.T, solver *sapi.Solver) {
	// Define an XOR function, not embedded in a Chimera graph.
	prob := make(sapi.Problem, 10)
	prob[0] = sapi.ProblemEntry{I: 0, J: 0, Value: 0.5}
	prob[1] = sapi.ProblemEntry{I: 1, J: 1, Value: 0.5}
	prob[2] = sapi.ProblemEntry{I: 2, J: 2, Value: 0.5}
	prob[3] = sapi.ProblemEntry{I: 3, J: 3, Value: -1.0}
	prob[4] = sapi.ProblemEntry{I: 0, J: 1, Value: 0.5}
	prob[5] = sapi.ProblemEntry{I: 0, J: 2, Value: 0.5}
	prob[6] = sapi.ProblemEntry{I: 0, J: 3, Value: -1.0}
	prob[7] = sapi.ProblemEntry{I: 1, J: 2, Value: 0.5}
	prob[8] = sapi.ProblemEntry{I: 1, J: 3, Value: -1.0}
	prob[9] = sapi.ProblemEntry{I: 2, J: 3, Value: -1.0}

	// Retrieve the solver's adjacency graph and coefficient ranges.
	adj, err := solver.HardwareAdjacency()
	if err != nil {
		t.Fatal(err)
	}
	prop := solver.Properties()
	ir := prop.IsingRanges
	if ir == nil {
		ir = &sapi.IsingRangeProperties{
			HMin: -1,
			HMax: 1,
			JMin: -1,
			JMax: 1,
		}
	}

	// Run the heuristic embedder.
	fep := sapi.NewFindEmbeddingParameters()
	fep.Verbose = false
	emb, err := sapi.FindEmbedding(prob, adj, fep)
	if err != nil {
		t.Fatal(err)
	}
	epr, err := sapi.EmbedProblem(prob, emb, adj, true, true, *ir)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a new problem from the embedded problem and the
	// newly introduced chains.
	const chStr = -2.0 // Chain strength
	eProb := make(sapi.Problem, len(epr.Prob), len(epr.Prob)+len(epr.JC))
	copy(eProb, epr.Prob)
	for _, ch := range epr.JC {
		pe := ch
		pe.Value = chStr
		eProb = append(eProb, pe)
	}

	// Set the solver's NumReads parameter to a large value.
	sp := solver.NewSolverParameters()
	switch sp := sp.(type) {
	case *sapi.SwOptimizeSolverParameters:
		sp.NumReads = 1000
	case *sapi.SwSampleSolverParameters:
		sp.NumReads = 1000
	case *sapi.QuantumSolverParameters:
		sp.NumReads = 1000
		sp.AutoScale = true
	}

	// Solve the problem.
	res, err := solver.SolveIsing(eProb, sp)
	if err != nil {
		t.Fatal(err)
	}

	// Unembed the answer.
	solns, err := sapi.UnembedAnswer(res.Solutions, epr.Emb,
		sapi.BrokenChainsMinimizeEnergy, prob)
	if err != nil {
		t.Fatal(err)
	}

	// Validate the solutions.  Because the energy of a correct solution
	// depends on the embedding, we check all lowest-energy solutions and
	// ignore all higher-energy solutions.
	correctEnergy := res.Energies[0]
	nSolns := 0
	for i, soln := range solns {
		a, b, y := (soln[0]+1)/2, (soln[1]+1)/2, (soln[2]+1)/2
		e := res.Energies[i]
		if e > correctEnergy {
			t.Logf("Ignoring high-energy (%.2f) solution %v XOR %v = %v",
				e, a == 1, b == 1, y == 1)
			continue
		}
		if (a ^ b) != y {
			t.Fatalf("Saw %v XOR %v = %v in solution %d (energy = %f)", a == 1, b == 1, y == 1, i+1, e)
		}
		nSolns++
	}
	if nSolns == 0 {
		t.Fatalf("Saw no valid solutions (and %d invalid ones)", len(solns))
	}
}

// TestLocalEmbedding ensures we can embed a problem in a local solver's
// topology, solve it, and get the correct answer.
func TestLocalEmbedding(t *testing.T) {
	_, solver := prepareLocal(t)
	testEmbedding(t, solver)
}

// TestRemoteEmbedding ensures we can embed a problem in a remote solver's
// topology, solve it, and get the correct answer.
func TestRemoteEmbedding(t *testing.T) {
	_, solver := prepareRemote(t)
	testEmbedding(t, solver)
}

// TestFixVariables ensures that FixVariables can detect that a problem
// variable is unnecessary.
func TestFixVariables(t *testing.T) {
	// Construct a QUBO problem.
	prob := make(sapi.Problem, 8)
	prob[0] = sapi.ProblemEntry{I: 1, J: 1, Value: 1}
	prob[1] = sapi.ProblemEntry{I: 2, J: 2, Value: 1}
	prob[2] = sapi.ProblemEntry{I: 3, J: 3, Value: 1}
	prob[3] = sapi.ProblemEntry{I: 4, J: 4, Value: 3}
	prob[4] = sapi.ProblemEntry{I: 1, J: 2, Value: 1}
	prob[5] = sapi.ProblemEntry{I: 1, J: 3, Value: -2}
	prob[6] = sapi.ProblemEntry{I: 2, J: 3, Value: -2}
	prob[7] = sapi.ProblemEntry{I: 1, J: 4, Value: 4}

	// Find fixed variables.
	fvr, err := prob.FixVariables(sapi.FixVariablesMethodOptimized)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the result.
	v, ok := fvr.FixedVars[4]
	if !ok {
		t.Fatal("Expected to see variable 4 fixed, but it wasn't")
	}
	if v != 0 {
		t.Fatalf("Expected to see variable 4 fixed to 0, but it was fixed to %d", v)
	}
	delete(fvr.FixedVars, 4)
	for k, v := range fvr.FixedVars {
		t.Fatalf("Did not expect variable %d to be fixed to %d", k, v)
	}
}
