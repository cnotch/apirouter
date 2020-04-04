// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package apirouter provides a lightning fast RESTful api router.
//
// A trivial example is:
//
//  package main
//
//  import (
//      "fmt"
//      "github.com/cnotch/apirouter"
//      "net/http"
//      "log"
//  )
//
//  func Index(w http.ResponseWriter, r *http.Request, _ apirouter.Params) {
//      fmt.Fprint(w, "Welcome!\n")
//  }
//
//  func Hello(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
//      fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
//  }
//
//  func main() {
//      router := apirouter.New(
//			apirouter.GET("/", Index),
//			apirouter.GET("/hello/:name", Hello),
//		)
//
//      log.Fatal(http.ListenAndServe(":8080", router))
//  }
//
// The registered pattern, against which the router matches incoming requests, can
// contain three types of parameters(default style):
// 	Syntax                      Type
//  :name                       named parameter
//  :name=regular-expressions   regular expression parameter
//  *name                       wildcard parameter
//
// Matching priority, on the example below the router will test the routes
// in the following order, /users/list then /users/:id=^\d+$
// then /users/:id then /users/*page.
//
// 	r:= apirouter.New(
// 		apirouter.GET("/users/:id",...),
// 		apirouter.GET(`/users/:id=^\d+$`,...),
// 		apirouter.GET("/users/*page",...),
// 		apirouter.GET("/users/list",...),
// 	)
//
// Named parameters are dynamic path segments. They match anything until the
// next '/' or the path end:
//  Pattern: /blog/:category/:post
//
//  Requests:
//   /blog/go/request-routers            match: category="go", post="request-routers"
//   /blog/go/request-routers/           no match
//   /blog/go/                           no match
//   /blog/go/request-routers/comments   no match
//
// If a parameter must match an exact pattern (digits only,
// for example), you can also set a regular expression constraint
// just after the parameter name and `=`.
//  Pattern: /users/:id=^\d+$
//
//  Requests:
//   /users/123            match: id="123"
//   /users/admin          no match
//   /users/123/           no match
//
// Wildcard parameters match anything until the path end, not including the
// directory index (the '/' before the '*'). Since they match anything
// until the end, wildcard parameters must always be the final path element.
//  Path: /files/*filepath
//
//  Requests:
//   /files/                             match: filepath=""
//   /files/LICENSE                      match: filepath="LICENSE"
//   /files/templates/article.html       match: filepath="templates/article.html"
//   /files                              no match
//
// The value of parameters is saved as a Params. The Params
// is passed to the Handler func as a third parameter.
// If the handler is registered with Handle or HandleFunc,
// params is stored in request's context.
//
// Example:
//
// 	r:=apirouter.New(
// 		apirouter.GET("/streams/:id", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
// 			id := ps.ByName("id")
// 			fmt.Fprintf(w, "Page of stream #%s", id)
// 		}),
//
// 		apirouter.HandleFunc("GET","/users/:id", func(w http.ResponseWriter, r *http.Request) {
// 			ps := apirouter.PathParams(r.Context())
// 			id := ps.ByName("id")
// 			fmt.Fprintf(w, "Page of user #%s", id)
// 		}),
// 	)
//
package apirouter

import (
	"net/http"
)

// Handler is a function that can be registered to a router to
// handle HTTP requests.
//
// Compared with http.HandlerFunc, it add the third parameter for
// the path parameters extracted from the HTTP request.
type Handler func(w http.ResponseWriter, r *http.Request, ps Params)

// Router implements http.Handler interface.
// It matches the URL of each incoming request, and calls the
// handler that most closely matches the URL.
//
// NOTES: The zero value for Route is not available,
// it must be created with call New() function.
type Router struct {
	get     tree
	post    tree
	delete  tree
	put     tree
	patch   tree
	head    tree
	connect tree
	trace   tree
	options tree

	notFoundHandler http.Handler
}

// New returns a new Router,which is initialized with
// the given options and default pattern style.
//
// The syntax of the pattern string is as follows:
//
// 	Pattern		= "/" Segments
// 	Segments	= Segment { "/" Segment }
// 	Segment		= LITERAL | Parameter
//	Parameter	= Anonymous | Named
//	Anonymous	= ":" | "*"
//	Named		= ":" FieldPath [ "=" Regexp ] | "*" FieldPath
// 	FieldPath	= IDENT { "." IDENT }
//
func New(options ...Option) *Router {
	return newWithParser(defaultStyleParser(0), options...)
}

// NewForGRPC returns a new Router,which is initialized with
// the given options and gRPC pattern style.
//
// The syntax of the pattern string is as follows:
//
// 	Pattern		= "/" Segments [ Verb ] ;
// 	Segments	= Segment { "/" Segment } ;
// 	Segment		= LITERAL | Parameter
//	Parameter	= Anonymous | Named
//	Anonymous	= "*" | "**"
//	Named		= "{" FieldPath [ "=" Wildcard ] "}"
//	Wildcard	= "*" | "**" | Regexp
// 	FieldPath	= IDENT { "." IDENT } ;
// 	Verb		= ":" LITERAL ;
//
func NewForGRPC(options ...Option) *Router {
	return newWithParser(googleStyleParser(0), options...)
}

func newWithParser(parser parser, options ...Option) *Router {
	r := &Router{notFoundHandler: http.NotFoundHandler()}
	r.get.parser = parser
	r.post.parser = parser
	r.delete.parser = parser
	r.put.parser = parser
	r.patch.parser = parser
	r.head.parser = parser
	r.connect.parser = parser
	r.trace.parser = parser
	r.options.parser = parser

	for _, opt := range options {
		opt.apply(r)
	}
	r.initTrees()
	return r
}

// Match returns the handler to use and path params
// matched the given method and path.
//
// If there is no registered handler that applies to the given method and path,
// Match returns a nil handler and an empty path parameters.
func (r *Router) Match(method string, path string) (h Handler, params Params) {
	t := r.selectTree(method)
	if t != nil {
		h = t.match(path, &params)
	}
	return
}

var emptyParams Params

// ServeHTTP dispatches the request to the first handler
// whose matches to req.Method and req.Path.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var h Handler
	t := r.selectTree(req.Method)
	if t != nil {
		path := req.URL.Path
		if h = t.staticMatch(path); h != nil {
			h(w, req, emptyParams)
			return
		}

		var params Params
		h = t.patternMatch(path, &params)
		if h != nil {
			h(w, req, params)
			return
		}
	}
	r.notFoundHandler.ServeHTTP(w, req)
}

func (r *Router) initTrees() {
	r.get.init()
	r.post.init()
	r.delete.init()
	r.put.init()
	r.patch.init()
	r.head.init()
	r.connect.init()
	r.trace.init()
	r.options.init()
}

// selectTree returns the tree by the given HTTP method.
func (r *Router) selectTree(method string) *tree {
	switch method {
	case http.MethodGet:
		return &r.get
	case http.MethodPost:
		return &r.post
	case http.MethodDelete:
		return &r.delete
	case http.MethodPut:
		return &r.put
	case http.MethodPatch:
		return &r.patch
	case http.MethodHead:
		return &r.head
	case http.MethodConnect:
		return &r.connect
	case http.MethodTrace:
		return &r.trace
	case http.MethodOptions:
		return &r.options
	default:
		return nil
	}
}
