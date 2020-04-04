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
