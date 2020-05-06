// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package apirouter_test

import (
	"net/http"
	"testing"

	"github.com/cnotch/apirouter"
	"github.com/stretchr/testify/assert"
)

func TestWrap(t *testing.T) {
	signature := ""
	unWrap := func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		signature += "D"
	}
	h := apirouter.Wrap(unWrap,
		apirouter.NewInterceptor(
			func(w http.ResponseWriter, r *http.Request) bool {
				signature += "A"
				return true
			}, func(r *http.Request) {
				signature += "B"
			}),
		apirouter.PreInterceptor(func(w http.ResponseWriter, r *http.Request) bool {
			signature += "C"
			return true
		}),
	)
	h(nil, nil, apirouter.Params{})
	assert.Equal(t, "ACDB", signature)
}

func TestWrapAbort(t *testing.T) {
	signature := ""
	unWrap := func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		signature += "D"
	}
	h := apirouter.Wrap(unWrap,
		apirouter.PreInterceptor(func(w http.ResponseWriter, r *http.Request) bool {
			signature += "A"
			return true
		}),
		apirouter.NewInterceptor(
			func(w http.ResponseWriter, r *http.Request) bool {
				signature += "B"
				return false
			}, func(r *http.Request) {
				signature += "C"
			}),
	)
	h(nil, nil, apirouter.Params{})
	assert.Equal(t, "AB", signature)
}
