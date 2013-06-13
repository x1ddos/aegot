// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package appengine_internal provides support for package appengine.
//
// Programs should not use this package directly. Its API is not stable.
// Use packages appengine and appengine/* instead.
package appengine_internal

// This package's implementation differs when running on a development App
// Server on a local machine and when running on an actual App Engine App
// Server in production, but that is a private implementation detail. The
// exported API is the same.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"code.google.com/p/goprotobuf/proto"
)

var (
	addrHTTP = flag.String("addr_http", "", "net:laddr to listen on for HTTP requests.")
	addrAPI  = flag.String("addr_api", "", "net:raddr to dial for API requests.")
)

// ProtoMessage is the same as proto.Message. It is defined here because user
// code cannot import package proto.
type ProtoMessage interface {
	Reset()
	String() string
	ProtoMessage()
}

var _ ProtoMessage = proto.Message(ProtoMessage(nil))

type ServeHTTPFunc func(netw, addr string)

var serveHTTPFunc ServeHTTPFunc

func RegisterHTTPFunc(f ServeHTTPFunc) {
	serveHTTPFunc = f
}

type CallOptions struct {
	Timeout time.Duration // if non-zero, overrides RPC default
}

// errorCodeMaps is a map of service name to the error code map for the service.
var errorCodeMaps = make(map[string]map[int32]string)

// RegisterErrorCodeMap is called from API implementations to register their
// error code map. This should only be called from init functions.
func RegisterErrorCodeMap(service string, m map[int32]string) {
	errorCodeMaps[service] = m
}

// APIError is the type returned by appengine.Context's Call method
// when an API call fails in an API-specific way. This may be, for instance,
// a taskqueue API call failing with TaskQueueServiceError::UNKNOWN_QUEUE.
type APIError struct {
	Service string
	Detail  string
	Code    int32 // API-specific error code
}

func (e *APIError) Error() string {
	if e.Code == 0 {
		if e.Detail == "" {
			return "APIError <empty>"
		}
		return e.Detail
	}
	s := fmt.Sprintf("API error %d", e.Code)
	if m, ok := errorCodeMaps[e.Service]; ok {
		s += " (" + e.Service + ": " + m[e.Code] + ")"
	} else {
		// Shouldn't happen, but provide a bit more detail if it does.
		s = e.Service + " " + s
	}
	if e.Detail != "" {
		s += ": " + e.Detail
	}
	return s
}

// CallError is the type returned by appengine.Context's Call method when an
// API call fails in a generic way, such as APIResponse::CAPABILITY_DISABLED.
type CallError struct {
	Detail string
	Code   int32
}

func (e *CallError) Error() string {
	var msg string
	switch e.Code {
	case 0: // OK
		return e.Detail
	case 4: // OVER_QUOTA
		msg = "Over quota"
	case 6: // CAPABILITY_DISABLED
		msg = "Capability disabled"
	case 9: // BUFFER_ERROR
		msg = "Buffer error"
	case 11: // CANCELLED
		msg = "Canceled"
	default:
		msg = fmt.Sprintf("Call error %d", e.Code)
	}
	return msg + ": " + e.Detail
}

// handleHealthCheck handles health check HTTP requests from the App Server.
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

// parseAddr parses a composite address of the form "net:addr".
func parseAddr(compAddr string) (net, addr string) {
	parts := strings.SplitN(compAddr, ":", 2)
	if len(parts) != 2 {
		log.Panicf("appengine: bad composite address %q", compAddr)
	}
	return parts[0], parts[1]
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("http.DefaultTransport and http.DefaultClient are not available in App Engine. " +
		"See https://developers.google.com/appengine/docs/go/urlfetch/overview")
}

func init() {
	// http.DefaultTransport doesn't work in production so break it
	// explicitly so it fails the same way in both dev and prod
	// (and with a useful error message)
	http.DefaultTransport = failingTransport{}
}

// appPackagesInitialized is closed at the start of Main, after all app packages
// have been initialized
var appPackagesInitialized = make(chan struct{})

// Main is designed so that the complete generated main.main package is:
//
//	package main
//
//	import (
//		"path/to/appengine_internal"
//		_ "myapp/package0"
//		_ "myapp/package1"
//	)
//
//	func main() {
//		appengine_internal.Main()
//	}
//
// The "myapp/packageX" packages are expected to register HTTP handlers
// in their init functions.
func Main() {
	var httpNet, httpAddr, apiNet, apiAddr string

	close(appPackagesInitialized)

	// Check flags.
	flag.Parse()
	if !IsDevAppServer() {
		if *addrHTTP == "" || *addrAPI == "" {
			log.Panic("appengine_internal.Main called without address flags.")
		}
		httpNet, httpAddr = parseAddr(*addrHTTP)
		apiNet, apiAddr = parseAddr(*addrAPI)
	}

	// Forward App Engine API calls to the appserver.
	initAPI(apiNet, apiAddr)

	// Serve HTTP requests forwarded from the appserver to us.
	http.HandleFunc("/_appengine_delegate_health_check", handleHealthCheck)
	if serveHTTPFunc == nil {
		log.Panic("appengine: no ServeHTTPFunc registered.")
	}
	serveHTTPFunc(httpNet, httpAddr)
}

// NamespaceMods is a map from API service to a function that will mutate an RPC request to attach a namespace.
// The function should be prepared to be called on the same message more than once; it should only modify the
// RPC request the first time.
var NamespaceMods = make(map[string]func(m proto.Message, namespace string))

// apiOverrides is a map of replacements for the implementation of API RPC calls.
var apiOverrides = make(map[struct{ service, method string }]func(proto.Message, proto.Message, *CallOptions) error)

func RegisterAPIOverride(service, method string, f func(proto.Message, proto.Message, *CallOptions) error) {
	apiOverrides[struct{ service, method string }{service, method}] = f
}

func UnregisterAPIOverride(service, method string) {
	delete(apiOverrides, struct{ service, method string }{service, method})
}
