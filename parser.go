// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package apirouter

import (
	"fmt"
	"regexp"
	"strings"
	"unsafe"
)

// parser parses the string representation of path pattern
type parser interface {
	parse(pattern string, regexps *[]*regexp.Regexp) route
	splitPath(path string) (segments, verb string)
}

type defaultStyleParser int

func (p defaultStyleParser) parse(pattern string, regexps *[]*regexp.Regexp) route {
	var params []string
	kbuilder := make([]byte, 0, len(pattern))
	segments := pattern

	prevChar := byte(0)
	c := byte(0)
	for i := 0; i < len(segments); i, prevChar = i+1, c {
		c = segments[i]
		kbuilder = append(kbuilder, c)

		if prevChar != '/' {
			continue
		}
		if c == '/' {
			panic(fmt.Errorf("router: pattern include empty segment - %q", segments))
		}

		if c == ':' { // named parameter
			m := strings.IndexByte(segments[i:], '/')
			var nameAndRe string
			if m < 0 { // last part
				nameAndRe = segments[i+1:]
				i = len(segments) - 1
			} else {
				nameAndRe = segments[i+1 : i+m]
				i = i + m - 1 // for i++
			}

			reSep := strings.IndexByte(nameAndRe, '=') // Search for a name/regexp separator.
			if reSep < 0 {                             // only name
				params = append(params, nameAndRe)
			} else {
				params = append(params, nameAndRe[:reSep])
				res := nameAndRe[reSep+1:]
				if res == "" {
					panic(fmt.Errorf("router: pattern has empty regular expression - %q", segments))
				}
				rec := -1 // regular expression keychar
				for j, exp := range *regexps {
					if exp.String() == res {
						rec = j
						break
					}
				}
				if rec == -1 { // regular expression not exist
					re := regexp.MustCompile(res)
					rec = len(*regexps)
					*regexps = append(*regexps, re)
				}

				kbuilder = append(kbuilder, '=', byte(rec))
			}
		} else if c == '*' { // wildcard parameter
			m := strings.IndexByte(segments[i:], '/')
			if m > 0 {
				panic(fmt.Errorf("router: '*' in pattern must is last segment - %q", segments))
			}
			params = append(params, segments[i+1:])
			i = len(segments) - 1
		}
	}

	return route{pattern: pattern,
		key:    *(*string)(unsafe.Pointer(&kbuilder)),
		params: params}
}

func (p defaultStyleParser) splitPath(path string) (segments, verb string) {
	return path, ""
}

type googleStyleParser int

func (p googleStyleParser) parse(pattern string, regexps *[]*regexp.Regexp) route {
	var params []string
	kbuilder := make([]byte, 0, len(pattern)+1)
	segments, verb := p.splitPath(pattern)

	prevChar := byte(0)
	c := byte(0)
	for i := 0; i < len(segments); i, prevChar = i+1, c {
		c = segments[i]
		if prevChar != '/' {
			kbuilder = append(kbuilder, c)
			continue
		}
		if c == '/' {
			panic(fmt.Errorf("router: pattern include empty segment - %q", segments))
		}
		if c != '{' && c != '*' {
			kbuilder = append(kbuilder, c)
			continue
		}

		begin := i
		m := strings.IndexByte(segments[begin:], '/')
		if m < 0 { // last part
			i = len(segments) - 1
		} else {
			i = begin + m - 1 // for i++
		}

		segment := segments[begin : i+1]
		// anonymous parameter
		if segment == "*" {
			kbuilder = append(kbuilder, ':')
			params = append(params, "")
			continue
		}
		if segment == "**" {
			if m > 0 {
				panic(fmt.Errorf("router: '*' in pattern must is last segment - %q", segments))
			}
			kbuilder = append(kbuilder, '*')
			params = append(params, "")
			continue
		}

		// {name=value},remove '{}'
		if segment[0] == '{' {
			if len(segment) < 2 || segment[len(segment)-1] != '}' {
				panic(fmt.Errorf("router: pattern  lack of '}' - %q", segments))
			}
			segment = segment[1 : len(segment)-1]
		}

		nvSep := strings.IndexByte(segment, '=') // Search for a name/value(regexp) separator.
		var name, value string
		if nvSep < 0 {
			name = segment
			value = "*"
		} else {
			name = segment[:nvSep]
			value = segment[nvSep+1:]
		}

		params = append(params, name)
		switch value {
		case "*": //named parameter
			kbuilder = append(kbuilder, ':')
		case "**": // wildcard
			if m > 0 {
				panic(fmt.Errorf("router: '*' in pattern must is last segment - %q", segments))
			}
			kbuilder = append(kbuilder, '*')
		default: // regexp
			if value == "" {
				panic(fmt.Errorf("router: pattern has empty regular expression - %q", segments))
			}
			rec := -1 // regular expression keychar
			for j, exp := range *regexps {
				if exp.String() == value {
					rec = j
					break
				}
			}
			if rec == -1 { // regular expression not exist
				re := regexp.MustCompile(value)
				rec = len(*regexps)
				*regexps = append(*regexps, re)
			}

			kbuilder = append(kbuilder, ':', '=', byte(rec))
		}
	}
	if verb != "" {
		kbuilder = append(kbuilder, verb...)
	}

	return route{pattern: pattern,
		key:    *(*string)(unsafe.Pointer(&kbuilder)),
		params: params}
}

func (p googleStyleParser) splitPath(path string) (segments, verb string) {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == ':' {
			return path[:i], path[i:]
		}
	}
	return path, ""
}
