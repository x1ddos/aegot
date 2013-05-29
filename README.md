App Engine for Go Testing
=====

Utils for testing apps that import (directly or indirectly) "appengine/*" packages.

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

Well, if you try executing `go test ./myapp` it won't even get to running the tests because Go won't be able to build your app & tests. It'll say something like "Can't import 'appengine'" package. That's because "appengine/*" packages are indeed in a different location (specifically, in SDK/goroot/src/pkg/).

You could symlink SDK/goroot/src/pkg/appengine to your GOPATH/src/appengine but that probably won't solve all the problems:

  - go test won't be able to build appengine package (so that later tests would run quicker)
  - SDK contains the whole Go release, but slightly modified, so you'll definitely bump into issues like "Undefined os.DisableWritesForAppEngine" because indeed, DisableWritesForAppEngine exists only in this specific Go version for App Engine.

So, this little project tries to solve these problems.

Instead of using Go version of App Engine for Go SDK, this tool will:

  * clone the original source files from code.google.com/p/appengine-go
  * apply a patch to one specific file (appengine\_internal/api\_dev.go)
  * build appengine packages to speedup later tests with "go test -i ..."


Usage
===

Let's say I'm in my app root which has a subdir called "myapp" (from the example above). You only need to do this once:

  * Install "aet" tool: `go get github.com/crhym3/aegot/aet`
  * Init: `aet init ./myapp`, which will do a couple things:
    - hg clone code.google.com/p/appengine-go
    - fetch a patched version of api_dev.go and overwrite the original file
    - tell Go to build appengine packages with "go test -i ./myapp" (if ./myapp argument was provided)

Sample test from the above will be able to run now with `aet test ./myapp`, but "aet test" doesn't do much actually.
It only manipulates GOPATH env variable and adds appengine-go local clone path to it.
So, alternatively, tests can be run with e.g. `GOPATH=$GOPATH:$GOPATH/appengine-go/src go test ./myapp`

```sh
$ aet -h

Usage: aet {init|test} [flags] ./path/to/*_test.go
  -c="hg clone -u": command to clone the repo; don't specify rev, url or d here
  -d="/Users/alex/go/src/appengine-go": expect appengine-go sources to be in d/src; required
  -rev="1.8.0": App Engine release version or repo revision; required for init
  -uc="hg update -r": command to update previously clonned repo
  -url="https://code.google.com/p/appengine-go/": appengine-go project repository URL

```

Alternatives
===

You might also want to check out other projects:

  - https://github.com/tenntenn/gae-go-testing
  - https://github.com/najeira/testbed
