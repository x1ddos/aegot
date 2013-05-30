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

func put(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	item := Item{Id: r.URL.Path[1:], Name: r.FormValue("name")}
	switch err := item.put(c); err {
	case nil:
		fmt.Fprintln(w, item.Name)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func init() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			get(w, r)
		case "POST", "PUT":
			put(w, r)
		default:
			http.Error(w, "", http.StatusBadRequest)
		}
	})
}
