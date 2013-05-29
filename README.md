App Engine for Go Testing
=====

Utils for running tests on packages that import (directly or indirectly) "appengine/*" packages.

Why
===

Let's say you have a Go app that looks something like this:

```go
package myapp

import (
  "appengine"
  "appengine/datastore"
)

func listTopics(w http.ResponseWriter, r *http.Request) {
  ctx := appengine.NewContext(r)
  // do something with ctx, e.g. use datastore to fetch some entities
}
```

and you have myapp_test.go:

```go
package myapp

import "testing"

func TestListTopics(t *testing.T) {
  // do some tests here
}
```

Well, if you try running `go test ./myapp` it won't even get to try running the tests because Go won't be able to build your app & tests. It'll say something like "Can't import 'appengine'" package. That's because "appengine/*" are indeed in a different location (specifically, in SDK/goroot/src/pkg/).

You could symlink SDK/goroot/src/pkg/appengine to your GOROOT/src/appengine but that probably won't solve the problem:

  - go test won't be able to build appengine package (so that later tests would run quicker)
  - SDK contains the whole Go release, but slightly modified, so you'll definitely bump into issues like "Undefined os.DisableWritesForAppEngine" because indeed, DisableWritesForAppEngine exists only in this specific Go version for App Engine.

So, this little project tries to solve these problems.

Instead of using built Go version of the SDK you've probably downloaded from https://developers.google.com/appengine/downloads, this tool will:

  # clone the original source files from code.google.com/p/appengine-go
  # apply a patch to one specific file (appengine_internal/api_dev.go)
  # build appengine packages to speedup later tests with "go test -i ..."

Usage
===

Let's say I'm in my app root which has a subdir called "myapp" (from the example above). You only need to do this once:

  # Install "aet" tool: `go get github.com/crhym3/aegot/aet`
  # Init: `aet init ./myapp`, which will do a couple things:
    - hg clone code.google.com/p/appengine-go
    - fetch a patched version of api_dev.go and overwrite the original file
    - tell Go to build appengine packages with "go test -i ./myapp" (if ./myapp argument was provided)

The sample test from the above will be able to run with `aet test ./myapp`, which actually doesn't do much. It only manipulates GOPATH env variable and adds appengine-go local clone path to it. So, alternatively, tests can be run with `GOPATH=$GOPATH:$GOPATH/appengine-go/src go test ./myapp`


Alternatives
===

You might also want to check out other projects:

  - https://github.com/tenntenn/gae-go-testing
  - https://github.com/najeira/testbed
