// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package apirouter

import (
	"regexp"
	"sort"
	"sync"

	"github.com/cnotch/queue"
)

const (
	rootState            = 1 // Since state 0 cannot be the parent state, set the root state to 1
	minBase              = rootState + 1
	endCode              = 0 // # code (the end of key)
	codeOffset           = endCode + 1
	growMultiple         = 1.5
	percentageOfNonempty = 0.95
)

// route stores the route entry in the router
type route struct {
	key     string
	h       Handler
	params  []string // parameters extracted from the pattern
	pattern string   // original pattern string
}

func code(c byte) int {
	return int(c) + codeOffset
}

// tree double-array trie for router。
type tree struct {
	// base stores the offset base address of the child state
	//	=0 free
	//	>0 offset base address of child
	//	<0 entry index
	base []int

	// check stores the parent state
	check []int

	// routes the list of route entry
	routes []route

	// res parameter validation regular expressions
	res []*regexp.Regexp

	// static pattern is handled separately
	// Learn from aero (https://github.com/aerogo/aero)
	static      map[string]Handler
	canBeStatic [2048]bool

	parser parser
}

func (t *tree) add(pattern string, h Handler) {
	route := t.parser.parse(pattern, &t.res)
	if len(route.params) == 0 { // static
		if t.static == nil {
			t.static = make(map[string]Handler)
		}
		t.static[pattern] = h
		t.canBeStatic[len(pattern)] = true
	} else {
		route.h = h
		t.routes = append(t.routes, route)
	}
}

func (t *tree) staticMatch(path string) Handler {
	if t.canBeStatic[len(path)] {
		if h, found := t.static[path]; found {
			return h
		}
	}
	return nil
}

func (t *tree) patternMatch(path string, params *Params) (h Handler) {
	path, verb := t.parser.splitPath(path)
	state := rootState

	lastStarState := -1 // last '*' state
	lastStarIndex := 0  // index of the last '*' in the path
	pcount := uint16(0) // parameter count
	sc := len(t.base)
OUTER:
	for i := 0; i < len(path); {
		// try to match the beginning '/' of current segment
		slashState := t.base[state] + code('/')
		if !(slashState < sc && state == t.check[slashState]) {
			return
		}
		state = slashState
		i++
		begin := i // begin index of current segment

		// try to match * wildcard
		next := t.base[slashState] + code('*')
		if next < sc && slashState == t.check[next] {
			lastStarIndex = begin
			lastStarState = next
		}

		// try to match current segment
		for ; i < len(path) && path[i] != '/'; i++ {
			next := t.base[state] + code(path[i])
			if next < sc && state == t.check[next] {
				state = next
				continue
			}

			// exact matching failed
			// try to match named parameter
			next = t.base[slashState] + code(':')
			if !(next < sc && slashState == t.check[next]) {
				state = -1
				break OUTER
			}
			state = next

			// the ending / of segment
			for ; i < len(path); i++ {
				if path[i] == '/' {
					break
				}
			}

			// regular expression parameters are not required in most cases
			if len(t.res) > 0 {
				// try match regular expressions
				state = t.matchReParam(state, sc, path[begin:i])
			}

			index := pcount << 1
			params.indices[index] = int16(begin)
			params.indices[index+1] = int16(i)
			pcount++
			continue OUTER
		}
	}

	// If all other matching fail, try using * wildcard
	if state == -1 {
		if lastStarState == -1 {
			return
		}
		index := pcount << 1
		params.indices[index] = int16(lastStarIndex)
		params.indices[index+1] = int16(len(path))
		pcount++
		state = lastStarState
	}

	if verb != "" { // match verb
		for i := 0; i < len(verb); i++ {
			next := t.base[state] + code(verb[i])
			if next < sc && state == t.check[next] {
				state = next
			} else {
				return
			}
		}
	}

	// get the end state
	endState := t.base[state] + endCode
	if endState < sc && t.check[endState] == state && t.base[endState] < 0 {
		i := -t.base[endState] - 1
		params.path = path
		params.names = t.routes[i].params
		h = t.routes[i].h
	}
	return
}

// regular expressions parameter include ':' + res[index]
func (t *tree) matchReParam(state, sc int, segment string) int {
	next := t.base[state] + code('=')
	if next < sc && state == t.check[next] {
		reState := next
		// check regular expressions
		for j := 0; j < len(t.res); j++ {
			next := t.base[reState] + j + codeOffset
			if next >= sc {
				break
			}
			if reState == t.check[next] { // exist  parameter reg expressions
				if t.res[j].MatchString(segment) {
					state = next // ok
					break
				}
			}
		}
	}
	return state
}

// match returns the handler and path parameters that matches the given path.
func (t *tree) match(path string, params *Params) (h Handler) {
	if t.canBeStatic[len(path)] {
		if handler, found := t.static[path]; found {
			return handler
		}
	}
	return t.patternMatch(path, params)
}

func (t *tree) init() {
	// sort and de-duplicate
	t.rearrange()
	t.grow((len(t.routes) + 1) * 2)
	if len(t.routes) == 0 {
		return
	}

	var q queue.Queue
	// get the child nodes of root
	rootChilds := t.getNodes(node{
		state: rootState,
		depth: 0,
		begin: 0,
		end:   len(t.routes),
	})

	var base int            // offset base of children
	nextCheckPos := minBase // check position for free state

	q.Push(rootChilds)
	for q.Len() > 0 {
		e, _ := q.Pop()
		curr := e.(*nodes)

		base, nextCheckPos = t.getBase(curr, nextCheckPos)
		t.base[curr.state] = base
		for i := 0; i < len(curr.childs); i++ {
			n := &curr.childs[i]
			n.state = base + n.code       // set state
			t.check[n.state] = curr.state // set parent state

			if n.code == endCode { // the end of key
				t.base[n.state] = -(n.begin + 1)
			} else {
				q.Push(t.getNodes(*n))
			}
		}

		curr.state = 0
		curr.childs = curr.childs[:0]
		nodesPool.Put(curr)
	}
}

func (t *tree) rearrange() {
	sort.Slice(t.routes, func(i, j int) bool {
		return t.routes[i].key < t.routes[j].key
	})

	// de-duplicate
	for i := len(t.routes) - 1; i > 0; i-- {
		if t.routes[i].key == t.routes[i-1].key {
			copy(t.routes[i-1:], t.routes[i:])
			t.routes = t.routes[:len(t.routes)-1]
		}
	}
}

func (t *tree) grow(n int) int {
	c := cap(t.base)
	size := int(growMultiple*float64(c)) + n
	newBase := make([]int, size)
	newCheck := make([]int, size)
	copy(newBase, t.base)
	copy(newCheck, t.check)
	t.base = newBase
	t.check = newCheck
	return size
}

func (t *tree) getBase(l *nodes, checkPos int) (base, nextCheckPos int) {
	nextCheckPos = checkPos
	minCode, number := l.numberOfStates()

	var pos int
	if minCode+minBase > nextCheckPos {
		pos = minCode + minBase
	} else {
		pos = nextCheckPos
	}

	nonZeroNum := 0
	first := true
OUTER:
	for ; ; pos++ {
		// check memory
		if pos+number > len(t.base) {
			t.grow(pos + number - len(t.base))
		}

		if t.check[pos] != 0 {
			nonZeroNum++
			continue
		} else if first {
			nextCheckPos = pos
			first = false
		}

		base = pos - minCode
		for i := 0; i < len(l.childs); i++ {
			n := &l.childs[i]
			if t.check[base+n.code] != 0 {
				continue OUTER
			}
		}
		break // found
	}

	// -- Simple heuristics --
	// if the percentage of non-empty contents in check between the
	// index
	// 'next_check_pos' and 'check' is greater than some constant value
	// (e.g. 0.9),
	// new 'next_check_pos' index is written by 'check'.
	if 1.0*float64(nonZeroNum)/float64(pos-nextCheckPos+1) >= percentageOfNonempty {
		nextCheckPos = pos
	}

	return
}

var nodesPool = sync.Pool{
	New: func() interface{} {
		return new(nodes)
	},
}

// getNodes returns the child nodes of a given node
func (t *tree) getNodes(n node) *nodes {
	l := nodesPool.Get().(*nodes)
	l.state = n.state

	i := n.begin
	if i < n.end && len(t.routes[i].key) == n.depth { // the end of key
		l.append(endCode, n.depth+1, i, i+1)
		i++
	}

	var currBegin int
	currCode := -1
	for ; i < n.end; i++ {
		code := code(t.routes[i].key[n.depth])
		if currCode != code {
			if currCode != -1 {
				l.append(currCode, n.depth+1, currBegin, i)
			}
			currCode = code
			currBegin = i
		}
	}
	if currCode != -1 {
		l.append(currCode, n.depth+1, currBegin, i)
	}
	return l
}

type node struct {
	code       int
	depth      int
	begin, end int
	state      int
}
type nodes struct {
	state  int
	childs []node
}

func (l *nodes) append(code, depth, begin, end int) {
	l.childs = append(l.childs, node{
		code:  code,
		depth: depth,
		begin: begin,
		end:   end,
	})
}

// The number of the required state
func (l *nodes) numberOfStates() (minCode, number int) {
	if len(l.childs) == 0 {
		return 0, 0
	}
	return l.childs[0].code, l.childs[len(l.childs)-1].code - l.childs[0].code + 1
}
