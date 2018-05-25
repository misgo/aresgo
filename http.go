/*
	框架路由包,封装了监听路由方法，及路由后回调函数方法
	@author : hyperion
	@since  : 2016-12-05
	@version: 1.0.1
*/
package aresgo

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/misgo/aresgo/router/fasthttp"
	"github.com/misgo/aresgo/text"
)

const (
	ActionGet     = "GET"
	ActionPost    = "POST"
	ActionConn    = "CONNECT"
	ActionPut     = "PUT"
	ActionDelete  = "DELETE"
	ActionHead    = "HEAD"
	ActionPatch   = "PATCH"
	ActionOptions = "OPTIONS"
	ActionTrace   = "TRACE"
)

var (
	defaultContentType = []byte("text/plain; charset=utf-8")
	questionMark       = []byte("?")
)

type (
	//http上下文
	Context struct {
		*fasthttp.RequestCtx
		AllowCrossDomain bool
		CrossOrigin      string
	}

	HandlerFunc func(*Context)      //路由分发函数
	HttpModule  func(*Context) bool //http拦截器

	//路由器
	Router struct {
		trees  map[string]*node //路由表
		rvList map[string]reflect.Value
		//		ActionList       map[string]*Controller
		NotFound            HandlerFunc //未找到路由函数(404错误页执行方法)
		MethodNotAllowed    HandlerFunc //不允许使用指定的方法。比如：未注册路由POST访问地址/user/login，那么通过POST请求时会报此方法的回调函数
		HttpModuleIntercept HandlerFunc //httpmodule，http拦截器拒绝访问，用来跳转拦截是默认执行操作

		RedirectTrailingSlash  bool //是否支持url末尾反斜杠跳转
		RedirectFixedPath      bool
		HandleMethodNotAllowed bool
		HandleOptions          bool

		PanicHandler func(*Context, interface{})

		IsUseAutoRoute  bool   //是否启用自动路由（不确定路由表，根据struct和action确定路由回调函数）
		AutoRoutePrefix string //前缀保证路由的地址不冲突

	}
	//服务器处理
	Server struct {
		FastServer    *fasthttp.Server
		Listener      net.Listener
		RouterHandler fasthttp.RequestHandler

		contextPool sync.Pool
	}
)

//----初始化路由------
func Routing() *Router {
	r := &Router{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOptions:          true,
		IsUseAutoRoute:         false,
	}
	return r
}

/**
 *********************服务器监听及处理****************start***********************************
 */

//监听服务器
func (router *Router) Listen(addr string) {
	s := &Server{}
	//将上下文对象初始化到缓冲池中
	s.contextPool.New = func() interface{} {
		return &Context{}
	}

	if s.RouterHandler == nil {
		defaultHandler := func(reqCtx *fasthttp.RequestCtx) {
			ctx := s.AcquireCtx(reqCtx)
			router.Handler(ctx)
			s.ReleaseCtx(ctx) //释放上下文
		}
		s.RouterHandler = defaultHandler
	}

	s.FastServer = &fasthttp.Server{
		Handler: s.RouterHandler,
		Name:    "aresgo server",
	}
	//控制台输出服务器信息
	timeNow := time.Now()
	serverStartTime := timeNow.Format("2006-01-02 15:04:05") //格式化时要注意，必须是个这个时间点，据说是Go的诞生日
	fmt.Printf("%s\r\n", Banner)
	fmt.Printf("-------- Server:%s，start time：%s --------\r\n", s.FastServer.Name, serverStartTime)

	//监听服务器器
	log.Fatal(fasthttp.ListenAndServe(addr, s.RouterHandler))
	//	ln, err := net.Listen("tcp4", addr)
	//	if err != nil {
	//		//		return err
	//	}
	//	f.FastServer.Serve(ln)
}

//在缓冲池中获取上下文（Context）
func (s *Server) AcquireCtx(reqCtx *fasthttp.RequestCtx) *Context {
	ctx := s.contextPool.Get().(*Context)
	ctx.RequestCtx = reqCtx
	return ctx
}

//释放上下文（Context）资源
func (s *Server) ReleaseCtx(ctx *Context) {
	s.contextPool.Put(ctx) //将上下文重新放回缓冲池
}

/**
 *********************服务器监听及处理****************end***********************************
 */

/**
 ************************路由句柄处理******start**************************************
 */
//自动路由函数
//Action中的方法名定义首字母大写其他字母小写
func (r *Router) autoroute(ctx *Context) {
	action := ctx.UserValue("action").(string)
	action = Text.FirstCharToUpper(action) //struct的方法名首字母大写其他字母小写
	path := Text.SubStrBytes(ctx.Path(), []byte(""), []byte("/"))
	pathKey := strings.ToUpper(string(path))

	if sv, ok := r.rvList[pathKey]; ok { //有路由对象
		_, bol := sv.Type().MethodByName(action)
		if !bol {
			if r.NotFound != nil { //404错误执行NotFound回调
				r.NotFound(ctx)
			}

		} else {
			args := make([]reflect.Value, 1)
			args[0] = reflect.ValueOf(ctx)
			sv.MethodByName(action).Call(args)
		}

	} else {
		if r.NotFound != nil { //404错误执行NotFound回调
			r.NotFound(ctx)
		}
	}
}

//路由处理句柄---注册自动路由
//示例：router.Register("/Path1/", &struct{}, "GET", "POST")
func (r *Router) Register(path string, s interface{}, httpmod HttpModule, actions ...string) {
	//路径操作
	pathTrim := strings.TrimRight(path, "/")
	pathKey := strings.ToUpper(pathTrim)       //自动路由表中对应的struct反射模型的Key
	sv := reflect.ValueOf(s)                   //struct反射模型
	path = fmt.Sprintf("%s/:action", pathTrim) //构造自定义路由控制路径
	//初始化自动路由表
	if r.rvList == nil {
		r.rvList = make(map[string]reflect.Value)
	}
	r.rvList[pathKey] = sv

	//根据访问模式将回调添加到不同的路由表
	for i := 0; i < len(actions); i++ {
		action := actions[i]
		if action == ActionGet { //Get请求
			r.Get(path, r.autoroute, httpmod)
		} else if action == ActionPost { //Post请求
			r.Post(path, r.autoroute, httpmod)
		}
	}

}

// 路由处理句柄---Get方式
func (r *Router) Get(path string, handle HandlerFunc, httpmod ...HttpModule) {
	if len(httpmod) >= 1 {
		r.Handle(ActionGet, path, handle, httpmod[0])
	} else {
		r.Handle(ActionGet, path, handle, nil)
	}

}

// 路由处理句柄---Head方式
func (r *Router) Head(path string, handle HandlerFunc) {
	r.Handle(ActionHead, path, handle, nil)
}

// 路由处理句柄---Options方式
func (r *Router) Options(path string, handle HandlerFunc) {
	r.Handle(ActionOptions, path, handle, nil)
}

// 路由处理句柄---Post方式
func (r *Router) Post(path string, handle HandlerFunc, httpmod ...HttpModule) {
	if len(httpmod) >= 1 {
		r.Handle(ActionPost, path, handle, httpmod[0])
	} else {
		r.Handle(ActionPost, path, handle, nil)
	}

}

// 路由处理句柄---Put方式
func (r *Router) Put(path string, handle HandlerFunc) {
	r.Handle(ActionPut, path, handle, nil)
}

// 路由处理句柄---Patch方式
func (r *Router) Patch(path string, handle HandlerFunc) {
	r.Handle(ActionPatch, path, handle, nil)
}

// 路由处理句柄---Delete方式
func (r *Router) Delete(path string, handle HandlerFunc) {
	r.Handle(ActionDelete, path, handle, nil)
}

//路由统一处理句柄，根据配置的路径将路径与回调函数添加到路由注册表中
func (r *Router) Handle(method string, path string, handle HandlerFunc, httpmod HttpModule) {
	if path[0] != '/' {
		panic("路由路径[" + path + "]必须以 '/' 开头")
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}

	root.addRoute(path, handle, httpmod)

}

/**
 ************************路由句柄处理******end****************************************
 */

func (r *Router) recv(ctx *Context) {
	if rcv := recover(); rcv != nil {
		r.PanicHandler(ctx, rcv)
	}
}

//手动查找方法+路径组合
//如果路径能找到，会返回方法和路径参数值
func (r *Router) Lookup(method, path string, ctx *Context) (HandlerFunc, HttpModule, bool) {
	if root := r.trees[method]; root != nil {
		return root.getValue(path, ctx)
	}
	return nil, nil, false
}

func (r *Router) allowed(path, reqMethod string) (allow string) {
	if path == "*" || path == "/*" { // server-wide
		for method := range r.trees {
			if method == ActionOptions {
				continue
			}

			// add request method to list of allowed methods
			if len(allow) == 0 {
				allow = method
			} else {
				allow += ", " + method
			}
		}
	} else { // specific path
		for method := range r.trees {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == ActionOptions {
				continue
			}

			handle, _, _ := r.trees[method].getValue(path, nil)
			if handle != nil {
				//添加方法到许可的方法列表里
				if len(allow) == 0 {
					allow = method
				} else {
					allow += ", " + method
				}
			}
		}
	}
	if len(allow) > 0 {
		allow += ", " + ActionOptions
	}
	return
}

//Handler方法实现了fasthttp.ListenAndServe的路由接口
//此方法用来处理回调函数
func (r *Router) Handler(ctx *Context) {
	if r.PanicHandler != nil {
		defer r.recv(ctx)
	}

	path := string(ctx.Path())     //访问路径
	method := string(ctx.Method()) //访问方法：GET,POST,...

	//autoPath:
	//响应头统一处理
	ctx.Response.Header.Set("Server", "areshttp")

	//记录访问信息
	ServerClientInfo(ctx)
	//回调处理

	//回调函数执行
	if root := r.trees[method]; root != nil {
		if f, h, tsr := root.getValue(path, ctx); f != nil {
			var b bool = true
			//http拦截器
			if h != nil {
				b = h(ctx)
			}
			if b {
				f(ctx) //如果回调函数存在，则将RequestCtx传入并执行
			} else {
				if r.HttpModuleIntercept != nil {
					r.HttpModuleIntercept(ctx) //拒绝访问默认方法
				}
			}

			return
		} else if method != ActionConn && path != "/" {
			code := 301 // 永久重定向
			if method != ActionGet {
				// 同一个方法临时性重定向
				// 1.3版本以下不支持308错误
				code = 307
			}

			if tsr && r.RedirectTrailingSlash {
				var uri string
				if len(path) > 1 && path[len(path)-1] == '/' {
					uri = path[:len(path)-1]
				} else {
					uri = path + "/"
				}
				ctx.Redirect(uri, code)
				return
			}

			// Try to fix the request path
			if r.RedirectFixedPath {
				fixedPath, found := root.findCaseInsensitivePath(
					CleanPath(path),
					r.RedirectTrailingSlash,
				)

				if found {
					queryBuf := ctx.URI().QueryString()
					if len(queryBuf) > 0 {
						fixedPath = append(fixedPath, questionMark...)
						fixedPath = append(fixedPath, queryBuf...)
					}
					uri := string(fixedPath)
					ctx.Redirect(uri, code)
					return
				}
			}
		}
	}
	if method == ActionOptions {
		// Handle OPTIONS requests
		if r.HandleOptions {
			if allow := r.allowed(path, method); len(allow) > 0 {
				ctx.Response.Header.Set("Allow", allow)
				return
			}
		}
	} else {
		// Handle 405
		if r.HandleMethodNotAllowed {
			if allow := r.allowed(path, method); len(allow) > 0 {
				ctx.Response.Header.Set("Allow", allow)
				if r.MethodNotAllowed != nil {
					r.MethodNotAllowed(ctx)
				} else {
					ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
					ctx.SetContentTypeBytes(defaultContentType)
					ctx.SetBodyString(fasthttp.StatusMessage(fasthttp.StatusMethodNotAllowed))
				}
				return
			}
		}
	}

	// Handle 404
	if r.NotFound != nil { //404错误执行NotFound回调
		r.NotFound(ctx)
	} else { //没有设置回调输出默认信息
		ctx.Error(fasthttp.StatusMessage(fasthttp.StatusNotFound),
			fasthttp.StatusNotFound)
	}
}

//ServerFiles方式提供一种方式用来访问静态资源，如包含页面的管理系统中涉及的CSS、Javascritp、图片等静态资源
// ServeFiles serves files from the given file system root.
//路径设计必须用这种形式：/XXX/*filepath(必须以/*filepath为结尾标识)，还必须有文件目录的绝对地址路径
//如静态文件路径存放在目录/var/www/static中，访问是想通过这种形式访问：http://www.XXX.com/static/XXX.js
//可以这样设置：r.ServerFiles("/static/*filepath","/var/www/static")
//通过这种方式可以创建一个纯静态的文件服务器，或者搭建一个包含模板静态资源的应用
func (r *Router) ServeFiles(path string, rootPath string) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("路径必须以/*filepath结尾 '" + path + "'")
	}
	prefix := path[:len(path)-10]

	fileHandler := fasthttp.FSHandler(rootPath, strings.Count(prefix, "/"))

	r.Get(path, func(ctx *Context) {
		fileHandler(ctx.RequestCtx)
	})
}

//----上下文处理方法---------start------
//输出Json数据
func (ctx *Context) ToJson(datas interface{}, msg ...string) {
	//设置response头
	if ctx.AllowCrossDomain && ctx.CrossOrigin != "" {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", ctx.CrossOrigin)
	} else {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	}
	ctx.Response.Header.Add("Accept-Encoding", "gzip")
	ctx.Response.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Response.Header.Add("Time", fmt.Sprintf("%d", time.Now().Unix()))
	ctx.SetContentType("application/json; charset=utf-8")

	//处理Json数据
	var res map[string]interface{} = make(map[string]interface{})
	var code int16 = 200
	var message string = ""
	msglen := len(msg)

	//处理返回的状态码及提示信息
	if msglen > 0 {
		codePara, err := strconv.ParseInt(msg[0], 10, 16) //将返回的状态码转为int16型
		if err != nil {
			code = 500
			message = fmt.Sprintf("json状态码节点[code=%s]有误，必须为10000以内的整数！", msg[0])
		} else {
			code = int16(codePara)
			if msglen > 1 {
				message = msg[1]
			}
		}
	}
	res["code"] = code
	res["message"] = message
	res["data"] = datas

	ret, err := json.Marshal(res)
	if err != nil {
		res["code"] = 500
		res["message"] = "生成的数据转换json时出错！"
		res["data"] = ""
		ret, _ = json.Marshal(res)
	}

	//编码及输出
	encoding := string(ctx.Request.Header.Peek("Content-Encoding"))
	if encoding == "gzip" { //gzip方式输出数据
		_, err := fasthttp.WriteGzip(ctx.Response.BodyWriter(), ret)
		if err != nil {
			res["code"] = 500
			res["message"] = "gzip压缩出错！"
			res["data"] = ""
			ctx.Response.SetBodyString(string(ret))
		}
	} else {
		ctx.Response.SetBodyString(string(ret))
	}
}

//输出Html数据
func (ctx *Context) ToHtml(datas interface{}, path ...string) {
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.Response.Header.Add("Time", fmt.Sprintf("%d", time.Now().Unix()))
	if len(path) > 0 { //模板方式输出
		t, err := template.ParseFiles(path[0])
		if err != nil {
			fmt.Println("[ERROR] ", err.Error())
			return
		}
		t = template.Must(t, err)
		err = t.Execute(ctx, datas)
		if err != nil {
			fmt.Println("[ERROR] ", err.Error())
			return
		}
	} else { //字符串方式输出
		fmt.Fprint(ctx, datas)
	}
}

//----上下文处理方法----------end------

//-----Cookie & Session -----start------
//设置Cookie
func (ctx *Context) SetCookie(key string, val string) {
	c := &fasthttp.Cookie{}
	c.SetKey(key)
	c.SetValue(val)
	ctx.Response.Header.SetCookie(c)

}

//获取Cookie
func (ctx *Context) GetCookie(key string) []byte {
	return ctx.Request.Header.Cookie(key)
}

//-----Cookie & Session ------end-------

//------------------path--------------------

//清除不规范路径
//清除规范：
/**
 ****清除不规范路径****
 *------清除规范----
 *1.多个斜杠替换成单斜杠
 *2.清除.路径（在当前目录下）
 *3.清除父路径的..,根路径中的/..用/替换
 */
func CleanPath(p string) string {
	// Turn empty string into "/"
	if p == "" {
		return "/"
	}

	n := len(p)
	var buf []byte

	// Invariants:
	//      reading from path; r is index of next byte to process.
	//      writing to buf; w is index of next byte to write.

	// path must start with '/'
	r := 1
	w := 1

	if p[0] != '/' {
		r = 0
		buf = make([]byte, n+1)
		buf[0] = '/'
	}

	trailing := n > 2 && p[n-1] == '/'

	// A bit more clunky without a 'lazybuf' like the path package, but the loop
	// gets completely inlined (bufApp). So in contrast to the path package this
	// loop has no expensive function calls (except 1x make)

	for r < n {
		switch {
		case p[r] == '/':
			// empty path element, trailing slash is added after the end
			r++

		case p[r] == '.' && r+1 == n:
			trailing = true
			r++

		case p[r] == '.' && p[r+1] == '/':
			// . element
			r++

		case p[r] == '.' && p[r+1] == '.' && (r+2 == n || p[r+2] == '/'):
			// .. element: remove to last /
			r += 2

			if w > 1 {
				// can backtrack
				w--

				if buf == nil {
					for w > 1 && p[w] != '/' {
						w--
					}
				} else {
					for w > 1 && buf[w] != '/' {
						w--
					}
				}
			}

		default:
			// real path element.
			// add slash if needed
			if w > 1 {
				bufApp(&buf, p, w, '/')
				w++
			}

			// copy element
			for r < n && p[r] != '/' {
				bufApp(&buf, p, w, p[r])
				w++
				r++
			}
		}
	}

	// re-append trailing slash
	if trailing && w > 1 {
		bufApp(&buf, p, w, '/')
		w++
	}

	if buf == nil {
		return p[:w]
	}
	return string(buf[:w])
}

//创建一个缓冲区
func bufApp(buf *[]byte, s string, w int, c byte) {
	if *buf == nil {
		if s[w] == c {
			return
		}

		*buf = make([]byte, len(s))
		copy(*buf, s[:w])
	}
	(*buf)[w] = c
}

//获取服务器及客户端相关信息
func ServerClientInfo(ctx *Context) {
	reqTime := time.Now().Format("2006-01-02 15:04:05")
	postParams := ctx.PostArgs().String()
	info := fmt.Sprintf("[ID:%d][%s] %s (%s %d) %s  agent:%s;query:%s;post:%s; ",
		ctx.ConnID(), reqTime, ctx.RemoteIP(), ctx.Method(), ctx.Response.StatusCode(), ctx.Path(), ctx.UserAgent(), ctx.QueryArgs().QueryString(), postParams)
	fmt.Println(info)
}
