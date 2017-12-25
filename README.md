sapi: Go bindings for D-Wave's SAPI library
===========================================

[![Go Report Card](https://goreportcard.com/badge/github.com/lanl/sapi)](https://goreportcard.com/report/github.com/lanl/sapi)
[![GoDoc](https://godoc.org/github.com/lanl/sapi?status.svg)](https://godoc.org/github.com/lanl/sapi)

Description
-----------

The main low-level interface to the quantum annealers produced by [D-Wave Systems, Inc.](https://www.dwavesys.com/) is D-Wave's Solver Application Programming Interface (SAPI) library.  D-Wave provides access to SAPI from C, Python, and MATLAB.  This package extends that set to include the [Go](https://golang.org/) programming language.

Installation
------------

Download, build, and install `sapi` like any other Go package:
```bash
go get github.com/lanl/sapi
```

The build process assumes that the C compiler can find the `dwave_sapi.h` header file and the `libdwave_sapi.so` library file.  (These are proprietary files provided by D-Wave.  If you don't have them, I can't give them to you.)

Documentation
-------------

The package documentation can be found online via [GoDoc](https://godoc.org/github.com/lanl/sapi).  The main source of documentation for SAPI in general is D-Wave's *Developer Guide for Python* (or C or MATLAB).

Limitations
-----------

The `sapi` package provides a useful but incomplete subset of SAPI.  If you find your favorite SAPI function missing from `sapi`, go ahead and submit a pull request or open an issue on GitHub.

License
-------

`sapi` is provided under a BSD-ish license with a "modifications must be indicated" clause.  See [the LICENSE file](https://github.com/lanl/sapi/blob/master/LICENSE.md) for the full text.

`sapi` is part of the Hybrid Quantum-Classical Computing suite, known internally as LA-CC-16-032.

Author
------

Scott Pakin, <pakin@lanl.gov>
