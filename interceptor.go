// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package apirouter

import "net/http"

// Interceptor provides a hook to intercept the execution of the HTTP request handler.
type Interceptor interface {
	// PreHandle achieve the preprocessing of the HTTP request handler (such as check login).
	// return value:
	// 	true - continue the process (such as calling the next interceptor or Handler);
	//	false - interrupte the process(such as the logon check fails)
	// and will not continue to invoke other interceptors or Handler,
	// in which case we need to generate a response through w;
	PreHandle(w http.ResponseWriter, r *http.Request, ps Params) bool

	// PostHandle achieve the post-processing of the HTTP request handler.
	PostHandle(r *http.Request, ps Params)
}

var (
	_     Interceptor = PreInterceptor(nil)
	_     Interceptor = PostInterceptor(nil)
	_     Interceptor = &chainInterceptor{}
	nopIt Interceptor = nopInterceptor{}
)

// PreInterceptor provides a hook function to intercept before the HTTP request handler is executed.
type PreInterceptor func(w http.ResponseWriter, r *http.Request, ps Params) bool

// PreHandle implements Intercetor.PreHandle.
func (f PreInterceptor) PreHandle(w http.ResponseWriter, r *http.Request, ps Params) bool {
	return f(w, r, ps)
}

// PostHandle implements Intercetor.PostHandle with no-op.
func (f PreInterceptor) PostHandle(r *http.Request, ps Params) {
	return
}

// PostInterceptor provides a hook function to intercept after the HTTP request handler is executed.
type PostInterceptor func(r *http.Request, ps Params)

// PreHandle implements Intercetor.PreHandle with no-op.
// It always returns true.
func (f PostInterceptor) PreHandle(w http.ResponseWriter, r *http.Request, ps Params) bool {
	return true
}

// PostHandle implements Intercetor.PostHandle.
func (f PostInterceptor) PostHandle(r *http.Request, ps Params) {
	f(r, ps)
}

// nopInterceptor is a no-op Interceptor.
type nopInterceptor struct{}

func (nopInterceptor) PreHandle(w http.ResponseWriter, r *http.Request, ps Params) bool { return true }
func (nopInterceptor) PostHandle(r *http.Request, ps Params)                            {}

// NewInterceptor returns a new Interceptor.
func NewInterceptor(pre PreInterceptor, post PostInterceptor) Interceptor {
	if pre == nil && post == nil {
		return nopIt
	}
	if pre == nil {
		return post
	}
	if post == nil {
		return pre
	}
	return interceptor{pre, post}
}

type interceptor struct {
	pre  PreInterceptor
	post PostInterceptor
}

func (it interceptor) PreHandle(w http.ResponseWriter, r *http.Request, ps Params) bool {
	return it.pre(w, r, ps)
}
func (it interceptor) PostHandle(r *http.Request, ps Params) {
	it.post(r, ps)
}

// ChainInterceptor creates a single interceptor out of a chain of many interceptors.
//
// PreHandle's Execution is done in left-to-right order.
// For example ChainInterceptor(one, two, three) will execute one before two before three, and three
// will see r changes of one and two.
// PostHandle's Execution is done in right-to-left order.
func ChainInterceptor(its ...Interceptor) Interceptor {
	switch len(its) {
	case 0:
		return nopIt
	case 1:
		return its[0]
	}

	ci := &chainInterceptor{make([]Interceptor, 0, len(its))}
	for _, it := range its {
		ci.addInterceptor(it)
	}
	return ci
}

type chainInterceptor struct {
	its []Interceptor
}

func (ci *chainInterceptor) addInterceptor(it Interceptor) {
	if it == nil {
		return
	}

	if subci, ok := it.(*chainInterceptor); ok {
		ci.its = append(ci.its, subci.its...)
	} else {
		ci.its = append(ci.its, it)
	}
}

func (ci *chainInterceptor) PreHandle(w http.ResponseWriter, r *http.Request, ps Params) bool {
	if len(ci.its) == 0 {
		return true
	}

	for _, it := range ci.its {
		if !it.PreHandle(w, r, ps) {
			return false
		}
	}
	return true
}

func (ci *chainInterceptor) PostHandle(r *http.Request, ps Params) {
	if len(ci.its) == 0 {
		return
	}
	for i := len(ci.its) - 1; i >= 0; i-- {
		it := ci.its[i]
		it.PostHandle(r, ps)
	}
}

// Wrap wraps the hanndler with the interceptors and transforms it into a different handler
func Wrap(h Handler, interceptors ...Interceptor) Handler {
	if len(interceptors) == 0 {
		return h
	}

	it := ChainInterceptor(interceptors...)
	return func(w http.ResponseWriter, r *http.Request, ps Params) {
		if it.PreHandle(w, r, ps) {
			h(w, r, ps)
			it.PostHandle(r, ps)
		}
	}
}

// WrapHandler wraps the http.Handler with the interceptors and transforms it into a different http.Handler
func WrapHandler(h http.Handler, interceptors ...Interceptor) http.Handler {
	if len(interceptors) == 0 {
		return h
	}

	it := ChainInterceptor(interceptors...)
	return wrapHandler{h, it}
}

type wrapHandler struct {
	h  http.Handler
	it Interceptor
}

func (wh wrapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ps := Params{}
	psp := PathParams(r.Context())
	if psp != nil {
		ps = *psp
	}

	if wh.it.PreHandle(w, r, ps) {
		wh.h.ServeHTTP(w, r)
		wh.it.PostHandle(r, ps)
	}
}

// WrapHandlerFunc wraps the http.HandlerFunc with the interceptors and transforms it into a different http.HandlerFunc
func WrapHandlerFunc(h http.HandlerFunc, interceptors ...Interceptor) http.HandlerFunc {
	if len(interceptors) == 0 {
		return h
	}

	it := ChainInterceptor(interceptors...)
	return func(w http.ResponseWriter, r *http.Request) {
		ps := Params{}
		psp := PathParams(r.Context())
		if psp != nil {
			ps = *psp
		}

		if it.PreHandle(w, r, ps) {
			h(w, r)
			it.PostHandle(r, ps)
		}
	}
}
