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
	"sort"
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

// Canonicalize ensures that each ProblemEntry in a given Problem has I ≤ J and
// that all {I. J} pairs are unique.
func (p Problem) Canonicalize() Problem {
	// Ensure that I ≤ J in each ProblemEntry.
	p1 := make(Problem, len(p))
	for i, pe := range p {
		if pe.I > pe.J {
			pe.I, pe.J = pe.J, pe.I
		}
		p1[i] = pe
	}

	// Sort the Problem by I then J.
	sort.Slice(p1, func(i, j int) bool {
		switch {
		case p1[i].I < p1[j].I:
			return true
		case p1[i].I > p1[j].I:
			return false
		default:
			return p1[i].J < p1[j].J
		}
	})

	// Merge duplicate {I, J} entries by summing their Values.
	p2 := make(Problem, 0, len(p1))
	for i, pe := range p1 {
		if i > 0 && pe.I == p1[i-1].I && pe.J == p1[i-1].J {
			p2[len(p2)-1].Value += pe.Value
		} else {
			p2 = append(p2, pe)
		}
	}
	return p2
}

// couplerMap returns a map from a spin to a list of all ProblemEntry structs
// that couple that spin.
func (p Problem) couplerMap() map[int][]ProblemEntry {
	cMap := make(map[int][]ProblemEntry, len(p))
	for _, pe := range p {
		// Skip field weights.
		i, j := pe.I, pe.J
		if i == j {
			continue
		}

		// Store I --> J entries.
		pes, found := cMap[i]
		if !found {
			pes = make([]ProblemEntry, 0, 1)
		}
		cMap[i] = append(pes, pe)

		// Store J --> I entries.
		pe.I, pe.J = pe.J, pe.I
		pes, found = cMap[j]
		if !found {
			pes = make([]ProblemEntry, 0, 1)
		}
		cMap[j] = append(pes, pe)
	}
	return cMap
}

// energyOffset returns the difference in energy between a QUBO and an
// Ising-model problem.
func (p Problem) energyOffset() float64 {
	he := 0.0
	Je := 0.0
	for _, pe := range p {
		if pe.I == pe.J {
			he += pe.Value
		} else {
			Je += pe.Value
		}
	}
	return he/2.0 + Je/4.0
}

// ToIsing converts a QUBO problem to an Ising-model problem.  It additionally
// returns an energy offset to add to each solution's energy.
func (p Problem) ToIsing() (Problem, float64) {
	ip := make(Problem, 0, len(p))
	cp := p.Canonicalize()
	cMap := cp.couplerMap()
	for _, pe := range cp {
		if pe.I == pe.J {
			// Convert a field weight.
			v := 0.0
			for _, elt := range cMap[pe.I] {
				v += elt.Value
			}
			pe.Value = pe.Value/2.0 + v/4.0
		} else {
			// Convert a coupler strength.
			pe.Value /= 4.0
		}
		ip = append(ip, pe)
	}
	return ip, cp.energyOffset()
}

// ToQubo converts an Ising-model problem to a QUBO problem.  It additionally
// returns an energy offset to add to each solution's energy.
func (p Problem) ToQubo() (Problem, float64) {
	qp := make(Problem, 0, len(p))
	cp := p.Canonicalize()
	cMap := cp.couplerMap()
	for _, pe := range cp {
		if pe.I == pe.J {
			// Convert a field weight.
			v := 0.0
			for _, elt := range cMap[pe.I] {
				v += elt.Value
			}
			pe.Value = pe.Value*2.0 - v*2.0
		} else {
			// Convert a coupler strength.
			pe.Value *= 4.0
		}
		qp = append(qp, pe)
	}
	return qp, -qp.energyOffset()
}
