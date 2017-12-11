// This file presents an interface to SAPI connection-related types and
// functions.

package sapi

// #cgo LDFLAGS: -ldwave_sapi
// #include <stdio.h>
// #include <stdlib.h>
// #include <dwave_sapi.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// A Connection represents a connection to a remote solver.
type Connection struct {
	conn  *C.sapi_Connection // SAPI connection object
	URL   string             // Connection name
	Token string             // Token to authenticate a user
	Proxy string             // Proxy URL
}

// LocalConnection returns a connection to the set of local solvers (i.e.,
// simulators).
func LocalConnection() *Connection {
	conn := C.sapi_localConnection()
	return &Connection{
		conn:  conn,
		URL:   "",
		Token: "",
		Proxy: "",
	}
}

// RemoteConnection establishes a connection to a set of remote solvers (i.e.,
// D-Wave hardware).
func RemoteConnection(url, token, proxy string) (*Connection, error) {
	// Establish a connection.
	var conn *C.sapi_Connection
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))
	cProxy := C.CString(proxy)
	defer C.free(unsafe.Pointer(cProxy))
	cErr := make([]C.char, C.SAPI_ERROR_MESSAGE_MAX_SIZE)
	ret := C.sapi_remoteConnection(cURL, cToken, cProxy, &conn, &cErr[0])
	if ret != C.SAPI_OK {
		return nil, newErrorf(ret, "%s", C.GoString(&cErr[0]))
	}
	connObj := &Connection{
		conn:  conn,
		URL:   url,
		Token: token,
		Proxy: proxy,
	}

	// Free the connection when it gets GC'd away.
	runtime.SetFinalizer(connObj, func(c *Connection) {
		if c.conn != nil {
			C.sapi_freeConnection(c.conn)
			c.conn = nil
		}
	})
	return connObj, nil
}

// Solvers returns a list of all solvers available on the current connection.
func (c *Connection) Solvers() ([]string, error) {
	cList := C.sapi_listSolvers(c.conn)
	if cList == nil {
		return nil, newErrorf(C.SAPI_ERR_INVALID_PARAMETER, "Failed to retrieve the solver list")
	}
	list := make([]string, 0, 2)
	lPtr := (*[1 << 30]*C.char)(unsafe.Pointer(cList))
	for _, cp := range lPtr {
		if cp == nil {
			break
		}
		list = append(list, C.GoString(cp))
	}
	return list, nil
}
