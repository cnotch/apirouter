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
BenchmarkAero_Param               	22894290	        53.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param          	26056345	        49.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param              	 1000000	      1313 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param                	14033247	        86.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param         	  471302	      2514 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param        	 1845642	       661 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param         	11531433	       111 ns/op	      32 B/op	       1 allocs/op
BenchmarkAero_Param5              	15220465	        76.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param5         	16584570	        73.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param5             	  676971	      1514 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param5               	 7864692	       157 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param5        	  352486	      3550 ns/op	    1344 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param5       	 1618995	       747 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param5        	 3806864	       312 ns/op	     160 B/op	       1 allocs/op
BenchmarkAero_Param20             	33207046	        37.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param20        	 6746254	       178 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param20            	  368894	      3033 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param20              	 2918322	       418 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param20       	  133246	      8468 ns/op	    3452 B/op	      12 allocs/op
BenchmarkGowwwRouter_Param20      	 1133896	      1084 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param20       	 1000000	      1043 ns/op	     640 B/op	       1 allocs/op
BenchmarkAero_ParamWrite          	12175170	        94.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParamWrite     	12727315	        91.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParamWrite         	  926679	      1388 ns/op	     360 B/op	       4 allocs/op
BenchmarkGin_ParamWrite           	 7916719	       154 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParamWrite    	  509479	      2583 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParamWrite   	  605558	      1822 ns/op	     976 B/op	       8 allocs/op
BenchmarkHttpRouter_ParamWrite    	 8017930	       140 ns/op	      32 B/op	       1 allocs/op
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

BenchmarkAero_GithubStatic        	22273357	        56.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubStatic   	39102495	        29.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubStatic       	 1000000	      1290 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubStatic         	11974648	       101 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubStatic  	  206440	      5688 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GithubStatic 	14272084	        80.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GithubStatic  	26662975	        44.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GithubParam         	11351373	       104 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubParam    	13471605	        88.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubParam        	  797965	      1445 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubParam          	 6053378	       194 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubParam   	  143592	      8242 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GithubParam  	 1545063	       774 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GithubParam   	 4412121	       268 ns/op	      96 B/op	       1 allocs/op
BenchmarkAero_GithubAll           	   53400	     22543 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubAll      	   63091	     18703 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubAll          	    3766	    302612 ns/op	   71457 B/op	     609 allocs/op
BenchmarkGin_GithubAll            	   27432	     44130 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubAll     	     280	   3937812 ns/op	  251655 B/op	    1994 allocs/op
BenchmarkGowwwRouter_GithubAll    	   10000	    158819 ns/op	   72144 B/op	     501 allocs/op
BenchmarkHttpRouter_GithubAll     	   22683	     51872 ns/op	   13792 B/op	     167 allocs/op
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

BenchmarkAero_GPlusStatic         	28585231	        42.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusStatic    	50082670	        22.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusStatic        	 1000000	      1223 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusStatic          	15986409	        77.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusStatic   	  605961	      1917 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GPlusStatic  	36783783	        32.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GPlusStatic   	45615872	        27.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_GPlusParam          	17030224	        69.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusParam     	19166234	        64.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusParam         	  933674	      1347 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusParam           	 9858549	       120 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusParam    	  342358	      3336 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlusParam   	 1681168	       697 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlusParam    	 6956578	       169 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlus2Params        	11809809	        99.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlus2Params   	12207518	       101 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlus2Params       	  846004	      1445 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlus2Params         	 7012731	       167 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlus2Params  	  176025	      6073 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlus2Params 	 1583908	       724 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlus2Params  	 5960208	       207 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlusAll            	 1000000	      1038 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusAll       	 1264317	       960 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusAll           	   66380	     18229 ns/op	    4576 B/op	      39 allocs/op
BenchmarkGin_GPlusAll             	  699788	      1806 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusAll      	   22892	     51858 ns/op	   16112 B/op	     128 allocs/op
BenchmarkGowwwRouter_GPlusAll     	  140877	      8911 ns/op	    4752 B/op	      33 allocs/op
BenchmarkHttpRouter_GPlusAll      	  477932	      2391 ns/op	     640 B/op	      11 allocs/op
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

BenchmarkAero_ParseStatic         	27548072	        44.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseStatic    	44134420	        26.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseStatic        	 1000000	      1251 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseStatic          	14490985	        82.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseStatic   	  552333	      2308 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_ParseStatic  	37487745	        34.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_ParseStatic   	45597188	        27.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkAero_ParseParam          	18783025	        62.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseParam     	20307903	        62.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseParam         	  829724	      1324 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseParam           	12734770	        94.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseParam    	  442592	      2627 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParseParam   	 1775415	       690 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_ParseParam    	 7677872	       159 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_Parse2Params        	17173902	        71.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Parse2Params   	16235028	        75.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Parse2Params       	  872305	      1450 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Parse2Params         	 9602316	       123 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Parse2Params  	  426825	      3118 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_Parse2Params 	 1751478	       688 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Parse2Params  	 6404632	       184 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_ParseAll            	  616926	      1735 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseAll       	  919738	      1425 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseAll           	   34266	     33465 ns/op	    9152 B/op	      78 allocs/op
BenchmarkGin_ParseAll             	  382478	      3297 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseAll      	   10000	    103164 ns/op	   30288 B/op	     250 allocs/op
BenchmarkGowwwRouter_ParseAll     	   85484	     13188 ns/op	    6912 B/op	      48 allocs/op
BenchmarkHttpRouter_ParseAll      	  380132	      3050 ns/op	     640 B/op	      16 allocs/op
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

BenchmarkAero_StaticAll           	  116581	     10364 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_StaticAll      	  189546	      6477 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_StaticAll          	    5899	    226027 ns/op	   55265 B/op	     471 allocs/op
BenchmarkGin_StaticAll            	   42382	     29302 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_StaticAll     	    1166	   1054177 ns/op	  153236 B/op	    1413 allocs/op
BenchmarkGowwwRouter_StaticAll    	   60615	     19923 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_StaticAll     	   96732	     12071 ns/op	       0 B/op	       0 allocs/op
```
