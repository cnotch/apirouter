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

[aero] (https://github.com/aerogo/aero) performance is good, but it's more like a framwork. [httprouter] (https://github.com/julienschmidt/httprouter) functions are focusing, but its performance is mediocre. 

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
		apirouter.Handle("GET","/",func(w http.ResponseWriter, r *http.Request, ps apirouter.Params){
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
	apirouter.Handle("GET", `/user/{id=^\d+$}/books`,h),
	apirouter.Handle("GET", "/user/{id}",h),
	apirouter.Handle("GET", "/user/{id}:verb",h),
	apirouter.Handle("GET", "/user/{id}/profile/{theme}",h),
	apirouter.Handle("GET", "/images/{file=**}",h),
	apirouter.Handle("GET", "/images/{file=**}:jpg",h),
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

If the handler is registered with [HTTPHandle](https://godoc.org/github.com/cnotch/apirouter/#HTTPHandle) or [HTTPHandleFunc](https://godoc.org/github.com/cnotch/apirouter/#HTTPHandleFunc), it is stored in request's context and can be accessed by [apirouter.PathParams](https://godoc.org/github.com/cnotch/apirouter/#PathParams).

#### Named

A named parameter begins with `:` and matches any value until the next `/` or end of path.

Example, with a parameter `id`:

```Go
r:=apirouter.New(
	apirouter.GET("/streams/:id", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of stream #%s", id)
	}),

	apirouter.HTTPHandleFunc("GET","/users/:id", func(w http.ResponseWriter, r *http.Request) {
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
	apirouter.HTTPHandle("GET","/static/*filepath", 
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

You can use [HTTPHandle](https://godoc.org/github.com/cnotch/apirouter/#HTTPHandle) and [HTTPHandleFunc](https://godoc.org/github.com/cnotch/apirouter/#HTTPHandleFunc) to register the Handler for the standard library([http.Handler](https://golang.org/pkg/net/http/#Handler) or [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc))

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
BenchmarkAero_Param               	21540604	        53.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param          	25946310	        46.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param              	 1000000	      1297 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param                	13221682	        93.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param         	  429475	      2461 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param        	 1821579	       659 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param         	10643727	       107 ns/op	      32 B/op	       1 allocs/op
BenchmarkAero_Param5              	15731380	        76.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param5         	16932524	        70.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param5             	  816888	      1511 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param5               	 7328193	       159 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param5        	  335257	      3628 ns/op	    1344 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param5       	 1626409	       743 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param5        	 4022265	       309 ns/op	     160 B/op	       1 allocs/op
BenchmarkAero_Param20             	31488735	        38.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param20        	 6812640	       170 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param20            	  407150	      3056 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param20              	 2837382	       429 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param20       	  138084	      8653 ns/op	    3452 B/op	      12 allocs/op
BenchmarkGowwwRouter_Param20      	 1000000	      1068 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param20       	 1179908	      1018 ns/op	     640 B/op	       1 allocs/op
BenchmarkAero_ParamWrite          	12122565	        97.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParamWrite     	13888183	        89.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParamWrite         	  931020	      1360 ns/op	     360 B/op	       4 allocs/op
BenchmarkGin_ParamWrite           	 7708123	       157 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParamWrite    	  461530	      2515 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParamWrite   	  554713	      1829 ns/op	     976 B/op	       8 allocs/op
BenchmarkHttpRouter_ParamWrite    	 8600326	       145 ns/op	      32 B/op	       1 allocs/op
```

### GithubAPI Routes: 203

``` Shell
   Aero: 478312 Bytes
   ApiRouter: 76688 Bytes
   Beego: 150936 Bytes
   Gin: 58512 Bytes
   GorillaMux: 1322784 Bytes
   GowwwRouter: 80008 Bytes
   HttpRouter: 37096 Bytes

BenchmarkAero_GithubStatic        	22760636	        54.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubStatic   	42693703	        29.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubStatic       	 1000000	      1280 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubStatic         	12199152	       107 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubStatic  	  213040	      5766 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GithubStatic 	14731132	        84.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GithubStatic  	24394756	        45.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GithubParam         	11435451	       103 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubParam    	12438667	        99.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubParam        	  917167	      1423 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubParam          	 5982903	       203 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubParam   	  156993	      7988 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GithubParam  	 1560400	       776 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GithubParam   	 4653098	       263 ns/op	      96 B/op	       1 allocs/op
BenchmarkAero_GithubAll           	   53347	     21956 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubAll      	   60822	     19068 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubAll          	    4308	    301569 ns/op	   71457 B/op	     609 allocs/op
BenchmarkGin_GithubAll            	   26431	     44441 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubAll     	     300	   4052300 ns/op	  251655 B/op	    1994 allocs/op
BenchmarkGowwwRouter_GithubAll    	   10000	    156146 ns/op	   72144 B/op	     501 allocs/op
BenchmarkHttpRouter_GithubAll     	   23202	     52077 ns/op	   13792 B/op	     167 allocs/op
```

### GPlusAPI Routes : 13

```Shell
   Aero: 26552 Bytes
   ApiRouter: 30976 Bytes
   Beego: 10272 Bytes
   Gin: 4384 Bytes
   GorillaMux: 66208 Bytes
   GowwwRouter: 5744 Bytes
   HttpRouter: 2760 Bytes

BenchmarkAero_GPlusStatic         	31431427	        42.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusStatic    	56323669	        23.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusStatic        	  938733	      1218 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusStatic          	15607772	        80.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusStatic   	  632060	      1864 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GPlusStatic  	35895400	        32.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GPlusStatic   	42249679	        29.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GPlusParam          	17786320	        68.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusParam     	18365196	        65.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusParam         	  941380	      1409 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusParam           	 9359449	       128 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusParam    	  325118	      3277 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlusParam   	 1754960	       683 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlusParam    	 7090489	       170 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlus2Params        	11922658	        98.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlus2Params   	11186883	       107 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlus2Params       	  753024	      1464 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlus2Params         	 6780692	       178 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlus2Params  	  191918	      6229 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlus2Params 	 1669560	       714 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlus2Params  	 5449536	       206 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlusAll            	 1000000	      1026 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusAll       	 1210138	      1005 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusAll           	   65090	     18437 ns/op	    4576 B/op	      39 allocs/op
BenchmarkGin_GPlusAll             	  679156	      1841 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusAll      	   23200	     51648 ns/op	   16112 B/op	     128 allocs/op
BenchmarkGowwwRouter_GPlusAll     	  140528	      8605 ns/op	    4752 B/op	      33 allocs/op
BenchmarkHttpRouter_GPlusAll      	  436131	      2347 ns/op	     640 B/op	      11 allocs/op
```
### ParseAPI Routes: 26

```Shell
   Aero: 29304 Bytes
   ApiRouter: 37456 Bytes
   Beego: 19280 Bytes
   Gin: 7776 Bytes
   GorillaMux: 105880 Bytes
   GowwwRouter: 9344 Bytes
   HttpRouter: 5024 Bytes

BenchmarkAero_ParseStatic         	26644269	        46.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseStatic    	48720290	        26.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseStatic        	 1000000	      1265 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseStatic          	14162275	        84.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseStatic   	  531556	      2294 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_ParseStatic  	35058130	        33.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_ParseStatic   	43125240	        29.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_ParseParam          	20620428	        61.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseParam     	19883750	        61.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseParam         	  911560	      1423 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseParam           	12397555	       100 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseParam    	  389101	      2803 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParseParam   	 1628454	       739 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_ParseParam    	 7091450	       167 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_Parse2Params        	16377946	        73.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Parse2Params   	16628174	        73.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Parse2Params       	  890661	      1386 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Parse2Params         	10027630	       125 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Parse2Params  	  371895	      3150 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_Parse2Params 	 1740235	       710 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Parse2Params  	 6144558	       198 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_ParseAll            	  633602	      1741 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseAll       	  904820	      1474 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseAll           	   35168	     33860 ns/op	    9152 B/op	      78 allocs/op
BenchmarkGin_ParseAll             	  344960	      3339 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseAll      	   10000	    115180 ns/op	   30288 B/op	     250 allocs/op
BenchmarkGowwwRouter_ParseAll     	   84831	     15241 ns/op	    6912 B/op	      48 allocs/op
BenchmarkHttpRouter_ParseAll      	  371112	      3221 ns/op	     640 B/op	      16 allocs/op
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

BenchmarkAero_StaticAll           	  114135	     10390 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_StaticAll      	  190582	      6254 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_StaticAll          	    6160	    227035 ns/op	   55265 B/op	     471 allocs/op
BenchmarkGin_StaticAll            	   43140	     29442 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_StaticAll     	    1202	   1038774 ns/op	  153236 B/op	    1413 allocs/op
BenchmarkGowwwRouter_StaticAll    	   59642	     20368 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_StaticAll     	  103846	     12098 ns/op	       0 B/op	       0 allocs/op
```
