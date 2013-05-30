// Test me with "aet ./samples/with-rpc-stub/myapp"
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
