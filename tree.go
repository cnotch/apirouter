// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package apirouter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unsafe"

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

// entry stores entry of route
type entry struct {
	pnames []string // parameter names extracted from the pattern
	h      Handler
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

	// keys the list of the key of route entry
	keys []string
	// es the list of route entry
	es []entry

	// res parameter validation regular expressions
	res []*regexp.Regexp

	// static pattern is handled separately
	// Learn from aero (https://github.com/aerogo/aero)
	static      map[string]Handler
	canBeStatic [2048]bool
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
	state := rootState

	lastStarState := -1 // last '*' state
	lastStarIndex := 0  // index of the last '*' in the path
	pcount := uint16(0) // parameter count
OUTER:
	for i, sc := 0, len(t.base); i < len(path); {
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

	// get the end state
	endState := t.base[state] + endCode
	currbase := t.base[endState]
	if t.check[endState] == state && currbase < 0 {
		i := -currbase - 1
		params.path = path
		params.names = t.es[i].pnames
		h = t.es[i].h
	}
	return
}

// regular expressions parameter include ':' + res[index]
func (t *tree) matchReParam(state, sc int, segment string) int {
	next := t.base[state] + code(':')
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

func (t *tree) add(pattern string, handler Handler) {
	e := entry{h: handler}
	key := make([]byte, 0, len(pattern)+1)

	prevChar := byte(0)
	c := byte(0)
	for i := 0; i < len(pattern); i, prevChar = i+1, c {
		c = pattern[i]
		key = append(key, c)

		if prevChar != '/' {
			continue
		}

		if c == ':' {
			m := strings.IndexByte(pattern[i:], '/')
			var nameAndRe string
			if m < 0 { // last part
				nameAndRe = pattern[i+1:]
				i = len(pattern) - 1
			} else {
				nameAndRe = pattern[i+1 : i+m]
				i = i + m - 1 // for i++
			}

			reSep := strings.IndexByte(nameAndRe, ':') // Search for a name/regexp separator.
			if reSep < 0 {                             // only name
				e.pnames = append(e.pnames, nameAndRe)
			} else {
				e.pnames = append(e.pnames, nameAndRe[:reSep])
				res := nameAndRe[reSep+1:]
				if res == "" {
					panic(fmt.Errorf("router: pattern has empty regular expression - %q", pattern))
				}
				rec := -1 // regular expression keychar
				for j, exp := range t.res {
					if exp.String() == res {
						rec = j
						break
					}
				}
				if rec == -1 { // regular expression not exist
					re := regexp.MustCompile(res)
					rec = len(t.res)
					t.res = append(t.res, re)
				}

				key = append(key, ':', byte(rec))
			}
		} else if c == '*' {
			m := strings.IndexByte(pattern[i:], '/')
			if m > 0 {
				panic(fmt.Errorf("router: '*' in pattern must is last segment - %q", pattern))
			}
			e.pnames = append(e.pnames, pattern[i+1:])
			i = len(pattern) - 1
		} else if c == '/' {
			panic(fmt.Errorf("router: pattern include empty segment - %q", pattern))
		}
	}

	if len(e.pnames) == 0 { // static
		if t.static == nil {
			t.static = make(map[string]Handler)
		}
		t.static[pattern] = handler
		t.canBeStatic[len(pattern)] = true
	} else {
		t.keys = append(t.keys, *(*string)(unsafe.Pointer(&key)))
		t.es = append(t.es, e)
	}
}

func (t *tree) init() {
	// sort and de-duplicate
	t.rearrange()
	t.grow((len(t.es) + 1) * 2)
	if len(t.es) == 0 {
		return
	}

	var q queue.Queue
	// get the child nodes of root
	rootChilds := t.getNodes(node{
		state: rootState,
		depth: 0,
		begin: 0,
		end:   len(t.es),
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

	t.keys = nil
}

type byKey tree

func (p *byKey) Len() int           { return len(p.keys) }
func (p *byKey) Less(i, j int) bool { return p.keys[i] < p.keys[j] }
func (p *byKey) Swap(i, j int) {
	p.keys[i], p.keys[j] = p.keys[j], p.keys[i]
	p.es[i], p.es[j] = p.es[j], p.es[i]
}

func (t *tree) rearrange() {
	sort.Sort((*byKey)(t))

	// de-duplicate
	for i := len(t.es) - 1; i > 0; i-- {
		if t.keys[i] == t.keys[i-1] {
			copy(t.es[i-1:], t.es[i:])
			t.es = t.es[:len(t.es)-1]
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
	if i < n.end && len(t.keys[i]) == n.depth { // the end of key
		l.append(endCode, n.depth+1, i, i+1)
		i++
	}

	var currBegin int
	currCode := -1
	for ; i < n.end; i++ {
		code := code(t.keys[i][n.depth])
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
