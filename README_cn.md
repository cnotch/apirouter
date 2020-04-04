ApiRouter
=========
[apirouter](https://godoc.org/github.com/cnotch/apirouter)提供了一个轻量级高效的 RESTful APIs 路由器。

## 动机

在开发服务端应用时，提供 RESTful APIs 是必要的。希望有这么一个库：
+ 简单：聚焦于路径参数的提取和路由
+ 高效：良好的性能

[aero](https://github.com/aerogo/aero) 性能不错，但它更像一个框架。 [httprouter](https://github.com/julienschmidt/httprouter) 比较聚焦，但性能一般般。

最后，决定自己开发一个，权当练习。

## 特性

- 极佳的性能： [性能报告](#benchmarks)
- 和标准库 [http.Handler](https://golang.org/pkg/net/http/#Handler) 兼容
- 支持匿名参数，命名参数，正则表达式参数和通配参数
- 支持 gRPC 风格的路径匹配规则
- 智能最佳匹配路由
- 匹配和接收路径参数无需分配内存

## 安装

1. 获取包：

	```Shell
	go get -u github.com/cnotch/apirouter
	```

2. 导入：

	```Go
	import "github.com/cnotch/apirouter"
	```

## 使用

建议和 [http.ServeMux](https://godoc.org/github.com/cnotch/apirouter/#ServeMux) 配合使用，充分利用标准库。

1. 创建一个路由器：

	```Go
	r := apirouter.New(
		apirouter.API("GET","/",func(w http.ResponseWriter, r *http.Request, ps apirouter.Params){
			fmt.Fprint(w, "Hello")
		}),
	)
	```

	HTTP 方法是区分大小写的 ([RFC 7231 4.1](https://tools.ietf.org/html/rfc7231#section-4.1)).  
	对标准的 HTTP 方法可以使用快捷函数，诸如：[apirouter.GET](https://godoc.org/github.com/cnotch/apirouter#GET)...

2. 将路由器分配到服务实例：

	```Go
	http.ListenAndServe(":8080", r)
	```

### 优先级

以下例子安装如下顺序检测路由：/users/list 、 /users/:id=^\d+$ 、 /users/:id 、 /users/*page.

```go
r:= apirouter.New(
	apirouter.GET("/users/:id",...),
	apirouter.GET(`/users/:id=^\d+$`,...),
	apirouter.GET("/users/*page",...),
	apirouter.GET("/users/list",...),
)
```

### 模式字串风格

### 默认风格
以下例子都是按默认风格解析模式字串：

```go
r:= apirouter.New(
	apirouter.API("GET", `/user/:id=^\d+$/books`,h),
	apirouter.API("GET", "/user/:id",h),
	apirouter.API("GET", "/user/:id/profile/:theme",h),
	apirouter.API("GET", "/images/*file",h),
)
```

默认风格语法：

```Shell
Pattern		= "/" Segments
Segments	= Segment { "/" Segment }
Segment		= LITERAL | Parameter
Parameter	= Anonymous | Named
Anonymous	= ":" | "*"
Named		= ":" FieldPath [ "=" Regexp ] | "*" FieldPath
FieldPath	= IDENT { "." IDENT }
```

#### gRPC 风格
以下例子使用 gRPC 风格：

```go
r:= apirouter.NewForGRPC(
	apirouter.API("GET", `/user/{id=^\d+$}/books`,h),
	apirouter.API("GET", "/user/{id}",h),
	apirouter.API("GET", "/user/{id}:verb",h),
	apirouter.API("GET", "/user/{id}/profile/{theme}",h),
	apirouter.API("GET", "/images/{file=**}",h),
	apirouter.API("GET", "/images/{file=**}:jpg",h),
)
```

gRPC 风格语法：

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

### 参数

参数值存储在 [Params](https://godoc.org/github.com/cnotch/apirouter/#Params) 中。 Params 作为第三个参数传递给函数 [Handler](https://godoc.org/github.com/cnotch/apirouter/#Handler).

如果使用 [Handle](https://godoc.org/github.com/cnotch/apirouter/#Handle) 或 [HandleFunc](https://godoc.org/github.com/cnotch/apirouter/#HandleFunc) 注册请求处理程序，它存储在请求上下文中并可以通过 [apirouter.PathParams](https://godoc.org/github.com/cnotch/apirouter/#PathParams) 获得。

#### 命名参数

命名参数以 `:` 开始，它匹配后续任意字符，直到路径结束或遇到下一个 `/`。

例子（包含参数`id`）：

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

如果不关心参数名，可以省略参数名：

```Go
r:=apirouter.New(
	apirouter.GET(`/users/:`, func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.Value(0)
		fmt.Fprintf(w, "Page of user #%s", id)
	}),
)
```

同级路径段中，静态值和命名参数不会产生冲突：

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

#### 正则表达式参数

如果参数需要精确匹配（比如仅数字），可以设置一个正则表达式参数。它在命名参数和`=` 后设置。

```Go
r:=apirouter.New(
	apirouter.GET(`/users/:id=^\d+$`, func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		id := ps.ByName("id")
		fmt.Fprintf(w, "Page of user #%s", id)
	}),
)
```

**NOTE:** 路由器中支持不超过 256 个不同的正则表达式。

**WARN:** 正则表达式将大幅降低性能。

同级路径段中，静态值、命名参数和正则表达式参数不会产生冲突：

```Go
r:=apirouter.New(
	apirouter.GET("/users/admin", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		fmt.Fprint(w, "admin page")
	}),

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

#### 通配参数

通配参数匹配任何字符直到路径结束，但不包含通配符前导 `/`。通配参数只能出现在模式字串的最后段中。

```Go
r:=apirouter.New(
	apirouter.GET("/files/*filepath", func(w http.ResponseWriter, r *http.Request, ps apirouter.Params) {
		filepath := ps.ByName("filepath")
		fmt.Fprintf(w, "Get file %s", filepath)
	}),
)
```

具有与通配符相同前缀的更深层路由路径将优先，且不存在冲突:

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

### 静态文件

和 [net/http.ServeMux](https://golang.org/pkg/net/http#ServeMux)类似。

以下例子使用标准库中 [net/http.FileServer](https://golang.org/pkg/net/http#FileServer):

```Go
r:=apirouter.New(
	apirouter.Handle("GET","/static/*filepath", 
		http.StripPrefix("/static/", http.FileServer(http.Dir("static")))),
)
```

### 自定义 “未找到” 处理器

当请求没有匹配的路由时，响应状态被设置为 404，并且默认发送一个空的主体。

但是您可以设置自己的“未找到”处理程序。
在这种情况下，由您来设置响应状态代码(通常为404):

```Go
r:=apirouter.New(
	apirouter.NotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})),
)
```

### 和 "http.Handler" 协同工作

可以使用 [Handle](https://godoc.org/github.com/cnotch/apirouter/#Handle) 和 [HandleFunc](https://godoc.org/github.com/cnotch/apirouter/#HandleFunc) 来注册标准库的 ([http.Handler](https://golang.org/pkg/net/http/#Handler) 或 [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc))

**NOTE:** 使用标准库需要添加新的上下文，对性能有一定的影响。

## Benchmarks

### Environment

```Shell
goos: darwin
goarch: amd64
pkg: github.com/julienschmidt/go-http-routing-benchmark
```

### Single Route

``` Shell
BenchmarkAero_Param               	23004813	        52.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param          	26834293	        44.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param              	  908820	      1292 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param                	13086147	        86.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param         	  458750	      2508 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param        	 1760046	       669 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param         	10678444	       106 ns/op	      32 B/op	       1 allocs/op

BenchmarkAero_Param5              	15816512	        78.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param5         	18431535	        64.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param5             	  771405	      1506 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param5               	 7365034	       155 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param5        	  323251	      3662 ns/op	    1344 B/op	      10 allocs/op
BenchmarkGowwwRouter_Param5       	 1602684	       749 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param5        	 3771800	       313 ns/op	     160 B/op	       1 allocs/op

BenchmarkAero_Param20             	32261661	        36.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Param20        	 7118907	       167 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Param20            	  375331	      3003 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Param20              	 2895128	       418 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Param20       	  137173	      8534 ns/op	    3451 B/op	      12 allocs/op
BenchmarkGowwwRouter_Param20      	 1000000	      1073 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Param20       	 1000000	      1041 ns/op	     640 B/op	       1 allocs/op

BenchmarkAero_ParamWrite          	13136160	        91.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParamWrite     	13807000	        87.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParamWrite         	  866906	      1367 ns/op	     360 B/op	       4 allocs/op
BenchmarkGin_ParamWrite           	 8124031	       149 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParamWrite    	  436358	      2580 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParamWrite   	  576049	      1839 ns/op	     976 B/op	       8 allocs/op
BenchmarkHttpRouter_ParamWrite    	 8232554	       140 ns/op	      32 B/op	       1 allocs/op
```

### GithubAPI Routes: 203

``` Shell
   Aero: 472856 Bytes
   ApiRouter: 93488 Bytes
   Beego: 150936 Bytes
   Gin: 58512 Bytes
   GorillaMux: 1322784 Bytes
   GowwwRouter: 80008 Bytes
   HttpRouter: 37096 Bytes

BenchmarkAero_GithubStatic        	22771987	        52.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubStatic   	39730426	        30.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubStatic       	  981472	      1285 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubStatic         	11793066	       104 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubStatic  	  216598	      5578 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GithubStatic 	15343174	        77.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GithubStatic  	24002008	        43.5 ns/op	       0 B/op	       0 allocs/op

BenchmarkAero_GithubParam         	10279070	       115 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubParam    	14197182	        81.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubParam        	  760636	      1438 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GithubParam          	 6255403	       191 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubParam   	  143298	      8273 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GithubParam  	 1589114	       762 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GithubParam   	 4567689	       265 ns/op	      96 B/op	       1 allocs/op

BenchmarkAero_GithubAll           	   52720	     22547 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GithubAll      	   68655	     17286 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GithubAll          	    4044	    302730 ns/op	   71457 B/op	     609 allocs/op
BenchmarkGin_GithubAll            	   27684	     43329 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GithubAll     	     294	   3991615 ns/op	  251654 B/op	    1994 allocs/op
BenchmarkGowwwRouter_GithubAll    	   10000	    157336 ns/op	   72144 B/op	     501 allocs/op
BenchmarkHttpRouter_GithubAll     	   23086	     53009 ns/op	   13792 B/op	     167 allocs/op
```

### GPlusAPI Routes : 13

```Shell
   Aero: 26840 Bytes
   ApiRouter: 32240 Bytes
   Beego: 10272 Bytes
   Gin: 4384 Bytes
   GorillaMux: 66208 Bytes
   GowwwRouter: 5744 Bytes
   HttpRouter: 2760 Bytes

BenchmarkAero_GPlusStatic         	29943988	        40.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusStatic    	53868162	        21.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusStatic        	  868545	      1224 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusStatic          	14917767	        77.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusStatic   	  634068	      1913 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_GPlusStatic  	38727555	        31.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GPlusStatic   	43723036	        26.8 ns/op	       0 B/op	       0 allocs/op

BenchmarkAero_GPlusParam          	17498322	        67.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusParam     	20866016	        58.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusParam         	  925665	      1356 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlusParam           	 9070120	       132 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusParam    	  352827	      3254 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlusParam   	 1751876	       686 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlusParam    	 6812570	       172 ns/op	      64 B/op	       1 allocs/op
BenchmarkAero_GPlus2Params        	12382214	       101 ns/op	       0 B/op	       0 allocs/op

BenchmarkApiRouter_GPlus2Params   	12620882	        93.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlus2Params       	  826052	      1454 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_GPlus2Params         	 6700472	       176 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlus2Params  	  198207	      6134 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_GPlus2Params 	 1663504	       730 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_GPlus2Params  	 5834750	       207 ns/op	      64 B/op	       1 allocs/op

BenchmarkAero_GPlusAll            	 1000000	      1008 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_GPlusAll       	 1336905	       900 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_GPlusAll           	   63999	     17971 ns/op	    4576 B/op	      39 allocs/op
BenchmarkGin_GPlusAll             	  672241	      1871 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_GPlusAll      	   22963	     51314 ns/op	   16112 B/op	     128 allocs/op
BenchmarkGowwwRouter_GPlusAll     	  139518	      8636 ns/op	    4752 B/op	      33 allocs/op
BenchmarkHttpRouter_GPlusAll      	  465562	      2384 ns/op	     640 B/op	      11 allocs/op
```

### ParseAPI Routes: 26

```Shell
   Aero: 29304 Bytes
   ApiRouter: 38928 Bytes
   Beego: 19280 Bytes
   Gin: 7776 Bytes
   GorillaMux: 105880 Bytes
   GowwwRouter: 9344 Bytes
   HttpRouter: 5024 Bytes

BenchmarkAero_ParseStatic         	25968115	        44.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseStatic    	48030700	        26.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseStatic        	  917887	      1268 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseStatic          	14337064	        83.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseStatic   	  560444	      2351 ns/op	     976 B/op	       9 allocs/op
BenchmarkGowwwRouter_ParseStatic  	36622987	        33.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_ParseStatic   	44521252	        28.2 ns/op	       0 B/op	       0 allocs/op

BenchmarkAero_ParseParam          	19545406	        61.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseParam     	20230845	        62.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseParam         	  892608	      1352 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_ParseParam           	12509785	        98.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseParam    	  409419	      2631 ns/op	    1280 B/op	      10 allocs/op
BenchmarkGowwwRouter_ParseParam   	 1781226	       682 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_ParseParam    	 7534678	       157 ns/op	      64 B/op	       1 allocs/op

BenchmarkAero_Parse2Params        	16480020	        71.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_Parse2Params   	17407262	        69.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_Parse2Params       	  758266	      1398 ns/op	     352 B/op	       3 allocs/op
BenchmarkGin_Parse2Params         	 9756090	       123 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_Parse2Params  	  424528	      3090 ns/op	    1296 B/op	      10 allocs/op
BenchmarkGowwwRouter_Parse2Params 	 1712110	       691 ns/op	     432 B/op	       3 allocs/op
BenchmarkHttpRouter_Parse2Params  	 6507544	       185 ns/op	      64 B/op	       1 allocs/op

BenchmarkAero_ParseAll            	  689119	      1761 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_ParseAll       	  942591	      1302 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_ParseAll           	   35602	     33737 ns/op	    9152 B/op	      78 allocs/op
BenchmarkGin_ParseAll             	  364644	      3346 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_ParseAll      	   10000	    101224 ns/op	   30288 B/op	     250 allocs/op
BenchmarkGowwwRouter_ParseAll     	   87921	     13179 ns/op	    6912 B/op	      48 allocs/op
BenchmarkHttpRouter_ParseAll      	  406630	      2940 ns/op	     640 B/op	      16 allocs/op
```

### Static Routes: 157

```Shell
   Aero: 34824 Bytes
   ApiRouter: 29040 Bytes
   Beego: 98456 Bytes
   Gin: 34936 Bytes
   GorillaMux: 585632 Bytes
   GowwwRouter: 24968 Bytes
   HttpRouter: 21680 Bytes

BenchmarkAero_StaticAll           	  108292	     10464 ns/op	       0 B/op	       0 allocs/op
BenchmarkApiRouter_StaticAll      	  203727	      5969 ns/op	       0 B/op	       0 allocs/op
BenchmarkBeego_StaticAll          	    5949	    225294 ns/op	   55265 B/op	     471 allocs/op
BenchmarkGin_StaticAll            	   42675	     28789 ns/op	       0 B/op	       0 allocs/op
BenchmarkGorillaMux_StaticAll     	    1035	   1068733 ns/op	  153236 B/op	    1413 allocs/op
BenchmarkGowwwRouter_StaticAll    	   59713	     20066 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_StaticAll     	   96944	     12553 ns/op	       0 B/op	       0 allocs/op
```
