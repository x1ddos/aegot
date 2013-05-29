// Used only for testing
// +build !appengine

package testutils

import (
	"net/http"
	"time"

	"appengine"

	aei "appengine_internal"
	"code.google.com/p/goprotobuf/proto"
)

type RpcStubFunc func(in, out proto.Message, opts *RpcCallOptions) error

type RpcCallOptions struct {
	Timeout time.Duration // if non-zero, overrides RPC default
}

// SetDevAppServer can make an app think it's running on production servers.
func SetDevAppServer(dev bool) {
	aei.IsDev = dev
}

func RegisterAPIOverride(service, method string, f RpcStubFunc) {
	proxy := func(in, out proto.Message, opts *aei.CallOptions) error {
		var o *RpcCallOptions
		if opts != nil {
			o = &RpcCallOptions{Timeout: opts.Timeout}
		}
		return f(in, out, o)
	}
	aei.RegisterAPIOverride(service, method, proxy)
}

func CreateTestContext(r *http.Request) appengine.Context {
	return aei.CreateContext(r, nil)
}

func DeleteTestContext(r *http.Request) {
	aei.DeleteContext(r)
}
