// This file provides various tests of the features of the sapi package.

package sapi_test

import (
	"github.com/lanl/sapi"
	"os"
	"testing"
)

// localSolver represents the name of a local solver to connect to.
const localSolverName = "c4-sw_optimize"

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
