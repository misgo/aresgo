/*
	配置文件解析类
	支持ini,json格式的文件，通过制定的文件路径将文件中的数据解析成相应的map结构
	亦可以通过访问符"."来直接获取或设置值
	@author : hyperion
	@since  : 2017-01-16
	@version: 1.0
*/
//Examples.
//
//  cnf, err := config.NewConfig("ini", "config.conf")
//
//  cnf APIS:
//
//  cnf.Set(key, val string) error
//  cnf.String(key string) string
//  cnf.Strings(key string) []string
//  cnf.Int(key string) (int, error)
//  cnf.Int64(key string) (int64, error)
//  cnf.Bool(key string) (bool, error)
//  cnf.Float(key string) (float64, error)
//  cnf.DefaultString(key string, defaultVal string) string
//  cnf.DefaultStrings(key string, defaultVal []string) []string
//  cnf.DefaultInt(key string, defaultVal int) int
//  cnf.DefaultInt64(key string, defaultVal int64) int64
//  cnf.DefaultBool(key string, defaultVal bool) bool
//  cnf.DefaultFloat(key string, defaultVal float64) float64
//  cnf.GetSection(section string) (map[string]string, error)
//  cnf.SaveConfigFile(filename string) error
package config

import (
	"fmt"
	"os"
	"reflect"
	"time"
)

//配置接口
// 定义了一些配置文件设置和获取的方法
type Configer interface {
	Set(key, val string) error   //support section::key type in given key when using ini type.
	String(key string) string    //support section::key type in key string when using ini and json type; Int,Int64,Bool,Float,DIY are same.
	Strings(key string) []string //get string slice
	Int(key string) (int, error)
	Int64(key string) (int64, error)
	Bool(key string) (bool, error)
	Float(key string) (float64, error)
	DefaultString(key string, defaultVal string) string      // support section::key type in key string when using ini and json type; Int,Int64,Bool,Float,DIY are same.
	DefaultStrings(key string, defaultVal []string) []string //get string slice
	DefaultInt(key string, defaultVal int) int
	DefaultInt64(key string, defaultVal int64) int64
	DefaultBool(key string, defaultVal bool) bool
	DefaultFloat(key string, defaultVal float64) float64
	GetVal(key string) (interface{}, error)
	GetSection(section string) (map[string]string, error)
	GetKeys() []string
	SaveConfigFile(filename string) error
}

// 定义了从配置文件中适配数据的接口
type Config interface {
	Parse(key string) (Configer, error)
	ParseData(data []byte) (Configer, error)
}

var adapters = make(map[string]Config) //配置文件适配器

//注册配置文件适配器，同一个名称如果注册两次会报出panic错误
//@param name 配置类型名称（）
func Register(name string, adapter Config) {
	if adapter == nil {
		panic("config: Register adapter is nil")
	}
	if _, ok := adapters[name]; ok {
		panic("config: Register called twice for adapter " + name)
	}
	adapters[name] = adapter
}

//根据适配器名称和文件路径获取配置文件数据
//@param adapterName 适配器名称：ini/json
//@param filename 文件路径
func NewConfig(adapterName, filename string) (Configer, error) {
	adapter, ok := adapters[adapterName]
	if !ok {
		return nil, fmt.Errorf("config: unknown adaptername %q (forgotten import?)", adapterName)
	}
	return adapter.Parse(filename)
}

//根据是适配器名称及byte数据获取配置文件数据
//@param adapterName 适配器名称：ini/json
//@param data 配置内容字节数组
func NewConfigData(adapterName string, data []byte) (Configer, error) {
	adapter, ok := adapters[adapterName]
	if !ok {
		return nil, fmt.Errorf("config: unknown adaptername %q (forgotten import?)", adapterName)
	}
	return adapter.ParseData(data)
}

//将所有的字符串值转换为环境变量
func ExpandValueEnvForMap(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		switch value := v.(type) {
		case string:
			m[k] = ExpandValueEnv(value)
		case map[string]interface{}:
			m[k] = ExpandValueEnvForMap(value)
		case map[string]string:
			for k2, v2 := range value {
				value[k2] = ExpandValueEnv(v2)
			}
			m[k] = value
		}
	}
	return m
}

/*
  返回转换为环境变量的值
  环境变量的值：开始字符："${"---结束字符"}",如果环境变量的值为空或者不存在则返回默认值
  允许的格式："${env}" , "${env||}}" , "${env||defaultValue}" , "defaultvalue"
  如：
	v1 := config.ExpandValueEnv("${GOPATH}")			// 返回全局变量.
	v2 := config.ExpandValueEnv("${GOAsta||/usr/local/go}")	// 返回默认值 "/usr/local/go/".
	v3 := config.ExpandValueEnv("value")				// 返回值"value".
*/
func ExpandValueEnv(value string) (realValue string) {
	realValue = value
	vLen := len(value)
	if vLen < 3 {
		return
	}
	// 如果不符合开始字符： "${" ，结束字符："}", 直接返回.
	if value[0] != '$' || value[1] != '{' || value[vLen-1] != '}' {
		return
	}

	key := ""
	defalutV := ""
	for i := 2; i < vLen; i++ {
		if value[i] == '|' && (i+1 < vLen && value[i+1] == '|') {
			key = value[2:i]
			defalutV = value[i+2 : vLen-1]
			break
		} else if value[i] == '}' {
			key = value[2:i]
			break
		}
	}

	realValue = os.Getenv(key)
	if realValue == "" {
		realValue = defalutV
	}

	return
}

//将值转换为布尔型
//允许的类型为：
//  逻辑是：1, 1.0, t, T, TRUE, true, True, YES, yes, Yes,Y, y, ON, on, On,
//  逻辑非：0, 0.0, f, F, FALSE, false, False, NO, no, No, N,n, OFF, off, Off.
func ParseBool(val interface{}) (value bool, err error) {
	if val != nil {
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			switch v {
			case "1", "t", "T", "true", "TRUE", "True", "YES", "yes", "Yes", "Y", "y", "ON", "on", "On":
				return true, nil
			case "0", "f", "F", "false", "FALSE", "False", "NO", "no", "No", "N", "n", "OFF", "off", "Off":
				return false, nil
			}
		case int8, int32, int64:
			strV := fmt.Sprintf("%s", v)
			if strV == "1" {
				return true, nil
			} else if strV == "0" {
				return false, nil
			}
		case float64:
			if v == 1 {
				return true, nil
			} else if v == 0 {
				return false, nil
			}
		}
		return false, fmt.Errorf("转换 %q: 语法错误", val)
	}
	return false, fmt.Errorf("转换 <nil>: 语法错误")
}

// 将任意类型转换成字符串
func ToString(x interface{}) string {
	switch y := x.(type) {
	case time.Time:
		return y.Format("A Monday")
	case string:
		return y
	case fmt.Stringer:
		return y.String()
	case error:
		return y.Error()

	}
	if v := reflect.ValueOf(x); v.Kind() == reflect.String {
		return v.String()
	}
	return fmt.Sprint(x)
}
