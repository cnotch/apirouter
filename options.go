// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package apirouter

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unsafe"
)

// Option represents all possible options to the New() function
type Option interface {
	apply(*Router)
}

type optionFunc func(*Router)

func (f optionFunc) apply(r *Router) {
	f(r)
}

// NotFoundHandler creates the option to set a request handler that
// replies to each request with a “404 page not found” reply.
func NotFoundHandler(handler http.Handler) Option {
	if handler == nil {
		panic("router: nil handler")
	}

	return optionFunc(func(r *Router) {
		r.notFoundHandler = handler
	})
}

// Handle creates the option to registers the handler for the given method and pattern.
func Handle(method string, pattern string, handler Handler) Option {
	if handler == nil {
		panic("router: nil handler")
	}

	if !strings.HasPrefix(pattern, "/") {
		panic(fmt.Errorf("router: pattern no leading / - %q", pattern))
	}

	return optionFunc(func(r *Router) {
		t := r.selectTree(method)
		if t == nil {
			panic(fmt.Errorf("router: unknown http method - %q", method))
		}
		t.add(pattern, handler)
	})
}

// GET is a shortcut for Handle(http.MethodGet, pattern, handler)
func GET(pattern string, handler Handler) Option {
	return Handle(http.MethodGet, pattern, handler)
}

// POST is a shortcut for Handle(http.MethodPost, pattern, handler)
func POST(pattern string, handler Handler) Option {
	return Handle(http.MethodPost, pattern, handler)
}

// PUT is a shortcut for Handle(http.MethodPut, pattern, handler)
func PUT(pattern string, handler Handler) Option {
	return Handle(http.MethodPut, pattern, handler)
}

// DELETE is a shortcut for Handle(http.MethodDelete, pattern, handler)
func DELETE(pattern string, handler Handler) Option {
	return Handle(http.MethodDelete, pattern, handler)
}

// HEAD is a shortcut for Handle(http.MethodHead, pattern, handler)
func HEAD(pattern string, handler Handler) Option {
	return Handle(http.MethodHead, pattern, handler)
}

// OPTIONS is a shortcut for Handle(http.MethodOptions, pattern, handler)
func OPTIONS(pattern string, handler Handler) Option {
	return Handle(http.MethodOptions, pattern, handler)
}

// PATCH is a shortcut for Handle(http.MethodPatch, pattern, handler)
func PATCH(pattern string, handler Handler) Option {
	return Handle(http.MethodPatch, pattern, handler)
}

var ctxOffset uintptr

func init() {
	var req http.Request
	sf, _ := reflect.TypeOf(req).FieldByName("ctx")
	ctxOffset = sf.Offset
}

// HTTPHandle creates the option to registers the stdlib handler for the given method and pattern.
func HTTPHandle(method string, pattern string, handler http.Handler) Option {
	if handler == nil {
		panic("router: nil handler")
	}

	return Handle(method, pattern, func(w http.ResponseWriter, r *http.Request, ps Params) {
		if ps.Count() > 0 {
			paramsCtx := newParamsCtx(r.Context())
			paramsCtx.params = ps
			ctxp := (*context.Context)(unsafe.Pointer(uintptr(unsafe.Pointer(r)) + ctxOffset))
			oldCtx := *ctxp
			*ctxp = paramsCtx

			handler.ServeHTTP(w, r)

			*ctxp = oldCtx
			paramsCtx.Close()
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

// HTTPHandleFunc creates the option to registers the stdlib handler function for the given method and pattern.
func HTTPHandleFunc(method string, pattern string, handler func(http.ResponseWriter, *http.Request)) Option {
	if handler == nil {
		panic("router: nil handler")
	}
	return HTTPHandle(method, pattern, http.HandlerFunc(handler))
}
