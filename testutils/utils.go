// Used only for testing:
// +build !appengine

// Although most of the methods here are proxies to appengine_internal methods,
// it is better to use these instead of invoking appengine_internal directly.
// 
// App Engine internals might change over time (even frequently) so if your
// tests use methods from this package you'll never have to change them.
package testutils

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"appengine"

	aei "appengine_internal"
	"code.google.com/p/goprotobuf/proto"
)

// SetDevAppServer can make an app think it's running on production servers.
func SetDevAppServer(dev bool) {
	aei.IsDev = dev
}

// CreateTestContext creates and registers a new appengine.Context associated
// with the request so that subsequent calls to appentine.NewContext(r) anywhere
// in the code will successfully return the context.
// 
// The caller is responsible to invoke DeleteTestContext(ctx) at the end of
// a test.
func CreateTestContext(r *http.Request) appengine.Context {
	return aei.CreateContext(r, nil)
}

// DeleteTestContext unregisters a context associated with the request.
// Subsequent calls to appengine.NewContext(r) will panic.
func DeleteTestContext(r *http.Request) {
	aei.DeleteContext(r)
}

// NewTestRequest creates http.Request and appengine.Context associated with
// the request. It panics if the request cannot be created.
// 
// Returns the newly created request and a function that removes associated
// context. The caller is responsible to invoke this function at the end of a
// test.
func NewTestRequest(method, path string, body []byte) (*http.Request, func()) {
	var buf io.Reader
	if body != nil {
		buf = bytes.NewBuffer(body)
	}
	req, err := http.NewRequest(method, path, buf)
	if err != nil {
		panic(err)
	}
	CreateTestContext(req)
	return req, func() {
		DeleteTestContext(req)
	}
}

// RpcStubFunc is a function type that replaces an API RPC implementation.
type RpcStubFunc func(in, out proto.Message, opts *RpcCallOptions) error

// RpcCallOptions is the equivalent of appengine_internal.CallOptions.
type RpcCallOptions struct {
	Timeout time.Duration // if non-zero, overrides RPC default
}

// RegisterAPIOverride replaces (stubs out) the implementation of an API RPC
// call. The caller is responsible to unregister the override at the end of a
// test.
// 
// Returns a function that can unregister the stub. Here's an example:
// 		
// 		func TestSomething(t *testing.T) {
// 			rpcStub = func(in, out proto.Message, *RpcCallOptions) error {
// 				req := in.(pb.SomeProtoRequest)
// 				resp := out.(pb.SomeProtoResponse)
// 				// do something with req / resp or return an error
// 			}
// 			unregister = RegisterAPIOverride("user", "SomeRpcMethod", rpcStub)
// 			defer unregister()
// 			
// 			// test code that (probably indirectly) calls "user.SomeRpcMethod"
// 		}
// 		
func RegisterAPIOverride(service, method string, f RpcStubFunc) func() {
	proxy := func(in, out proto.Message, opts *aei.CallOptions) error {
		var o *RpcCallOptions
		if opts != nil {
			o = &RpcCallOptions{Timeout: opts.Timeout}
		}
		return f(in, out, o)
	}
	aei.RegisterAPIOverride(service, method, proxy)
	return func() {
		UnregisterAPIOverride(service, method)
	}
}

// UnregisterAPIOverride removes stubbed API RPC implementation from registered
// overrides.
func UnregisterAPIOverride(service, method string) {
	aei.UnregisterAPIOverride(service, method)
}
