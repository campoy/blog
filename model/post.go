package model

import (
	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
)

// A Post
type Post struct {
	Title    string
	Text     string
	Comments []Comment `datastore:"-"`
	Key      string    `datastore:"-"`
	Creation time.Time
	Author   string
}

func NewPost(title, text string) *Post {
	return &Post{
		Creation: time.Now(),
		Title:    title,
		Text:     text,
	}
}

func (p *Post) Save(c appengine.Context) error {
	p.Author = userEmail(c)
	k := datastore.NewKey(c, "Post", "", 0, userKey(c))
	k, err := datastore.Put(c, k, p)
	p.Key = k.Encode()
	return err
}

func (p *Post) FetchComments(c appengine.Context) error {
	var all, mine []Comment
	errc := make(chan error, 2)
	go func() {
		cs, err := fetchComments(c, p.Key, nil)
		all = cs
		errc <- err
	}()
	go func() {
		cs, err := fetchComments(c, p.Key, userKey(c))
		mine = cs
		errc <- err
	}()
	for i := 0; i < 2; i++ {
		if err := <-errc; err != nil {
			return err
		}
	}
	p.Comments = nil
	mergeComments(&p.Comments, all, mine)
	return nil
}

// min returns the minimum value of a list of numbers.
func min(n int, rest ...int) int {
	m := n
	for _, v := range rest {
		if v < m {
			m = v
		}
	}
	return m
}

// mergePosts merges two slices of Post sorted by Creation
// time into a new slice with the n newest ones.
// It removes duplicates by giving priority to the
// elements in the first slice.
func mergePosts(m *[]Post, a, b []Post, n int) {
	for l := n - len(*m); l > 0; l = n - len(*m) {
		if len(a) == 0 {
			*m = append(*m, b[:min(l, len(b))]...)
			return
		}
		if len(b) == 0 {
			*m = append(*m, a[:min(l, len(a))]...)
			return
		}
		switch {
		case a[0].Key == b[0].Key:
			*m, a, b = append(*m, a[0]), a[1:], b[1:]
		case a[0].Creation.Before(b[0].Creation):
			*m, b = append(*m, b[0]), b[1:]
		default:
			*m, a = append(*m, a[0]), a[1:]
		}
	}
}

// FetchPosts fetches the n newest posts.
func FetchPosts(c appengine.Context, n int) ([]Post, error) {
	all, err := fetchPosts(c, n, nil)
	if err != nil {
		return nil, err
	}
	mine, err := fetchPosts(c, n, userKey(c))
	if err != nil {
		return nil, err
	}
	m := []Post{}
	mergePosts(&m, mine, all, n)

	// We fetch the comments for all posts concurrently.
	errc := make(chan error, len(m))
	for i := range m {
		go func(p *Post) {
			errc <- p.FetchComments(c)
		}(&m[i])
	}
	for _ = range m {
		if err := <-errc; err != nil {
			return nil, err
		}
	}
	return m, nil
}

// fetchPosts fetches the n newest posts.
// If the user key is not nil, the key is used to limit the list
// of fetched Posts to the descendants of that user.
// If the user key is nil the list of Posts will be fetched from memcache,
// if available, and stored into memcache otherwise.
func fetchPosts(c appengine.Context, n int, user *datastore.Key) ([]Post, error) {
	p := make([]Post, 0, n)
	q := datastore.NewQuery("Post").
		Order("-Creation").
		Limit(n)
	if user != nil {
		q = q.Ancestor(user)
	} else {
		mk := "posts"
		_, err := memcache.JSON.Get(c, mk, &p)
		if err == nil {
			return p, nil
		}
		defer func() {
			memcache.JSON.Set(c, &memcache.Item{
				Key:        mk,
				Object:     &p,
				Expiration: 5 * time.Second,
			})
		}()
	}
	ks, err := q.GetAll(c, &p)
	if err != nil {
		return nil, fmt.Errorf("query posts: %v", err)
	}
	for i, k := range ks {
		p[i].Key = k.Encode()
	}
	return p, nil
}

// FetchPostsForUser fetches the n newest posts for the specified user.
func FetchPostsForUser(c appengine.Context, n int, usr string) ([]Post, error) {
	return fetchPosts(c, n, datastore.NewKey(c, "User", usr, 0, nil))
}
