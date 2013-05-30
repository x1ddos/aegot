# Testing with Google App Engine for Go

Utils for testing apps that import (directly or indirectly) "appengine/*" packages.

## Why

---

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

  * clone the original source files from [code.google.com/p/appengine-go][4]
  * apply a patch to a couple files in appengine_internal/ dir
  * build appengine packages to speedup later tests with "go test -i ..."


## Usage

---

Let's say I'm in my app root which has a subdir called "myapp" (from the example above). You only need to do this once:

  * Install "aet" tool: `go get github.com/crhym3/aegot/aet`
  * Init: `aet init ./myapp`, which [will do a couple things][7]:
    - hg clone code.google.com/p/appengine-go
    - fetch patched versions of appengine\_internal/ and overwrite the original files
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

## Testutils

---

There's small colletion of methods that work sort of like proxies to
appengine_internal which you can use to stub out App Engine internal RPCs.

For example, let's say we have `myapp/handlers.go` with the following content:

```go
package myapp

import (
  "fmt"
  "net/http"

  "appengine"
  "appengine/datastore"
)

type Item struct {
  Id   string `datastore:"-"`
  Name string
}

func get(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  item := &Item{Id: r.URL.Path[1:]}
  switch err := item.get(c); err {
  case nil:
    fmt.Fprint(w, item.Name)
  case datastore.ErrNoSuchEntity:
    http.NotFound(w, r)
  default:
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}
```

and `myapp/items.go` (note the "// +build ..." tag):

```go
// +build appengine

package myapp

import (
  "appengine"
  "appengine/datastore"
)

func (item *Item) get(c appengine.Context) error {
  if item.Id == "" {
    return datastore.ErrNoSuchEntity
  }
  key := datastore.NewKey(c, "Item", item.Id, 0, nil)
  return datastore.Get(c, key, item)
}
```

If we wanted to just test handlers, we could stub the code in items.go with
items\_stub.go (again, note the "// +build ..." tag):

```go
// +build !appengine

package myapp

import (
  "errors"

  "appengine"
  "appengine/datastore"
)

func (item *Item) get(c appengine.Context) error {
  switch item.Id {
  case "does-not-exist":
    return datastore.ErrNoSuchEntity
  case "error":
    return errors.New("Some fake get error")
  default:
    item.Name = item.Id
  }
  return nil
}
```

and, assuming you installed `aet` and did `aet init`, our handlers_test.go
could look like this:

```go
package myapp

import (
  "net/http"
  "net/http/httptest"
  "testing"

  tu "github.com/crhym3/aegot/testutils"
)

func TestGetOk(t *testing.T) {
  const itemId = "valid-id"

  r, deleteContext := tu.NewTestRequest("GET", "/"+itemId, nil)
  defer deleteContext()
  w := httptest.NewRecorder()

  get(w, r)

  if w.Code != http.StatusOK {
    t.Errorf("Expected 200, got %d", w.Code)
  }
  body := string(w.Body.Bytes())
  if body != itemId {
    t.Errorf("Expected %q, got %q", itemId, body)
  }
}

func TestGetErrors(t *testing.T) {
  tt := []*struct {
    path string
    code int
  }{
    {"/does-not-exist", 404},
    {"/error", 500},
  }
  for _, ti := range tt {
    r, deleteContext := tu.NewTestRequest("GET", ti.path, nil)
    defer deleteContext()
    w := httptest.NewRecorder()

    get(w, r)

    if w.Code != ti.code {
      t.Errorf("Expected %d, got %d", ti.code, w.Code)
    }
  }
}
```

So, that was easy.
Now, immagine that you needed to test code in items.go for
some reason. Well, you could do that by stubbing out "datastore\_v3" service
methods. For instance, items_test.go:

```go
package myapp

import (
  "testing"

  "appengine"  
  "code.google.com/p/goprotobuf/proto"
  pb "appengine_internal/datastore"

  tu "github.com/crhym3/aegot/testutils"
)

func TestPutItem(t *testing.T) {
  const (
    itemId   = "some-id"
    itemName = "test"
  )

  putStub := func(in, out proto.Message, _ *tu.RpcCallOptions) error {
    req := in.(*pb.PutRequest)

    if len(req.GetEntity()) != 1 {
      t.Error("Expected 1 entity, got %d", len(req.GetEntity()))
    }
    ent := req.GetEntity()[0]
    id := ent.GetKey().GetPath().GetElement()[0].GetName()
    if id != itemId {
      t.Error("Expected ID %q, got %q", itemId, id)
    }
    if len(ent.GetProperty()) != 1 {
      t.Error("Expected 1 property, got: %d", len(ent.GetProperty()))
    }
    prop := ent.GetProperty()[0]
    if prop.GetName() != "Name" {
      t.Error("Invalid property name: %q", prop.GetName())
    }
    val := prop.GetValue().GetStringValue()
    if val != itemName {
      t.Error("Expected %q, got %q", itemName, val)
    }

    resp := out.(*pb.PutResponse)
    resp.Key = []*pb.Reference{ent.GetKey()}
    return nil
  }
  unregister := tu.RegisterAPIOverride("datastore_v3", "Put", putStub)
  defer unregister()

  r, deleteContext := tu.NewTestRequest("PUT", "/"+itemId, nil)
  defer deleteContext()

  item := Item{Id: itemId, Name: itemName}
  // appengine.NewContext() will use the one created in NewTestRequest() above
  if err := item.put(appengine.NewContext(r)); err != nil {
    t.Error(err)
  }
}
```

Note that in this case we don't use "// +build ..." tags because we want to
test the actual code in items.go.

For more examples see:

* [samples dir][2]
* [tests in go-endpoints][3]

[Testutils full documentation][1].


## Alternatives

---

You might also want to check out other projects:

  - [github.com/tenntenn/gae-go-testing][5]
  - [github.com/najeira/testbed][6]


[1]: http://godoc.org/github.com/crhym3/aegot/testutils
[2]: https://github.com/crhym3/aegot/tree/master/samples
[3]: https://github.com/crhym3/go-endpoints/tree/master/endpoints
[4]: http://code.google.com/p/appengine-go
[5]: https://github.com/tenntenn/gae-go-testing
[6]: https://github.com/najeira/testbed
[7]: https://github.com/crhym3/aegot/tree/master/aet/initsrc.go
