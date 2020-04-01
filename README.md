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
BenchmarkAero_Param               	23852071	        53.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param          	29387841	        44.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param              	  894325	      1298 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param                	12475012	        91.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param         	  418878	      2562 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param        	 1800291	       670 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param         	10597258	       108 ns/op	      32 B/op	       1 allocs/op
BenchmarkAero_Param5              	14609031	        85.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param5         	18887922	        63.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param5             	  762898	      1548 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param5               	 7427611	       158 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param5        	  301797	      3620 ns/op	    1344 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param5       	 1593031	       745 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param5        	 3837132	       324 ns/op	     160 B/op	       1 allocs/op
BenchmarkAero_Param20             	31909300	        37.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param20        	 7497290	       161 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param20            	  402992	      3071 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param20              	 2887786	       412 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param20       	  133100	      8533 ns/op	    3452 B/op	      12 allocs/op
BenchmarkGowwwRouter_Param20      	 1000000	      1115 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param20       	 1000000	      1098 ns/op	     640 B/op	       1 allocs/op
BenchmarkAero_ParamWrite          	13123207	        92.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParamWrite     	14104640	        85.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParamWrite         	  861788	      1402 ns/op	     360 B/op	       4 allocs/op
BenchmarkGin_ParamWrite           	 8369965	       155 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParamWrite    	  433839	      2586 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParamWrite   	  615397	      1848 ns/op	     976 B/op	       8 allocs/op
BenchmarkHttpRouter_ParamWrite    	 8072284	       148 ns/op	      32 B/op	       1 allocs/op
```

### GithubAPI Routes: 203

``` Shell
   Aero: 476616 Bytes
   ApiRouter: 76896 Bytes
   Beego: 150936 Bytes
   Gin: 58512 Bytes
   GorillaMux: 1322784 Bytes
   GowwwRouter: 80008 Bytes
   HttpRouter: 37096 Bytes

BenchmarkAero_GithubStatic        	22462923	        53.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubStatic   	36505447	        31.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubStatic       	 1000000	      1290 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubStatic         	11429608	       103 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubStatic  	  180069	      5845 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GithubStatic 	15091183	        80.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GithubStatic  	25893327	        44.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GithubParam         	11979506	       101 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubParam    	15567512	        82.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubParam        	  906562	      1458 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubParam          	 5954107	       198 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubParam   	  155583	      8280 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GithubParam  	 1538620	       770 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GithubParam   	 4556206	       260 ns/op	      96 B/op	       1 allocs/op
BenchmarkAero_GithubAll           	   55035	     22052 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubAll      	   64652	     17741 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubAll          	    3372	    302814 ns/op	   71457 B/op	     609 allocs/op
BenchmarkGin_GithubAll            	   27670	     43803 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubAll     	     278	   3895173 ns/op	  251655 B/op	    1994 allocs/op
BenchmarkGowwwRouter_GithubAll    	   10000	    156772 ns/op	   72144 B/op	     501 allocs/op
BenchmarkHttpRouter_GithubAll     	   23284	     51159 ns/op	   13792 B/op	     167 allocs/op

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

BenchmarkAero_GPlusStatic         	29818162	        40.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusStatic    	48817503	        22.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusStatic        	 1000000	      1274 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusStatic          	14494784	        77.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusStatic   	  683160	      1909 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GPlusStatic  	35441674	        33.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GPlusStatic   	36591706	        29.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GPlusParam          	16693532	        71.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusParam     	19121349	        60.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusParam         	  777931	      1366 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusParam           	 9632288	       122 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusParam    	  346806	      3360 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlusParam   	 1703626	       682 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlusParam    	 6702982	       179 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlus2Params        	11441380	       106 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlus2Params   	12883824	        94.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlus2Params       	  798392	      1463 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlus2Params         	 6891424	       176 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlus2Params  	  186945	      6475 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlus2Params 	 1584742	       716 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlus2Params  	 5746395	       216 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlusAll            	 1000000	      1016 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusAll       	 1313586	       882 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusAll           	   65610	     18270 ns/op	    4576 B/op	      39 allocs/op
BenchmarkGin_GPlusAll             	  668157	      1853 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusAll      	   22755	     52274 ns/op	   16112 B/op	     128 allocs/op
BenchmarkGowwwRouter_GPlusAll     	  132283	      8770 ns/op	    4752 B/op	      33 allocs/op
BenchmarkHttpRouter_GPlusAll      	  428167	      2438 ns/op	     640 B/op	      11 allocs/op

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

BenchmarkAero_ParseStatic         	27687204	        45.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseStatic    	45673702	        28.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseStatic        	 1000000	      1269 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseStatic          	14970594	        83.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseStatic   	  527785	      2369 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_ParseStatic  	34471850	        33.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_ParseStatic   	41460633	        28.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_ParseParam          	20797064	        60.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseParam     	20923816	        56.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseParam         	  738157	      1356 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseParam           	12282522	        92.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseParam    	  448134	      2623 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParseParam   	 1778294	       683 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_ParseParam    	 7670318	       163 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_Parse2Params        	16204402	        72.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Parse2Params   	18347480	        69.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Parse2Params       	  879038	      1427 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Parse2Params         	 9939373	       121 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Parse2Params  	  399722	      3132 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_Parse2Params 	 1638937	       703 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Parse2Params  	 6475725	       187 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_ParseAll            	  716336	      1782 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseAll       	  827854	      1368 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseAll           	   35646	     33790 ns/op	    9152 B/op	      78 allocs/op
BenchmarkGin_ParseAll             	  350886	      3476 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseAll      	   10000	    107983 ns/op	   30288 B/op	     250 allocs/op
BenchmarkGowwwRouter_ParseAll     	   87190	     13315 ns/op	    6912 B/op	      48 allocs/op
BenchmarkHttpRouter_ParseAll      	  406676	      3022 ns/op	     640 B/op	      16 allocs/op

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

BenchmarkAero_StaticAll           	  119220	     10581 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_StaticAll      	  194785	      6237 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_StaticAll          	    5282	    226977 ns/op	   55265 B/op	     471 allocs/op
BenchmarkGin_StaticAll            	   42614	     28440 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_StaticAll     	    1094	   1159951 ns/op	  153236 B/op	    1413 allocs/op
BenchmarkGowwwRouter_StaticAll    	   60482	     20263 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_StaticAll     	   94354	     12376 ns/op	       0 B/op	       0 allocs/op

```
