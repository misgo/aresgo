/*
	框架类，封装了通用访问方法
	@author : hyperion
	@since  : 2018-02-14
	@version: 1.0.180214
*/
package frame

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"path"
	"strings"

	//	"mime"
	"mime/multipart"
	"net/http"

	"github.com/misgo/aresgo/router/fasthttp"
	"github.com/misgo/aresgo/text"
)

const (
	ContentTypeForm = "form"
	ContentTypeJSON = "json"
)

//结构体定义
type (
	App struct {
	}
	//符合表单multipart/form-data 复合表单字段，可以用来传输文件，也可传输普通字段
	MultipartField struct {
		IsFile    bool
		FieldName string
		FileName  string
		Value     []byte
	}
	//普通表单（application/x-www-0）字段
	FormField struct {
		FieldName string
		Value     []byte
	}
	//文件对象
	File struct {
		FieldName   string // 文件字段名
		FileName    string //文件名称
		ContentType string //文件类型，此类型指是mime类型
		MediaType   string //媒体类型。如：attentment,....
		Extension   string //文件后缀名
		Size        int64  //文件大小
		Value       []byte //文件内容，以byte数组方式返回
		URL         string //文件地址
	}
	//HTTP响应对象
	Response struct {
		Server          string                  //服务器名
		Status          int                     //返回状态码
		ContentType     string                  //http内容类型（content-type）
		ContentEncoding string                  //内容编码或压缩模式
		ContentLength   int                     //内容长度及容量
		MediaType       string                  //媒体类型，如果为媒体文件则获取媒体类型，如：attentment,...
		FileName        string                  //文件名，如果未文件类型则获取文件名
		Extension       string                  //文件后缀名
		Body            []byte                  //接收的Body体
		Header          fasthttp.ResponseHeader //fasthttp.responseheader响应头
	}
)

//全局变量
var (
	Debug        bool   //是否为调试模式
	CacheMode    string //缓存模式
	CacheTimeout int64  //缓存失效时间
)

//Get方式获取数据
func Get(url string, contentTypeKey string, para ...map[string][]byte) ([]byte, error) {
	var resp *Response
	var err error
	if len(para) > 0 {
		resp, err = Curl("GET", url, contentTypeKey, para[0])
	} else {
		resp, err = Curl("GET", url, contentTypeKey)
	}
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

//POST方式访问远端接口，传参为JSON
//@param url 地址
//@param json Json字符串
func PostJson(url string, data interface{}) ([]byte, error) {
	resp, err := Curl("POST", url, ContentTypeJSON, data)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

//POST方式访问远端地址，参数类型为键值对形式
func Post(url string, data map[string][]byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.New("请求参数不可以为空")
	}
	resp, err := Curl("POST", url, ContentTypeForm, data)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

//Post文件型数据到指定URL并接收返回值。此方法用来提交单个文件
//@param fieldName 文件对应的字段名称
//@param fielName 文件名称
//@param url 提交到远程的URL地址
//@return    （远程响应返回的byte字节组，错误信息）
//@since hyperion 2019-08-08
func PostFile(fieldName string, header *multipart.FileHeader, url string) ([]byte, error) {
	body, err := GetFileStream(header)
	if err != nil {
		return nil, err
	}
	fields := []MultipartField{
		{
			IsFile:    true,
			FieldName: fieldName,
			FileName:  header.Filename,
			Value:     body,
		},
	}
	return PostMultipartForm(fields, url)
}

//POST方式向远程提交multipart/form-data表单，可包含文件或其他字段
//@param fields 表单字段，可以是文件型字段，也可以是普通字段
//@param uri URL地址
//@return responseBody 远程响应返回的byte字节组
// @return  err 错误信息
//@since hyperion 2019-08-06
func PostMultipartForm(fields []MultipartField, url string) (respBody []byte, err error) {
	if len(fields) < 1 {
		return nil, errors.New("字段列表不可以为空")
	}
	boundary := Text.Guid() //生成字符分割符
	body := bytes.NewBuffer(nil)
	//写入消息头
	for _, field := range fields {
		body.WriteString("--" + boundary + "\r\n")
		if field.IsFile { //文件类型
			body.WriteString(fmt.Sprintf(`Content-Disposition: form-data; name="%s"; filename="%s"`, field.FieldName, field.FileName))
			body.WriteString("\r\n")
			body.WriteString("Content-Type: application/octet-stream")
		} else { //表单字段类型
			body.WriteString(fmt.Sprintf(`Content-Disposition: form-data; name="%s"; `, field.FieldName))
		}
		body.WriteString("\r\n\r\n")
		body.Write(field.Value)
		body.WriteString("\r\n\r\n")
	}
	body.WriteString("--" + boundary + "--\r\n")

	contentType := fmt.Sprintf(" multipart/form-data; boundary=%s", boundary)
	//发送请求
	resp, err := http.Post(url, contentType, body)
	if err != nil {
		Text.Log("post_data").Error(fmt.Sprintf("http Post[%s]:%s\r\n", url, err.Error()))
		return
	}
	defer resp.Body.Close()
	//解析远程服务器返回数据结构
	if resp.StatusCode != 200 {
		Text.Log("post_data").Error(fmt.Sprintf("http Post[%s] code:%s\r\n", url, resp.StatusCode))
		return nil, errors.New("数据发送失败")
	}
	respBody, err = ioutil.ReadAll(resp.Body)
	return
}

//获取上传文件字节流
//@para fileHeader 上传文件，前台可通过ctx.Fromfile方法获取
func GetFileStream(fileHeader *multipart.FileHeader) (body []byte, err error) {
	if fileHeader == nil {
		return nil, errors.New("未能获取文件数据")
	}
	file, openErr := fileHeader.Open()
	if openErr != nil {
		return nil, errors.New("未能打开文件")
	}
	fileBytes, readErr := ioutil.ReadAll(file)
	if readErr != nil {
		return nil, errors.New("未能读取文件")
	}
	return fileBytes, nil
}

// 抓取远程文件
func GetRemoteFile(action string, url string, isJson bool, params ...map[string]string) (fileInfo *File, err error) {
	if !strings.Contains(url, "http://") && !strings.Contains(url, "https://") {
		return nil, errors.New("请输入完整的URL，需包含http://或https://")
	}
	if action == "" {
		action = "GET"
	}
	var resp *Response
	var fields map[string]string
	if len(params) > 0 {
		fields = params[0]
	}
	//var contentTypeKey string = ContentTypeForm
	var contentTypeKey string = "txt"
	if isJson {
		contentTypeKey = ContentTypeJSON
	}
	//resp, err = PostForm(fields, url)
	if fields != nil {
		resp, err = Curl(action, url, contentTypeKey, fields)
	} else {
		resp, err = Curl(action, url, contentTypeKey)
	}

	if err != nil {
		return nil, err
	}
	fileInfo = &File{}
	fileInfo.URL = url
	fileInfo.ContentType = resp.ContentType
	fileInfo.Extension = resp.Extension
	fileInfo.FileName = resp.FileName
	fileInfo.MediaType = resp.MediaType
	fileInfo.Size = int64(resp.ContentLength)
	fileInfo.Value = resp.Body
	return fileInfo, nil
}

//访问远端接口，支持POST和GET
//@param action  操作参数，POST  or  GET
//@param url 远端地址URL
//@param contentTypeKey 请求内容类型(content-type)的键，如:form,json，可以使用ContentTypeForm,ContentTypeJSON来传输，其他类型默认为"text/plain"
func Curl(action string, url string, contentTypeKey string, para ...interface{}) (response *Response, err error) {
	if _, ok := Mime[contentTypeKey]; !ok {
		return nil, errors.New("获取content-type类型有误")
	}
	contentTypeKey = strings.Trim(contentTypeKey, "")
	action = strings.ToUpper(action)
	var contentType string = GetContentType("txt") //默认content-type为text/plain
	var reqBody []byte
	var params interface{}
	if len(para) > 0 {
		params = para[0]
	}
	if Debug {
		//fmt.Printf("action:%s;\r\nurl:%s;\r\nconentTypeKey:%s;\r\npara:%v;\r\n", action, url, contentTypeKey, para)
	}

	if contentTypeKey == ContentTypeForm { //普通form表单提交
		contentType = GetContentType(ContentTypeForm)
		if fields, ok := params.(map[string][]byte); ok {
			sb := Text.NewString("")
			for k, v := range fields {
				sb.AppendBytes([]byte("&"))
				sb.AppendBytes([]byte(k))
				sb.AppendBytes([]byte("="))
				sb.AppendBytes(v)
			}
			reqBody = sb.ToBytes()
		} else {
			return nil, errors.New("表单类型请求参数类型必须为：map[string][]byte")
		}
	} else { //JSON或其他方式提交，将数据转成JSON序列化进行提交
		if contentTypeKey == ContentTypeJSON { //JSON方式提交要更改content-type
			contentType = GetContentType(ContentTypeJSON)
		}
		if params != nil {
			reqBody, err = json.Marshal(params)
			if err != nil {
				return nil, errors.New("请求数据JSON解析出错")
			}
			reqBody = bytes.Replace(reqBody, []byte("\\u003c"), []byte("<"), -1)
			reqBody = bytes.Replace(reqBody, []byte("\\u003e"), []byte(">"), -1)
			reqBody = bytes.Replace(reqBody, []byte("\\u0026"), []byte("&"), -1)
		}

	}
	if Debug {
		fmt.Printf("request data:%v\r\n", string(reqBody))
	}
	//请求数据处理
	req := &fasthttp.Request{}
	req.Header.SetMethod(action)
	req.Header.SetContentType(fmt.Sprintf("%s;charset=utf-8", contentType, ""))
	req.Header.Set("Connection", "Keep-Alive")
	req.SetRequestURI(url)
	if reqBody != nil { //如果body体没有数据非要append，会导致某些应用返回body错误
		req.AppendBody(reqBody)
	}
	//返回数据处理
	resp := &fasthttp.Response{}
	client := &fasthttp.Client{}
	client.Name = "areshttp"
	err = client.Do(req, resp)
	defer resp.Reset()
	if resp.StatusCode() != 200 {
		return nil, errors.New(fmt.Sprintf("地址[%s]链接失败！", url))
	} else if err != nil {
		return nil, err
	} else {
		if Debug {
			fmt.Printf("response data:%v\r\n", string(resp.Body()))
		}
		response = &Response{}
		response.Status = resp.StatusCode()
		response.Body = resp.Body()
		//获取请求头
		response.Header = resp.Header
		response.ContentLength = resp.Header.ContentLength()
		response.Server = string(resp.Header.Peek("server"))
		response.ContentEncoding = string(resp.Header.Peek("content-encoding"))
		response.ContentType = string(resp.Header.Peek("content-type"))
		fileExt := path.Ext(url)
		if fileExt != "" {
			response.Extension = Text.SubStr(fileExt, ".")
		} else {
			response.Extension = GetFileExt(response.ContentType)
		}
		//获取内容部署节点
		disposition := resp.Header.Peek("Content-disposition")

		if len(disposition) > 0 {
			mediaType, params, _ := mime.ParseMediaType(string(disposition))
			//获取媒体类型
			if mediaType != "" {
				response.MediaType = mediaType
			}
			//获取文件名称及扩展名
			if _, ok := params["filename"]; ok {
				response.FileName = params["filename"]
				response.Extension = Text.SubStr(path.Ext(response.FileName), ".") //文件类型以filename为准
			}
		}
		return response, nil
	}

}
