// This file presents examples of the sapi package's features.

package sapi_test

import (
	"fmt"
	"github.com/lanl/sapi"
	"os"
	"time"
)

// Declare global variables to convey that these would be initialized
// outside of the code excerpts that comprise our examples.
var (
	solver *sapi.Solver
	prob   sapi.Problem
	sp     sapi.SolverParameters
)

// Connect to a remote solver by reading connection parameters from environment
// variables (using the dw tool's naming conventions).  If either the URL or
// token is not specified, establish a local connection instead.  This is
// essentially the operation of the NewSolver convenience function.
func ExampleRemoteConnection() {
	// Query the environment for the connection parameters.
	url := os.Getenv("DW_INTERNAL__HTTPLINK")
	token := os.Getenv("DW_INTERNAL__TOKEN")
	var proxy *string
	if strp, found := os.LookupEnv("DW_INTERNAL__HTTPPROXY"); found {
		proxy = &strp
	}

	// Establish a connection to either a remote or local solver.
	var conn *sapi.Connection
	var err error
	if url == "" || token == "" {
		conn = sapi.LocalConnection()
	} else {
		conn, err = sapi.RemoteConnection(url, token, proxy)
		if err != nil {
			panic(err)
		}
	}

	// Code to acquire a solver from the connection would normally appear
	// here.
	_ = conn
}

// Specify solver-specific parameters.
func ExampleSolverParameters() {
	// Set the number of reads to 1000.  In the case of
	// sapi.QuantumSolverParameters, also enable autoscaling.  Note that
	// sapi.SwHeuristicSolverParameters doesn't accept either of those
	// parameters so a case for that type is not included in the following.
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

	// Code to pass sp to one of the Solve* calls would normally appear
	// here.
	_ = sp
}

// Submit a problem asynchronously and wait for it to complete.
func ExampleSolver_AsyncSolveIsing() {
	// Asynchronously solve problem prob with solver parameters sp.
	sub, err := solver.AsyncSolveIsing(prob, sp)
	if err != nil {
		panic(err)
	}
	for !sub.AwaitCompletion(2 * time.Second) {
	}
	ir, err := sub.Result()
	if err != nil {
		panic(err)
	}

	// Code to do something with ir would normally appear here.
	_ = ir
}

// Solve a maximally frustrated problem on a local solver.
func Example_frustration() {
	// Connect to the c4-sw_optimize local solver.  See the
	// RemoteConnection example for code that connects to either a local or
	// remote solver based on a set of environment variables.
	conn := sapi.LocalConnection()
	solver, err := conn.Solver("c4-sw_optimize")
	if err != nil {
		panic(err)
	}

	// Construct an Ising-model problem in which all edges in the entire
	// graph are antiferromagnetically coupled.
	adj, err := solver.HardwareAdjacency()
	if err != nil {
		panic(err)
	}
	prob := make(sapi.Problem, len(adj))
	for i, cp := range adj {
		prob[i].I = cp.I
		prob[i].J = cp.J
		prob[i].Value = 1.0
	}

	// Solve the problem using the solver's default parameters.  See the
	// SolverParameters example for code that sets solver-specific
	// parameters.
	sp := solver.NewSolverParameters()
	ir, err := solver.SolveIsing(prob, sp)
	if err != nil {
		panic(err)
	}

	// Output all of the solutions found.
	for i, soln := range ir.Solutions {
		fmt.Printf("%5d) energy = %f, tally = %d, solution = %v\n",
			i+1, ir.Energies[i], ir.Occurrences[i], soln)
	}
}
