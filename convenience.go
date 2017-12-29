// This file provides convenience routines that do not map to individual SAPI
// functions.

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"os"
)

// NewSolver is a convenience function that establishes either a remote or
// local connection to a solver.  NewSolver queries the environment for
// connection parameters solver URL (DW_INTERNAL__HTTPLINK), API token
// (DW_INTERNAL__TOKEN), proxy URL (DW_INTERNAL__HTTPPROXY), and solver name
// (DW_INTERNAL__SOLVER) and invokes either RemoteConnection and
// LocalConnection, as appropriate, followed by the Solver method on the
// corresponding connection.
func NewSolver() (*Solver, error) {
	// Query the environment for the connection parameters.
	url := os.Getenv("DW_INTERNAL__HTTPLINK")
	token := os.Getenv("DW_INTERNAL__TOKEN")
	var proxy *string
	if strp, found := os.LookupEnv("DW_INTERNAL__HTTPPROXY"); found {
		proxy = &strp
	}

	// Establish a connection to either a remote or local solver.
	var conn *Connection
	var err error
	if url == "" || token == "" {
		conn = LocalConnection()
	} else {
		conn, err = RemoteConnection(url, token, proxy)
		if err != nil {
			return nil, err
		}
	}

	// Return the specified solver.
	sName := os.Getenv("DW_INTERNAL__SOLVER")
	if sName == "" {
		return nil, newErrorf(C.SAPI_ERR_INVALID_PARAMETER, "A solver must be named via the DW_INTERNAL__SOLVER environment variable")
	}
	return conn.Solver(sName)
}
