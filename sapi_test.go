// This file provides various tests of the features of the sapi package.

package sapi_test

import (
	"github.com/lanl/sapi"
	"os"
	"testing"
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

// getRemoteParams extracts from the environment the parameters needed for a
// remote connection.  If one of the URL, token, or solver name is not set, the
// function skips the current test.
func getRemoteParams(t *testing.T) (url, token, proxy, solver string) {
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
	proxy = os.Getenv("DW_INTERNAL__HTTPPROXY")
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

// TestLocalSolver ensures we can connect to a local solver.
func TestLocalSolver(t *testing.T) {
	conn := sapi.LocalConnection()
	_, err := conn.GetSolver(localSolverName)
	if err != nil {
		t.Fatal(err)
	}
}

// TestRemoteSolver ensures we can connect to a remote solver.
func TestRemoteSolver(t *testing.T) {
	url, token, proxy, solverName := getRemoteParams(t)
	conn, err := sapi.RemoteConnection(url, token, proxy)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.GetSolver(solverName)
	if err != nil {
		t.Fatal(err)
	}
}

// findFourCycle finds a set of four distinct qubits with connections (0, 1), (1,
// 2), (2, 3), and (3, 0).
func findFourCycle(s *sapi.Solver) []int {
	// Construct an adjacency list from the list of couplers.
	props := s.GetProperties()
	adj := make(map[int]map[int]bool)
	for _, cp := range props.Couplers {
		q0, q1 := cp[0], cp[1]
		if _, ok := adj[q0]; !ok {
			adj[q0] = make(map[int]bool, 8)
		}
		adj[q0][q1] = true
		if _, ok := adj[q1]; !ok {
			adj[q1] = make(map[int]bool, 8)
		}
		adj[q1][q0] = true
	}

	// Search every set of four neighbors until we find a square.
	for _, q0 := range props.Qubits {
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

// Solve for all valid rows in an AND truth table.
func testAND(t *testing.T, solver *sapi.Solver) {
	// Find a set of qubits we can use.
	square := findFourCycle(solver)
	if square == nil {
		t.Fatalf("Failed to find a 4-cycle in the %s solver", localSolverName)
	}
	q0, q1, q2, q3 := square[0], square[1], square[2], square[3]

	// Construct a simple problem (an AND truth table).
	prob := make(sapi.Problem, 8)
	prob[0] = sapi.ProblemEntry{I: q0, J: q0, Value: -0.125}
	prob[1] = sapi.ProblemEntry{I: q1, J: q1, Value: -0.125}
	prob[2] = sapi.ProblemEntry{I: q2, J: q2, Value: -0.25}
	prob[3] = sapi.ProblemEntry{I: q3, J: q3, Value: 0.5}
	prob[4] = sapi.ProblemEntry{I: q0, J: q1, Value: -1.0}
	prob[5] = sapi.ProblemEntry{I: q1, J: q2, Value: 0.25}
	prob[6] = sapi.ProblemEntry{I: q2, J: q3, Value: -0.5}
	prob[7] = sapi.ProblemEntry{I: q3, J: q0, Value: -0.5}

	// Set the solver NumReads parameter to a large value.
	sp := solver.NewSolverParameters()
	sp.SetNumReads(1000)

	// Solve the problem.
	ir, err := solver.SolveIsing(prob, sp)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that each solution is either correct or sits at high enough
	// energy that we know it's incorrect.
	const correctEnergy = -1.75
	s2b := map[int8]bool{-1: false, +1: true}
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
	}
}

// TestLocalSolveIsing ensures we can solve an Ising-model problem on a local
// solver.
func TestLocalSolveIsing(t *testing.T) {
	conn := sapi.LocalConnection()
	solver, err := conn.GetSolver(localSolverName)
	if err != nil {
		t.Fatal(err)
	}
	testAND(t, solver)
}

// TestLocalSolveIsing ensures we can solve an Ising-model problem on a remote
// solver.
func TestRemoteSolveIsing(t *testing.T) {
	url, token, proxy, solverName := getRemoteParams(t)
	conn, err := sapi.RemoteConnection(url, token, proxy)
	if err != nil {
		t.Fatal(err)
	}
	solver, err := conn.GetSolver(solverName)
	if err != nil {
		t.Fatal(err)
	}
	testAND(t, solver)
}
