package model

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
)

type User struct{}

func SaveUser(c appengine.Context) error {
	_, err := datastore.Put(c, userKey(c), &User{})
	return err
}

func userEmail(c appengine.Context) string {
	return user.Current(c).Email
}
func userKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "User", userEmail(c), 0, nil)
}

func save(c appengine.Context, kind string, v interface{}) error {
	k := datastore.NewKey(c, kind, "", 0, userKey(c))
	_, err := datastore.Put(c, k, v)
	return err
}
