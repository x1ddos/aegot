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

func (item *Item) put(c appengine.Context) error {
	if item.Id == "error" {
		return errors.New("Some fake put error")
	}
	return nil
}
