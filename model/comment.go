package model

import (
	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
)

// A Comment contains the information related to a comment done on a post at a
// given moment by an author.
type Comment struct {
	Text     string
	PostKey  string
	Key      string `datastore:"-"`
	Creation time.Time
	Author   string
}

// NewComment returns a new Comment given a text and the key to a post.
// It returns an error if the key is not valid.
func NewComment(text, postKey string) (*Comment, error) {
	if _, err := datastore.DecodeKey(postKey); err != nil {
		return nil, fmt.Errorf("wrong post key: %v", err)
	}

	return &Comment{
		Creation: time.Now(),
		Text:     text,
		PostKey:  postKey,
	}, nil
}

// Save puts the Comment in the datastore.
func (cm *Comment) Save(c appengine.Context) error {
	cm.Author = userEmail(c)
	k := datastore.NewKey(c, "Comment", "", 0, userKey(c))
	k, err := datastore.Put(c, k, cm)
	cm.Key = k.Encode()
	return err
}

// fetchComments fetches all the comments for the given post key.
// If the user key is not nil, the key is used to limit the list
// of fetched Comment to the descendants of that user.
func fetchComments(c appengine.Context, postKey string, user *datastore.Key) ([]Comment, error) {
	cs := []Comment{}
	q := datastore.NewQuery("Comment").
		Filter("PostKey =", postKey).
		Order("Creation")
	if user != nil {
		q = q.Ancestor(user)
	} else {
		mk := "comments-" + postKey
		_, err := memcache.JSON.Get(c, mk, &cs)
		if err == nil {
			return cs, nil
		}
		defer func() {
			memcache.JSON.Set(c, &memcache.Item{
				Key:        mk,
				Object:     &cs,
				Expiration: 5 * time.Second,
			})
		}()
	}
	ks, err := q.GetAll(c, &cs)
	if err != nil {
		return nil, fmt.Errorf("query comments: %v", err)
	}
	for i, k := range ks {
		cs[i].Key = k.Encode()
	}
	return cs, nil
}

// mergeComments merges two slices of Comment sorted by Creation
// time into a new sorted slice.
// It removes duplicates by giving priority to the
// elements in the first slice.
func mergeComments(m *[]Comment, a, b []Comment) {
	for {
		if len(a) == 0 {
			*m = append(*m, b...)
			return
		}
		if len(b) == 0 {
			*m = append(*m, a...)
			return
		}
		switch {
		case a[0].Key == b[0].Key:
			*m, a, b = append(*m, a[0]), a[1:], b[1:]
		case a[0].Creation.After(b[0].Creation):
			*m, b = append(*m, b[0]), b[1:]
		default:
			*m, a = append(*m, a[0]), a[1:]
		}
	}
}
