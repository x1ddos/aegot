// +build appengine

package myapp

import (
	"errors"

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

func (item *Item) put(c appengine.Context) (err error) {
	if item.Id == "" {
		return errors.New("Invalid item ID")
	}
	if item.Name == "" {
		return errors.New("Invalid item Name")
	}
	key := datastore.NewKey(c, "Item", item.Id, 0, nil)
	_, err = datastore.Put(c, key, item)
	return
}
