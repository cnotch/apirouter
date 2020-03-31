package apirouter_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/cnotch/apirouter"
	"github.com/stretchr/testify/assert"
)

type route struct {
	method string
	path   string
}

func loadRouterSingle(method, path string, h apirouter.Handler) http.Handler {
	return apirouter.New(apirouter.Handle(method, path, h))
}

func loadRouter(routes []route, h apirouter.Handler) *apirouter.Router {
	options := make([]apirouter.Option, len(routes))
	for i, route := range routes {
		options[i] = apirouter.Handle(route.method, route.path, h)
	}

	return apirouter.New(options...)
}

type testCase struct {
	method      string
	path        string
	hasHandler  bool
	wantPNames  []string
	wantPValues []string
}

func runTestCases(t *testing.T, r *apirouter.Router, testCases []testCase) {
	for _, tc := range testCases {
		h, ps := r.Match(tc.method, tc.path)
		if tc.hasHandler {
			assert.NotNil(t, h)
			for i := 0; i < ps.Count(); i++ {
				assert.Equal(t, tc.wantPNames[i], ps.Name(i))
				assert.Equal(t, tc.wantPValues[i], ps.Value(i))
			}
		} else {
			assert.Nil(t, h)
		}
	}
}

func TestRouterMatch(t *testing.T) {
	routes := []route{
		{"GET", "/"},
		{"GET", "/user"},
		{"GET", `/user/:id:^\d+$/books`},
		{"GET", "/user/:id"},
		{"GET", "/user/:id/profile"},
		{"GET", "/user/:id/profile/:theme"},
		{"GET", "/user/:id/:something"},
		{"GET", "/admin"},
		{"GET", `/admin/:role:^\d+$`},
		{"GET", "/中国人"},
	}

	testCases := []testCase{
		{"GET", "/", true, nil, nil},
		{"GET", "/user", true, nil, nil},
		{"GET", "/user/123/books", true, []string{"id"}, []string{"123"}},
		{"GET", "/user/guest", true, []string{"id"}, []string{"guest"}},
		{"GET", "/user/guest/profile", true, []string{"id"}, []string{"guest"}},
		{"GET", "/user/guest/profile/456", true, []string{"id", "theme"}, []string{"guest", "456"}},
		{"GET", "/user/guest/456", true, []string{"id", "something"}, []string{"guest", "456"}},
		{"GET", "/admin", true, nil, nil},
		{"GET", "/x", false, nil, nil},
		{"GET", "/user/guest/456/x", false, nil, nil},
		{"GET", "/admin/x", false, nil, nil},
		{"GET", "/admin/888", true, []string{"role"}, []string{"888"}},
		{"GET", "/中", false, nil, nil},
		{"GET", "/中国", false, nil, nil},
		{"GET", "/中国人", true, nil, nil},
		{http.MethodHead, "/", false, nil, nil},
		{http.MethodPost, "/", false, nil, nil},
		{http.MethodPut, "/", false, nil, nil},
		{http.MethodPatch, "/", false, nil, nil},
		{http.MethodDelete, "/", false, nil, nil},
		{http.MethodConnect, "/", false, nil, nil},
		{http.MethodOptions, "/", false, nil, nil},
		{http.MethodTrace, "/", false, nil, nil},
	}
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	runTestCases(t, loadRouter(routes, page), testCases)
}

func TestRouterMatchWildcards(t *testing.T) {
	routes := []route{
		{"GET", "/"},
		{"GET", "/images"},
		{"GET", "/images/*file"},
		{"GET", "/videos/*file"},
		{"GET", "/*anything"},
	}

	testCases := []testCase{
		{"GET", "/", true, nil, nil},
		{"GET", "/images", true, nil, nil},
		{"GET", "/images/hello.webp", true, []string{"file"}, []string{"hello.webp"}},
		{"GET", "/videos/hello.webm", true, []string{"file"}, []string{"hello.webm"}},
		{"GET", "/documents/hello.txt", true, []string{"anything"}, []string{"documents/hello.txt"}},
	}
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}

	runTestCases(t, loadRouter(routes, page), testCases)
}

func TestRouterServeHTTP(t *testing.T) {
	handleCount := 0
	router := apirouter.New(
		apirouter.Handle("GET", "/user/repos", func(w http.ResponseWriter, req *http.Request, ps apirouter.Params) {
			handleCount++
			assert.Zero(t, ps.Count())
		}),
		apirouter.HTTPHandleFunc("GET", "/user/:name", func(w http.ResponseWriter, req *http.Request) {
			handleCount++
			pp := apirouter.PathParams(req.Context())
			assert.NotNil(t, pp)
			assert.Equal(t, 1, pp.Count())
			assert.Equal(t, "name", pp.Name(0))
			assert.Equal(t, "gordon", pp.Value(0))
			assert.Equal(t, "gordon", pp.ByName("name"))
		}),
		apirouter.Handle("GET", "/:a/:b/:c/:d/:e", func(w http.ResponseWriter, req *http.Request, ps apirouter.Params) {
			handleCount++
			assert.Equal(t, 5, ps.Count())
			assert.Equal(t, "a", ps.Name(0))
			assert.Equal(t, "test1", ps.Value(0))
			assert.Equal(t, "test2", ps.ByName("b"))
		}),
		apirouter.NotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "/test1/test2", req.URL.Path)
		})),
	)
	w := new(mockResponseWriter)
	r, _ := http.NewRequest("GET", "/user/repos", nil)
	router.ServeHTTP(w, r)

	r, _ = http.NewRequest("GET", "/user/gordon", nil)
	router.ServeHTTP(w, r)

	r, _ = http.NewRequest("GET", "/test1/test2/test3/test4/test5", nil)
	router.ServeHTTP(w, r)

	r, _ = http.NewRequest("GET", "/test1/test2", nil)
	router.ServeHTTP(w, r)

	assert.Equal(t, 3, handleCount)
}

func TestRouterNewPanic(t *testing.T) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	assert.Panics(t, func() {
		router := apirouter.New(apirouter.Handle("GET", "/", nil))
		_ = router
	})
	assert.Panics(t, func() {
		router := apirouter.New(apirouter.Handle("GET", "/", apirouter.Handler(nil)))
		_ = router
	})
	assert.Panics(t, func() {
		router := apirouter.New(apirouter.Handle("GET", "user/admin", page))
		_ = router
	})
	assert.Panics(t, func() {
		router := apirouter.New(apirouter.Handle("GET", "/user//admin", page))
		_ = router
	})
	assert.Panics(t, func() {
		router := apirouter.New(apirouter.Handle("GET", "/user/*id/books", page))
		_ = router
	})
	assert.Panics(t, func() {
		router := apirouter.New(apirouter.Handle("PICK", "/", page))
		_ = router
	})
	assert.Panics(t, func() {
		router := apirouter.New(apirouter.NotFoundHandler(nil))
		_ = router
	})
}

func BenchmarkStaticRoutes(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	routes := staticRoutes
	r := loadRouter(routes, page)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, route := range routes {
			if h, _ := r.Match(route.method, route.path); h == nil {
				b.Fatal(route.method + " " + route.path)
			}
		}
	}
}

func BenchmarkGitHubRoutes(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	routes := githubAPI
	r := loadRouter(routes, page)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, route := range routes {
			if h, _ := r.Match(route.method, route.path); h == nil {
				b.Fatal(route.method + " " + route.path)
			}
		}
	}
}

func BenchmarkApiRouter_New(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	routes := githubAPI

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r := loadRouter(routes, page)
		_ = r
	}
}

func BenchmarkApiRouter_Param(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	router := loadRouterSingle("GET", "/user/:name", page)

	r, _ := http.NewRequest("GET", "/user/gordon", nil)
	benchRequest(b, router, r)
}

func BenchmarkApiRouter_Param5(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	router := loadRouterSingle("GET", "/:a/:b/:c/:d/:e", page)

	r, _ := http.NewRequest("GET", "/test/test/test/test/test", nil)
	benchRequest(b, router, r)
}

func BenchmarkApiRouter_Param20(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	router := loadRouterSingle("GET", "/:a/:b/:c/:d/:e/:f/:g/:h/:i/:j/:k/:l/:m/:n/:o/:p/:q/:r/:s/:t", page)

	r, _ := http.NewRequest("GET", "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t", nil)
	benchRequest(b, router, r)
}

func BenchmarkApiRouter_ParamWrite(b *testing.B) {
	router := loadRouterSingle("GET", "/user/:name", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		io.WriteString(w, ps.ByName("name"))
	})

	r, _ := http.NewRequest("GET", "/user/gordon", nil)
	benchRequest(b, router, r)
}

func BenchmarkApiRouter_GithubStatic(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	r := loadRouter(githubAPI, page)

	req, _ := http.NewRequest("GET", "/user/repos", nil)
	benchRequest(b, r, req)
}

func BenchmarkApiRouter_GithubParam(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	r := loadRouter(githubAPI, page)

	req, _ := http.NewRequest("GET", "/repos/julienschmidt/httprouter/stargazers", nil)
	benchRequest(b, r, req)
}

func BenchmarkApiRouter_GithubAll(b *testing.B) {
	page := func(_ http.ResponseWriter, req *http.Request, ps apirouter.Params) {}
	routes := githubAPI
	handler := loadRouter(routes, page)

	w := new(mockResponseWriter)
	r, _ := http.NewRequest("GET", "/", nil)
	u := r.URL
	rq := u.RawQuery

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, route := range routes {
			r.Method = route.method
			r.RequestURI = route.path
			u.Path = route.path
			u.RawQuery = rq
			handler.ServeHTTP(w, r)
		}
	}
}

func benchRequest(b *testing.B, router http.Handler, r *http.Request) {
	w := new(mockResponseWriter)
	u := r.URL
	rq := u.RawQuery
	r.RequestURI = u.RequestURI()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		u.RawQuery = rq
		router.ServeHTTP(w, r)
	}
}

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

// http://developer.github.com/v3/
var githubAPI = []route{
	// OAuth Authorizations
	{"GET", "/authorizations"},
	{"GET", "/authorizations/:id"},
	{"POST", "/authorizations"},
	//{"PUT", "/authorizations/clients/:client_id"},
	//{"PATCH", "/authorizations/:id"},
	{"DELETE", "/authorizations/:id"},
	{"GET", "/applications/:client_id/tokens/:access_token"},
	{"DELETE", "/applications/:client_id/tokens"},
	{"DELETE", "/applications/:client_id/tokens/:access_token"},

	// Activity
	{"GET", "/events"},
	{"GET", "/repos/:owner/:repo/events"},
	{"GET", "/networks/:owner/:repo/events"},
	{"GET", "/orgs/:org/events"},
	{"GET", "/users/:user/received_events"},
	{"GET", "/users/:user/received_events/public"},
	{"GET", "/users/:user/events"},
	{"GET", "/users/:user/events/public"},
	{"GET", "/users/:user/events/orgs/:org"},
	{"GET", "/feeds"},
	{"GET", "/notifications"},
	{"GET", "/repos/:owner/:repo/notifications"},
	{"PUT", "/notifications"},
	{"PUT", "/repos/:owner/:repo/notifications"},
	{"GET", "/notifications/threads/:id"},
	//{"PATCH", "/notifications/threads/:id"},
	{"GET", "/notifications/threads/:id/subscription"},
	{"PUT", "/notifications/threads/:id/subscription"},
	{"DELETE", "/notifications/threads/:id/subscription"},
	{"GET", "/repos/:owner/:repo/stargazers"},
	{"GET", "/users/:user/starred"},
	{"GET", "/user/starred"},
	{"GET", "/user/starred/:owner/:repo"},
	{"PUT", "/user/starred/:owner/:repo"},
	{"DELETE", "/user/starred/:owner/:repo"},
	{"GET", "/repos/:owner/:repo/subscribers"},
	{"GET", "/users/:user/subscriptions"},
	{"GET", "/user/subscriptions"},
	{"GET", "/repos/:owner/:repo/subscription"},
	{"PUT", "/repos/:owner/:repo/subscription"},
	{"DELETE", "/repos/:owner/:repo/subscription"},
	{"GET", "/user/subscriptions/:owner/:repo"},
	{"PUT", "/user/subscriptions/:owner/:repo"},
	{"DELETE", "/user/subscriptions/:owner/:repo"},

	// Gists
	{"GET", "/users/:user/gists"},
	{"GET", "/gists"},
	//{"GET", "/gists/public"},
	//{"GET", "/gists/starred"},
	{"GET", "/gists/:id"},
	{"POST", "/gists"},
	//{"PATCH", "/gists/:id"},
	{"PUT", "/gists/:id/star"},
	{"DELETE", "/gists/:id/star"},
	{"GET", "/gists/:id/star"},
	{"POST", "/gists/:id/forks"},
	{"DELETE", "/gists/:id"},

	// Git Data
	{"GET", "/repos/:owner/:repo/git/blobs/:sha"},
	{"POST", "/repos/:owner/:repo/git/blobs"},
	{"GET", "/repos/:owner/:repo/git/commits/:sha"},
	{"POST", "/repos/:owner/:repo/git/commits"},
	//{"GET", "/repos/:owner/:repo/git/refs/*ref"},
	{"GET", "/repos/:owner/:repo/git/refs"},
	{"POST", "/repos/:owner/:repo/git/refs"},
	//{"PATCH", "/repos/:owner/:repo/git/refs/*ref"},
	//{"DELETE", "/repos/:owner/:repo/git/refs/*ref"},
	{"GET", "/repos/:owner/:repo/git/tags/:sha"},
	{"POST", "/repos/:owner/:repo/git/tags"},
	{"GET", "/repos/:owner/:repo/git/trees/:sha"},
	{"POST", "/repos/:owner/:repo/git/trees"},

	// Issues
	{"GET", "/issues"},
	{"GET", "/user/issues"},
	{"GET", "/orgs/:org/issues"},
	{"GET", "/repos/:owner/:repo/issues"},
	{"GET", "/repos/:owner/:repo/issues/:number"},
	{"POST", "/repos/:owner/:repo/issues"},
	//{"PATCH", "/repos/:owner/:repo/issues/:number"},
	{"GET", "/repos/:owner/:repo/assignees"},
	{"GET", "/repos/:owner/:repo/assignees/:assignee"},
	{"GET", "/repos/:owner/:repo/issues/:number/comments"},
	//{"GET", "/repos/:owner/:repo/issues/comments"},
	//{"GET", "/repos/:owner/:repo/issues/comments/:id"},
	{"POST", "/repos/:owner/:repo/issues/:number/comments"},
	//{"PATCH", "/repos/:owner/:repo/issues/comments/:id"},
	//{"DELETE", "/repos/:owner/:repo/issues/comments/:id"},
	{"GET", "/repos/:owner/:repo/issues/:number/events"},
	//{"GET", "/repos/:owner/:repo/issues/events"},
	//{"GET", "/repos/:owner/:repo/issues/events/:id"},
	{"GET", "/repos/:owner/:repo/labels"},
	{"GET", "/repos/:owner/:repo/labels/:name"},
	{"POST", "/repos/:owner/:repo/labels"},
	//{"PATCH", "/repos/:owner/:repo/labels/:name"},
	{"DELETE", "/repos/:owner/:repo/labels/:name"},
	{"GET", "/repos/:owner/:repo/issues/:number/labels"},
	{"POST", "/repos/:owner/:repo/issues/:number/labels"},
	{"DELETE", "/repos/:owner/:repo/issues/:number/labels/:name"},
	{"PUT", "/repos/:owner/:repo/issues/:number/labels"},
	{"DELETE", "/repos/:owner/:repo/issues/:number/labels"},
	{"GET", "/repos/:owner/:repo/milestones/:number/labels"},
	{"GET", "/repos/:owner/:repo/milestones"},
	{"GET", "/repos/:owner/:repo/milestones/:number"},
	{"POST", "/repos/:owner/:repo/milestones"},
	//{"PATCH", "/repos/:owner/:repo/milestones/:number"},
	{"DELETE", "/repos/:owner/:repo/milestones/:number"},

	// Miscellaneous
	{"GET", "/emojis"},
	{"GET", "/gitignore/templates"},
	{"GET", "/gitignore/templates/:name"},
	{"POST", "/markdown"},
	{"POST", "/markdown/raw"},
	{"GET", "/meta"},
	{"GET", "/rate_limit"},

	// Organizations
	{"GET", "/users/:user/orgs"},
	{"GET", "/user/orgs"},
	{"GET", "/orgs/:org"},
	//{"PATCH", "/orgs/:org"},
	{"GET", "/orgs/:org/members"},
	{"GET", "/orgs/:org/members/:user"},
	{"DELETE", "/orgs/:org/members/:user"},
	{"GET", "/orgs/:org/public_members"},
	{"GET", "/orgs/:org/public_members/:user"},
	{"PUT", "/orgs/:org/public_members/:user"},
	{"DELETE", "/orgs/:org/public_members/:user"},
	{"GET", "/orgs/:org/teams"},
	{"GET", "/teams/:id"},
	{"POST", "/orgs/:org/teams"},
	//{"PATCH", "/teams/:id"},
	{"DELETE", "/teams/:id"},
	{"GET", "/teams/:id/members"},
	{"GET", "/teams/:id/members/:user"},
	{"PUT", "/teams/:id/members/:user"},
	{"DELETE", "/teams/:id/members/:user"},
	{"GET", "/teams/:id/repos"},
	{"GET", "/teams/:id/repos/:owner/:repo"},
	{"PUT", "/teams/:id/repos/:owner/:repo"},
	{"DELETE", "/teams/:id/repos/:owner/:repo"},
	{"GET", "/user/teams"},

	// Pull Requests
	{"GET", "/repos/:owner/:repo/pulls"},
	{"GET", "/repos/:owner/:repo/pulls/:number"},
	{"POST", "/repos/:owner/:repo/pulls"},
	//{"PATCH", "/repos/:owner/:repo/pulls/:number"},
	{"GET", "/repos/:owner/:repo/pulls/:number/commits"},
	{"GET", "/repos/:owner/:repo/pulls/:number/files"},
	{"GET", "/repos/:owner/:repo/pulls/:number/merge"},
	{"PUT", "/repos/:owner/:repo/pulls/:number/merge"},
	{"GET", "/repos/:owner/:repo/pulls/:number/comments"},
	//{"GET", "/repos/:owner/:repo/pulls/comments"},
	//{"GET", "/repos/:owner/:repo/pulls/comments/:number"},
	{"PUT", "/repos/:owner/:repo/pulls/:number/comments"},
	//{"PATCH", "/repos/:owner/:repo/pulls/comments/:number"},
	//{"DELETE", "/repos/:owner/:repo/pulls/comments/:number"},

	// Repositories
	{"GET", "/user/repos"},
	{"GET", "/users/:user/repos"},
	{"GET", "/orgs/:org/repos"},
	{"GET", "/repositories"},
	{"POST", "/user/repos"},
	{"POST", "/orgs/:org/repos"},
	{"GET", "/repos/:owner/:repo"},
	//{"PATCH", "/repos/:owner/:repo"},
	{"GET", "/repos/:owner/:repo/contributors"},
	{"GET", "/repos/:owner/:repo/languages"},
	{"GET", "/repos/:owner/:repo/teams"},
	{"GET", "/repos/:owner/:repo/tags"},
	{"GET", "/repos/:owner/:repo/branches"},
	{"GET", "/repos/:owner/:repo/branches/:branch"},
	{"DELETE", "/repos/:owner/:repo"},
	{"GET", "/repos/:owner/:repo/collaborators"},
	{"GET", "/repos/:owner/:repo/collaborators/:user"},
	{"PUT", "/repos/:owner/:repo/collaborators/:user"},
	{"DELETE", "/repos/:owner/:repo/collaborators/:user"},
	{"GET", "/repos/:owner/:repo/comments"},
	{"GET", "/repos/:owner/:repo/commits/:sha/comments"},
	{"POST", "/repos/:owner/:repo/commits/:sha/comments"},
	{"GET", "/repos/:owner/:repo/comments/:id"},
	//{"PATCH", "/repos/:owner/:repo/comments/:id"},
	{"DELETE", "/repos/:owner/:repo/comments/:id"},
	{"GET", "/repos/:owner/:repo/commits"},
	{"GET", "/repos/:owner/:repo/commits/:sha"},
	{"GET", "/repos/:owner/:repo/readme"},
	//{"GET", "/repos/:owner/:repo/contents/*path"},
	//{"PUT", "/repos/:owner/:repo/contents/*path"},
	//{"DELETE", "/repos/:owner/:repo/contents/*path"},
	//{"GET", "/repos/:owner/:repo/:archive_format/:ref"},
	{"GET", "/repos/:owner/:repo/keys"},
	{"GET", "/repos/:owner/:repo/keys/:id"},
	{"POST", "/repos/:owner/:repo/keys"},
	//{"PATCH", "/repos/:owner/:repo/keys/:id"},
	{"DELETE", "/repos/:owner/:repo/keys/:id"},
	{"GET", "/repos/:owner/:repo/downloads"},
	{"GET", "/repos/:owner/:repo/downloads/:id"},
	{"DELETE", "/repos/:owner/:repo/downloads/:id"},
	{"GET", "/repos/:owner/:repo/forks"},
	{"POST", "/repos/:owner/:repo/forks"},
	{"GET", "/repos/:owner/:repo/hooks"},
	{"GET", "/repos/:owner/:repo/hooks/:id"},
	{"POST", "/repos/:owner/:repo/hooks"},
	//{"PATCH", "/repos/:owner/:repo/hooks/:id"},
	{"POST", "/repos/:owner/:repo/hooks/:id/tests"},
	{"DELETE", "/repos/:owner/:repo/hooks/:id"},
	{"POST", "/repos/:owner/:repo/merges"},
	{"GET", "/repos/:owner/:repo/releases"},
	{"GET", "/repos/:owner/:repo/releases/:id"},
	{"POST", "/repos/:owner/:repo/releases"},
	//{"PATCH", "/repos/:owner/:repo/releases/:id"},
	{"DELETE", "/repos/:owner/:repo/releases/:id"},
	{"GET", "/repos/:owner/:repo/releases/:id/assets"},
	{"GET", "/repos/:owner/:repo/stats/contributors"},
	{"GET", "/repos/:owner/:repo/stats/commit_activity"},
	{"GET", "/repos/:owner/:repo/stats/code_frequency"},
	{"GET", "/repos/:owner/:repo/stats/participation"},
	{"GET", "/repos/:owner/:repo/stats/punch_card"},
	{"GET", "/repos/:owner/:repo/statuses/:ref"},
	{"POST", "/repos/:owner/:repo/statuses/:ref"},

	// Search
	{"GET", "/search/repositories"},
	{"GET", "/search/code"},
	{"GET", "/search/issues"},
	{"GET", "/search/users"},
	{"GET", "/legacy/issues/search/:owner/:repository/:state/:keyword"},
	{"GET", "/legacy/repos/search/:keyword"},
	{"GET", "/legacy/user/search/:keyword"},
	{"GET", "/legacy/user/email/:email"},

	// Users
	{"GET", "/users/:user"},
	{"GET", "/user"},
	//{"PATCH", "/user"},
	{"GET", "/users"},
	{"GET", "/user/emails"},
	{"POST", "/user/emails"},
	{"DELETE", "/user/emails"},
	{"GET", "/users/:user/followers"},
	{"GET", "/user/followers"},
	{"GET", "/users/:user/following"},
	{"GET", "/user/following"},
	{"GET", "/user/following/:user"},
	{"GET", "/users/:user/following/:target_user"},
	{"PUT", "/user/following/:user"},
	{"DELETE", "/user/following/:user"},
	{"GET", "/users/:user/keys"},
	{"GET", "/user/keys"},
	{"GET", "/user/keys/:id"},
	{"POST", "/user/keys"},
	//{"PATCH", "/user/keys/:id"},
	{"DELETE", "/user/keys/:id"},
}
var staticRoutes = []route{
	{"GET", "/"},
	{"GET", "/cmd.html"},
	{"GET", "/code.html"},
	{"GET", "/contrib.html"},
	{"GET", "/contribute.html"},
	{"GET", "/debugging_with_gdb.html"},
	{"GET", "/docs.html"},
	{"GET", "/effective_go.html"},
	{"GET", "/files.log"},
	{"GET", "/gccgo_contribute.html"},
	{"GET", "/gccgo_install.html"},
	{"GET", "/go-logo-black.png"},
	{"GET", "/go-logo-blue.png"},
	{"GET", "/go-logo-white.png"},
	{"GET", "/go1.1.html"},
	{"GET", "/go1.2.html"},
	{"GET", "/go1.html"},
	{"GET", "/go1compat.html"},
	{"GET", "/go_faq.html"},
	{"GET", "/go_mem.html"},
	{"GET", "/go_spec.html"},
	{"GET", "/help.html"},
	{"GET", "/ie.css"},
	{"GET", "/install-source.html"},
	{"GET", "/install.html"},
	{"GET", "/logo-153x55.png"},
	{"GET", "/Makefile"},
	{"GET", "/root.html"},
	{"GET", "/share.png"},
	{"GET", "/sieve.gif"},
	{"GET", "/tos.html"},
	{"GET", "/articles"},
	{"GET", "/articles/go_command.html"},
	{"GET", "/articles/index.html"},
	{"GET", "/articles/wiki"},
	{"GET", "/articles/wiki/edit.html"},
	{"GET", "/articles/wiki/final-noclosure.go"},
	{"GET", "/articles/wiki/final-noerror.go"},
	{"GET", "/articles/wiki/final-parsetemplate.go"},
	{"GET", "/articles/wiki/final-template.go"},
	{"GET", "/articles/wiki/final.go"},
	{"GET", "/articles/wiki/get.go"},
	{"GET", "/articles/wiki/http-sample.go"},
	{"GET", "/articles/wiki/index.html"},
	{"GET", "/articles/wiki/Makefile"},
	{"GET", "/articles/wiki/notemplate.go"},
	{"GET", "/articles/wiki/part1-noerror.go"},
	{"GET", "/articles/wiki/part1.go"},
	{"GET", "/articles/wiki/part2.go"},
	{"GET", "/articles/wiki/part3-errorhandling.go"},
	{"GET", "/articles/wiki/part3.go"},
	{"GET", "/articles/wiki/test.bash"},
	{"GET", "/articles/wiki/test_edit.good"},
	{"GET", "/articles/wiki/test_Test.txt.good"},
	{"GET", "/articles/wiki/test_view.good"},
	{"GET", "/articles/wiki/view.html"},
	{"GET", "/codewalk"},
	{"GET", "/codewalk/codewalk.css"},
	{"GET", "/codewalk/codewalk.js"},
	{"GET", "/codewalk/codewalk.xml"},
	{"GET", "/codewalk/functions.xml"},
	{"GET", "/codewalk/markov.go"},
	{"GET", "/codewalk/markov.xml"},
	{"GET", "/codewalk/pig.go"},
	{"GET", "/codewalk/popout.png"},
	{"GET", "/codewalk/run"},
	{"GET", "/codewalk/sharemem.xml"},
	{"GET", "/codewalk/urlpoll.go"},
	{"GET", "/devel"},
	{"GET", "/devel/release.html"},
	{"GET", "/devel/weekly.html"},
	{"GET", "/gopher"},
	{"GET", "/gopher/appenginegopher.jpg"},
	{"GET", "/gopher/appenginegophercolor.jpg"},
	{"GET", "/gopher/appenginelogo.gif"},
	{"GET", "/gopher/bumper.png"},
	{"GET", "/gopher/bumper192x108.png"},
	{"GET", "/gopher/bumper320x180.png"},
	{"GET", "/gopher/bumper480x270.png"},
	{"GET", "/gopher/bumper640x360.png"},
	{"GET", "/gopher/doc.png"},
	{"GET", "/gopher/frontpage.png"},
	{"GET", "/gopher/gopherbw.png"},
	{"GET", "/gopher/gophercolor.png"},
	{"GET", "/gopher/gophercolor16x16.png"},
	{"GET", "/gopher/help.png"},
	{"GET", "/gopher/pkg.png"},
	{"GET", "/gopher/project.png"},
	{"GET", "/gopher/ref.png"},
	{"GET", "/gopher/run.png"},
	{"GET", "/gopher/talks.png"},
	{"GET", "/gopher/pencil"},
	{"GET", "/gopher/pencil/gopherhat.jpg"},
	{"GET", "/gopher/pencil/gopherhelmet.jpg"},
	{"GET", "/gopher/pencil/gophermega.jpg"},
	{"GET", "/gopher/pencil/gopherrunning.jpg"},
	{"GET", "/gopher/pencil/gopherswim.jpg"},
	{"GET", "/gopher/pencil/gopherswrench.jpg"},
	{"GET", "/play"},
	{"GET", "/play/fib.go"},
	{"GET", "/play/hello.go"},
	{"GET", "/play/life.go"},
	{"GET", "/play/peano.go"},
	{"GET", "/play/pi.go"},
	{"GET", "/play/sieve.go"},
	{"GET", "/play/solitaire.go"},
	{"GET", "/play/tree.go"},
	{"GET", "/progs"},
	{"GET", "/progs/cgo1.go"},
	{"GET", "/progs/cgo2.go"},
	{"GET", "/progs/cgo3.go"},
	{"GET", "/progs/cgo4.go"},
	{"GET", "/progs/defer.go"},
	{"GET", "/progs/defer.out"},
	{"GET", "/progs/defer2.go"},
	{"GET", "/progs/defer2.out"},
	{"GET", "/progs/eff_bytesize.go"},
	{"GET", "/progs/eff_bytesize.out"},
	{"GET", "/progs/eff_qr.go"},
	{"GET", "/progs/eff_sequence.go"},
	{"GET", "/progs/eff_sequence.out"},
	{"GET", "/progs/eff_unused1.go"},
	{"GET", "/progs/eff_unused2.go"},
	{"GET", "/progs/error.go"},
	{"GET", "/progs/error2.go"},
	{"GET", "/progs/error3.go"},
	{"GET", "/progs/error4.go"},
	{"GET", "/progs/go1.go"},
	{"GET", "/progs/gobs1.go"},
	{"GET", "/progs/gobs2.go"},
	{"GET", "/progs/image_draw.go"},
	{"GET", "/progs/image_package1.go"},
	{"GET", "/progs/image_package1.out"},
	{"GET", "/progs/image_package2.go"},
	{"GET", "/progs/image_package2.out"},
	{"GET", "/progs/image_package3.go"},
	{"GET", "/progs/image_package3.out"},
	{"GET", "/progs/image_package4.go"},
	{"GET", "/progs/image_package4.out"},
	{"GET", "/progs/image_package5.go"},
	{"GET", "/progs/image_package5.out"},
	{"GET", "/progs/image_package6.go"},
	{"GET", "/progs/image_package6.out"},
	{"GET", "/progs/interface.go"},
	{"GET", "/progs/interface2.go"},
	{"GET", "/progs/interface2.out"},
	{"GET", "/progs/json1.go"},
	{"GET", "/progs/json2.go"},
	{"GET", "/progs/json2.out"},
	{"GET", "/progs/json3.go"},
	{"GET", "/progs/json4.go"},
	{"GET", "/progs/json5.go"},
	{"GET", "/progs/run"},
	{"GET", "/progs/slices.go"},
	{"GET", "/progs/timeout1.go"},
	{"GET", "/progs/timeout2.go"},
	{"GET", "/progs/update.bash"},
}
