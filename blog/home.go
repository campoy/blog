package blog

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"appengine"
	"appengine/user"

	"github.com/campoy/blog/model"
)

var homeTmpl = template.Must(template.ParseFiles("templates/home.tmpl"))

type page struct {
	User      string
	LoginURL  string
	LogoutURL string
	Posts     []model.Post
}

func newPage(c appengine.Context) (*page, error) {
	var p page
	u := user.Current(c)
	if u == nil {
		url, err := user.LoginURL(c, "/")
		if err != nil {
			return nil, fmt.Errorf("login url: %v", err)
		}
		p.LoginURL = url
		return &p, nil
	}

	p.User = u.Email
	url, err := user.LogoutURL(c, "/")
	if err != nil {
		return nil, fmt.Errorf("logout url: %v", err)
	}
	p.LogoutURL = url
	return &p, nil
}

func homeHandler(w io.Writer, r *http.Request) error {
	c := appengine.NewContext(r)
	p, err := newPage(c)
	if err != nil {
		return fmt.Errorf("new page: %v", err)
	}

	if len(p.User) > 0 {
		if path := r.URL.Path; len(path) > 1 {
			p.Posts, err = model.FetchPostsForUser(c, 10, path[1:])
		} else {
			p.Posts, err = model.FetchPosts(c, 10)
		}
		if err != nil {
			return fmt.Errorf("fetch posts: %v", err)
		}
	}

	return homeTmpl.Execute(w, p)
}

func postHandler(w io.Writer, r *http.Request, u *user.User) error {
	p := model.NewPost(
		r.FormValue("title"),
		r.FormValue("content"),
	)

	if err := p.Save(appengine.NewContext(r)); err != nil {
		return fmt.Errorf("save post: %v", err)
	}
	return redirectTo("/")
}

func commentHandler(w io.Writer, r *http.Request, u *user.User) error {
	c, err := model.NewComment(
		r.FormValue("comment"),
		r.FormValue("post-key"),
	)
	if err != nil {
		return fmt.Errorf("new comment: %v", err)
	}

	if c.Save(appengine.NewContext(r)); err != nil {
		return fmt.Errorf("save comment: %v", err)
	}
	return redirectTo("/")
}

type redirectTo string

func (r redirectTo) Error() string { return string(r) }

type handler func(io.Writer, *http.Request) error

func (f handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := &bytes.Buffer{}
	err := f(b, r)
	if err != nil {
		if red, ok := err.(redirectTo); ok {
			http.Redirect(w, r, string(red), http.StatusMovedPermanently)
			return
		}
		appengine.NewContext(r).Errorf("request failed: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(b.Bytes())
}

type authHandler func(io.Writer, *http.Request, *user.User) error

func (f authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := handler(func(w io.Writer, r *http.Request) error {
		c := appengine.NewContext(r)
		u := user.Current(c)

		if u == nil {
			return fmt.Errorf("sign in first")
		}
		if err := model.SaveUser(c); err != nil {
			return fmt.Errorf("save user: %v", err)
		}

		return f(w, r, u)
	})
	h.ServeHTTP(w, r)
}

func init() {
	http.Handle("/", handler(homeHandler))
	http.Handle("/post", authHandler(postHandler))
	http.Handle("/comment", authHandler(commentHandler))
}
