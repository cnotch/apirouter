ApiRouter
=========
Package [apirouter](https://godoc.org/github.com/cnotch/apirouter) provides a lightning fast RESTful api router.

My English is too poor. The project documentation and code comments are mainly modeled after the following projects:
+ [gowwwrouter](https://github.com/gowww/router)
+ [aero](https://github.com/aerogo/aero)
+ [httprouter](https://github.com/julienschmidt/httprouter)

Thanks to these open source projects.

## Motivation
When developing server-side applications, it is necessary to provide RESTful APIs. I would like to have a library:
+ Simple: focus on path parameter extraction and routing
+ Fast: better performance

[aero](https://github.com/aerogo/aero) performance is good, but it's more like a framwork. [httprouter](https://github.com/julienschmidt/httprouter) functions are focusing, but its performance is mediocre. 

I had to write one myself as an exercise

## Features

- Best Performance: [Benchmarks speak for themselves](#benchmarks)
- Compatibility with the [http.Handler](https://golang.org/pkg/net/http/#Handler) interface
- Named parameters, regular expressions parameters and wildcard parameters
- Support google RESTful api style
- Smart prioritized routes
- No allocations, matching and retrieve parameters don't allocates.

## Installing

1. Get package:

	```Shell
	go get -u github.com/cnotch/apirouter
	```

2. Import it in your code:

	```Go
	import "github.com/cnotch/apirouter"
	```

## Usage

1. Make a new router:

	```Go
	r := apirouter.New(
		apirouter.API("GET","/",func(w http.ResponseWriter, r *http.Request, ps apirouter.Params){
			fmt.Fprint(w, "Hello")
		}),
	)
	```

	Remember that HTTP methods are case-sensitive and uppercase by convention ([RFC 7231 4.1](https://tools.ietf.org/html/rfc7231#section-4.1)).  
	So you can directly use the built-in shortcuts for standard HTTP methods, such as [apirouter.GET](https://godoc.org/github.com/cnotch/apirouter#GET)...

2. Give the router to the server:

	```Go
	http.ListenAndServe(":8080", r)
	```
### Precedence example

On the example below the router will test the routes in the following order, /users/list then /users/:id=^\d+$ then /users/:id then /users/*page.

```go
r:= apirouter.New(
	apirouter.GET("/users/:id",...),
	apirouter.GET(`/users/:id=^\d+$`,...),
	apirouter.GET("/users/*page",...),
	apirouter.GET("/users/list",...),
)
```

### Pattern Styles

### Default style
On the example below the router will use default style.

```go
r:= apirouter.New(...)
```

or

```go
r:= apirouter.New(apirouter.DefaultStyle,...)
```

Default sytle syntax:

```Shell
Pattern		= "/" Segments
Segments	= Segment { "/" Segment }
Segment		= LITERAL | Parameter
Parameter	= Anonymous | Named
Anonymous	= ":" | "*"
Named		= ":" FieldPath [ "=" Regexp ] | "*" FieldPath
FieldPath	= IDENT { "." IDENT }
```

#### Google style
On the example below the router will use google style.

```go
r:= apirouter.New(apirouter.GoogleStyle,
	apirouter.API("GET", `/user/{id=^\d+$}/books`,h),
	apirouter.API("GET", "/user/{id}",h),
	apirouter.API("GET", "/user/{id}:verb",h),
	apirouter.API("GET", "/user/{id}/profile/{theme}",h),
	apirouter.API("GET", "/images/{file=**}",h),
	apirouter.API("GET", "/images/{file=**}:jpg",h),
)
```

Google sytle syntax:

```Shell
Pattern		= "/" Segments [ Verb ] ;
Segments	= Segment { "/" Segment } ;
Segment		= LITERAL | Parameter
Parameter	= Anonymous | Named
Anonymous	= "*" | "**"
Named		= "{" FieldPath [ "=" Wildcard ] "}"
Wildcard	= "*" | "**" | Regexp
FieldPath	= IDENT { "." IDENT } ;
Verb		= ":" LITERAL ;
```

### Parameters

The value of parameters is saved as a [Params](https://godoc.org/github.com/cnotch/apirouter/#Params). The Params is passed to the [Handler](https://godoc.org/github.com/cnotch/apirouter/#Handler) func as a third parameter.

If the handler is registered with [Handle](https://godoc.org/github.com/cnotch/apirouter/#Handle) or [HandleFunc](https://godoc.org/github.com/cnotch/apirouter/#HandleFunc), it is stored in request's context and can be accessed by [apirouter.PathParams](https://godoc.org/github.com/cnotch/apirouter/#PathParams).

#### Named

A named parameter begins with `:` and matches any value until the next `/` or end of path.

Example, with a parameter `id`:

```Go
r:=apirouter.New(
	apirouter.GET("/streams/:id", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of stream #%s", id)
	}),

	apirouter.HandleFunc("GET","/users/:id", func(w http.ResponseWriter, r *http.Request) {
		ps := apirouter.PathParams(r.Context())
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of user #%s", id)
	}),
)
```

If you don't need to retrieve the parameter value by name, you can omit the parameter name:

```Go
r:=apirouter.New(
	apirouter.GET(`/users/:`, func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.Value(0)
		fmt.Fprintf(w, "Page of user #%s", id)
	}),
)
```

<details>
<summary>No surprise</summary>

A parameter can be used on the same level as a static route, without conflict:

```Go
r:=apirouter.New(
	apirouter.GET("/users/admin", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		fmt.Fprint(w, "admin page")
	}),

	apirouter.GET("/users/:id", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of user #%s", id)
	}),
)
```
</details>

#### Regular expressions

If a parameter must match an exact pattern (digits only, for example), you can also set a [regular expression](https://golang.org/pkg/regexp/syntax) constraint just after the parameter name and `=`:

```Go
r:=apirouter.New(
	apirouter.GET(`/users/:id=^\d+$`, func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of user #%s", id)
	}),
)
```

**NOTE:** No more than 256 different regular expressions are allowed.

**WARN:** Regular expressions can significantly reduce performance.

<details>
<summary>No surprise</summary>

A parameter with a regular expression can be used on the same level as a simple parameter, without conflict:

```Go
r:=apirouter.New(
	apirouter.GET(`/users/:id=^\d+$`, func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of user #%s", id)
	}),

	apirouter.GET("/users/:id", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of user #%s", id)
	}),
)
```

</details>

#### Wildcard

Wildcard parameters match anything until the path end, not including the directory index (the '/' before the '*'). Since they match anything until the end, wildcard parameters must always be the final path element.

The rest of the request path becomes the parameter value of `*`:

```Go
r:=apirouter.New(
	apirouter.GET("/files/*filepath", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		filepath := ps.ByName("filepath")
		fmt.Fprintf(w, "Get file %s", filepath)
	}),
)
```

<details>
<summary>No surprise</summary>

Deeper route paths with the same prefix as the wildcard will take precedence, without conflict:

```Go
// Will match:
// 	/files/one
// 	/files/two
// 	...
r:=apirouter.New(
	apirouter.GET("/files/:name", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		name := ps.ByName("name")
		fmt.Fprintf(w, "Get root file #%s", name)
	}),
)

// Will match:
// 	/files/one/...
// 	/files/two/...
// 	...
r:=apirouter.New(
	apirouter.GET("/files/*filepath", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		filepath := ps.ByName("filepath")
		fmt.Fprintf(w, "Get file %s", filepath)
	}),
)

// Will match:
// 	/files/movies/one
// 	/files/movies/two
// 	...
r:=apirouter.New(
	apirouter.GET("/files/movies/:name", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		name := ps.ByName("name")
		fmt.Fprintf(w, "Get movie #%s", name)
	}),
)
```
</details>

### Static files

For serving static files, like for the standard [net/http.ServeMux](https://golang.org/pkg/net/http#ServeMux), just bring your own handler.

Example, with the standard [net/http.FileServer](https://golang.org/pkg/net/http#FileServer):

```Go
r:=apirouter.New(
	apirouter.Handle("GET","/static/*filepath", 
		http.StripPrefix("/static/", http.FileServer(http.Dir("static")))),
)
```

### Custom "not found" handler

When a request match no route, the response status is set to 404 and an empty body is sent by default.

But you can set your own "not found" handler.  
In this case, it's up to you to set the response status code (normally 404):

```Go
r:=apirouter.New(
	apirouter.NotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})),
)
```

### Work with "http.Handler"

You can use [Handle](https://godoc.org/github.com/cnotch/apirouter/#Handle) and [HandleFunc](https://godoc.org/github.com/cnotch/apirouter/#HandleFunc) to register the Handler for the standard library([http.Handler](https://golang.org/pkg/net/http/#Handler) or [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc))

**NOTE:** Since the Handler using the standard library needs to add a new context to the Request, performance can suffer.

## Benchmarks

### Environment

```Shell
goos: darwin
goarch: amd64
pkg: github.com/julienschmidt/go-http-routing-benchmark
```

### Single Route

``` Shell
BenchmarkAero_Param               	24144682	        52.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param          	24836985	        49.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param              	  964224	      1323 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param                	12476570	        88.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param         	  432448	      2489 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param        	 1860969	       660 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param         	10968362	       106 ns/op	      32 B/op	       1 allocs/op
BenchmarkAero_Param5              	15488991	        79.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param5         	17314315	        68.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param5             	  778066	      1513 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param5               	 7728501	       158 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param5        	  351091	      3606 ns/op	    1344 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param5       	 1606382	       742 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param5        	 3943095	       314 ns/op	     160 B/op	       1 allocs/op
BenchmarkAero_Param20             	32496225	        37.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param20        	 6949796	       171 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param20            	  347565	      3031 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param20              	 2845663	       422 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param20       	  133251	      8543 ns/op	    3452 B/op	      12 allocs/op
BenchmarkGowwwRouter_Param20      	 1000000	      1065 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param20       	 1000000	      1079 ns/op	     640 B/op	       1 allocs/op
BenchmarkAero_ParamWrite          	12568646	        97.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParamWrite     	13623519	        90.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParamWrite         	  934646	      1345 ns/op	     360 B/op	       4 allocs/op
BenchmarkGin_ParamWrite           	 7493145	       159 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParamWrite    	  442160	      2581 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParamWrite   	  608452	      1819 ns/op	     976 B/op	       8 allocs/op
BenchmarkHttpRouter_ParamWrite    	 8014755	       140 ns/op	      32 B/op	       1 allocs/op
```

### GithubAPI Routes: 203

``` Shell
   Aero: 506040 Bytes
   ApiRouter: 89264 Bytes
   Beego: 150936 Bytes
   Gin: 58512 Bytes
   GorillaMux: 1322784 Bytes
   GowwwRouter: 80008 Bytes
   HttpRouter: 37096 Bytes

BenchmarkAero_GithubStatic        	24764742	        52.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubStatic   	44066424	        29.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubStatic       	 1000000	      1260 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubStatic         	12070008	       108 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubStatic  	  213408	      5478 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GithubStatic 	13270420	        92.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GithubStatic  	24445467	        45.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GithubParam         	10188027	       118 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubParam    	14502672	        84.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubParam        	  971031	      1502 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubParam          	 5943706	       201 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubParam   	  146402	      8306 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GithubParam  	 1522910	       777 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GithubParam   	 4485136	       278 ns/op	      96 B/op	       1 allocs/op
BenchmarkAero_GithubAll           	   52974	     22689 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubAll      	   65557	     18919 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubAll          	    3519	    298226 ns/op	   71457 B/op	     609 allocs/op
BenchmarkGin_GithubAll            	   27138	     45814 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubAll     	     307	   3911106 ns/op	  251655 B/op	    1994 allocs/op
BenchmarkGowwwRouter_GithubAll    	   10000	    155828 ns/op	   72144 B/op	     501 allocs/op
BenchmarkHttpRouter_GithubAll     	   23499	     51290 ns/op	   13792 B/op	     167 allocs/op
```

### GPlusAPI Routes : 13

```Shell
   Aero: 26552 Bytes
   ApiRouter: 31952 Bytes
   Beego: 10272 Bytes
   Gin: 4384 Bytes
   GorillaMux: 66208 Bytes
   GowwwRouter: 5744 Bytes
   HttpRouter: 2760 Bytes

BenchmarkAero_GPlusStatic         	27960792	        41.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusStatic    	57509564	        22.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusStatic        	 1000000	      1231 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusStatic          	14276492	        80.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusStatic   	  569024	      1899 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GPlusStatic  	34480376	        34.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GPlusStatic   	41569572	        26.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GPlusParam          	18108130	        69.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusParam     	19340407	        63.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusParam         	  938853	      1335 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusParam           	 9421104	       131 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusParam    	  335938	      3297 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlusParam   	 1770891	       683 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlusParam    	 6602436	       176 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlus2Params        	11700950	        99.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlus2Params   	12474241	        98.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlus2Params       	  862087	      1454 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlus2Params         	 6744548	       175 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlus2Params  	  185772	      6279 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlus2Params 	 1646191	       715 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlus2Params  	 5442988	       209 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlusAll            	 1000000	      1078 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusAll       	 1255639	       986 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusAll           	   64459	     18017 ns/op	    4576 B/op	      39 allocs/op
BenchmarkGin_GPlusAll             	  570110	      1930 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusAll      	   22620	     51007 ns/op	   16112 B/op	     128 allocs/op
BenchmarkGowwwRouter_GPlusAll     	  137460	      8661 ns/op	    4752 B/op	      33 allocs/op
BenchmarkHttpRouter_GPlusAll      	  485757	      2412 ns/op	     640 B/op	      11 allocs/op
```

### ParseAPI Routes: 26

```Shell
   Aero: 29304 Bytes
   ApiRouter: 38608 Bytes
   Beego: 19280 Bytes
   Gin: 7776 Bytes
   GorillaMux: 105880 Bytes
   GowwwRouter: 9344 Bytes
   HttpRouter: 5024 Bytes

BenchmarkAero_ParseStatic         	24398053	        52.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseStatic    	49408676	        25.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseStatic        	 1000000	      1278 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseStatic          	14866940	        87.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseStatic   	  483002	      2423 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_ParseStatic  	35223200	        33.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_ParseStatic   	42254116	        28.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_ParseParam          	17067716	        64.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseParam     	20084414	        59.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseParam         	  939766	      1334 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseParam           	11693838	       102 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseParam    	  404385	      2593 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParseParam   	 1800579	       670 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_ParseParam    	 7680414	       152 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_Parse2Params        	17208108	        74.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Parse2Params   	17270054	        72.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Parse2Params       	  836359	      1367 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Parse2Params         	 9392289	       130 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Parse2Params  	  385114	      3097 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_Parse2Params 	 1761308	       718 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Parse2Params  	 6646308	       183 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_ParseAll            	  693284	      1813 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseAll       	  936142	      1400 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseAll           	   35727	     33790 ns/op	    9152 B/op	      78 allocs/op
BenchmarkGin_ParseAll             	  333061	      3477 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseAll      	   10000	    106275 ns/op	   30288 B/op	     250 allocs/op
BenchmarkGowwwRouter_ParseAll     	   90518	     13095 ns/op	    6912 B/op	      48 allocs/op
BenchmarkHttpRouter_ParseAll      	  419498	      3036 ns/op	     640 B/op	      16 allocs/op
```

### Static Routes: 157

```Shell
   Aero: 34536 Bytes
   ApiRouter: 29040 Bytes
   Beego: 98456 Bytes
   Gin: 34936 Bytes
   GorillaMux: 585632 Bytes
   GowwwRouter: 24968 Bytes
   HttpRouter: 21680 Bytes

BenchmarkAero_StaticAll           	  118082	     10180 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_StaticAll      	  184347	      6248 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_StaticAll          	    5565	    224430 ns/op	   55265 B/op	     471 allocs/op
BenchmarkGin_StaticAll            	   40398	     29213 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_StaticAll     	    1224	   1051964 ns/op	  153236 B/op	    1413 allocs/op
BenchmarkGowwwRouter_StaticAll    	   58590	     20201 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_StaticAll     	  103761	     11498 ns/op	       0 B/op	       0 allocs/op
```
