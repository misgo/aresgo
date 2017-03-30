Aresgo
---------------
aresgo是一个简单快速开发go应用的高性能框架，你可以用她来开发一些Api、Web及其他的一些服务应用，她是一个RESTful的框架。她包含快速的Http实现、Url路由与转发、Redis的实现、Mysql的CURD实现、JSON和INI配置文件的读写，以及其他一些方法的使用。后续会继续将一些常用应用添加到框架中。


产品特点（Features）
-----------------

* 实现思路借鉴iris-go,beego等框架
* http实现封装了fasthttp，fasthttp的方法和实现可以直接使用，如果使用fasthttp请引入包：github.com/aresgo/router/fasthttp
* mysql的实现封装了go-sql-driver，并作了CURD的扩展，同时考虑mysql的主从结构，可以通过配置文件进行主从配置，读写分离，从而使用更灵活更方便
* redis的实现封装garyburd/redigo，可以通过配置文件进行主从配置
* 配置文件管理（Json和ini）采用beego的框架方法

安装（Installation）
--------------------
使用“go get”命令：

>$ go get github.com/misgo/aresgo

用法（Usage）
-------------------
使用aresgo框架，你只需要在源文件都加上：

>import "github.com/aresgo"

或者如果使用框架中的某个包的方法，可以这样使用：

>import "github.com/aresgo/text"

http实现
---------------
```go

import "github.com/aresgo"

func main(){

  //初始化路由
  router := aresgo.Routing()

  //定义404错误页
  router.Get("/404.html", NotFound)
  router.NotFound = NotFound  //只要访问不存在的方法或地址就会跳转这个页面

  //输出方法
  router.Get("/hello/:name", Hello)   //Get方法请求，Post请求时会报错
  router.Post("/hello/:name", Hello)  //Post方法请求，Get请求时会报错

  //注册对象，注册后对象(struct)的所有公共方法可以被调用
  router.Register("/passport/", &action.UserAction{}, aresgo.ActionGet) 
  
  //POST or GET or ...请求被拒绝时执行的方法，取决于路由方法的设置
  router.MethodNotAllowed = DisAllowedMethod  
  
  //监听IP与端口，阻塞式服务
  router.Listen(“127.0.0.1:9000”)

}
//404错误页
func NotFound(ctx *aresgo.Context) {
	fmt.Fprint(ctx, "页面不存在!\n")
}
// 欢迎页
func Hello(ctx *aresgo.Context) {
	fmt.Fprintf(ctx, "hello, 欢迎【%s】光临!\n", ctx.UserValue("name"))
}

```

* 使用Registerr方法，注册的struct的公共方法可以被调用，方法名称需要首字母大写其他小写
* 路由参数支持“：”和“*”，或者常量