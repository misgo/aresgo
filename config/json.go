/*
	配置文件解析类---json结构解析实现
	@author : hyperion
	@since  : 2017-01-17
	@version: 1.0.1
*/
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

//json配置对象，是对configer接口的实现
type JSONConfig struct {
}

// 根据文件地址返回一个配置文件接口器
func (js *JSONConfig) Parse(filename string) (Configer, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return js.ParseData(content)
}

// 根据json字符串返回配置文件器
func (js *JSONConfig) ParseData(data []byte) (Configer, error) {
	x := &JSONConfigContainer{
		data: make(map[string]interface{}),
	}
	err := json.Unmarshal(data, &x.data)
	if err != nil {
		var wrappingArray []interface{}
		err2 := json.Unmarshal(data, &wrappingArray)
		if err2 != nil {
			return nil, err
		}
		x.data["rootArray"] = wrappingArray
	}

	x.data = ExpandValueEnvForMap(x.data)

	return x, nil
}

//配置文件容器（获取值时可以通过"section.name"）
type JSONConfigContainer struct {
	data map[string]interface{}
	sync.RWMutex
}

//通过Key获取一个布尔值
func (c *JSONConfigContainer) Bool(key string) (bool, error) {
	val := c.getData(key)
	if val != nil {
		return ParseBool(val)
	}
	return false, fmt.Errorf("not exist key: %q", key)
}

//无错误时返回一个布尔值，否则返回一个默认值
func (c *JSONConfigContainer) DefaultBool(key string, defaultval bool) bool {
	if v, err := c.Bool(key); err == nil {
		return v
	}
	return defaultval
}

//通过Key返回一个Int型值
func (c *JSONConfigContainer) Int(key string) (int, error) {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(float64); ok {
			return int(v), nil
		}
		return 0, errors.New("not int value")
	}
	return 0, errors.New("not exist key:" + key)
}

//无错误时返回一个int，否则返回一个默认值
func (c *JSONConfigContainer) DefaultInt(key string, defaultval int) int {
	if v, err := c.Int(key); err == nil {
		return v
	}
	return defaultval
}

//通过Key返回一个int64型值
func (c *JSONConfigContainer) Int64(key string) (int64, error) {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(float64); ok {
			return int64(v), nil
		}
		return 0, errors.New("not int64 value")
	}
	return 0, errors.New("not exist key:" + key)
}

//无错误时返回一个int64，否则返回一个默认值
func (c *JSONConfigContainer) DefaultInt64(key string, defaultval int64) int64 {
	if v, err := c.Int64(key); err == nil {
		return v
	}
	return defaultval
}

//通过Key返回一个float型值
func (c *JSONConfigContainer) Float(key string) (float64, error) {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(float64); ok {
			return v, nil
		}
		return 0.0, errors.New("not float64 value")
	}
	return 0.0, errors.New("not exist key:" + key)
}

//无错误时返回一个float，否则返回一个默认值
func (c *JSONConfigContainer) DefaultFloat(key string, defaultval float64) float64 {
	if v, err := c.Float(key); err == nil {
		return v
	}
	return defaultval
}

//通过Key返回一个string型
func (c *JSONConfigContainer) String(key string) string {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(string); ok {
			return v
		}
	}
	return ""
}

//无错误时返回一个string，否则返回一个默认值
func (c *JSONConfigContainer) DefaultString(key string, defaultval string) string {
	// TODO FIXME should not use "" to replace non existence
	if v := c.String(key); v != "" {
		return v
	}
	return defaultval
}

//通过Key返回一个string数组
func (c *JSONConfigContainer) Strings(key string) []string {
	stringVal := c.String(key)
	if stringVal == "" {
		return nil
	}
	return strings.Split(c.String(key), ";")
}

//无错误时返回一个string数组，否则返回一个默认值
func (c *JSONConfigContainer) DefaultStrings(key string, defaultval []string) []string {
	if v := c.Strings(key); v != nil {
		return v
	}
	return defaultval
}

//通过片段字符串返回一个map（ini文件可用，json文件不可用）
func (c *JSONConfigContainer) GetSection(section string) (map[string]string, error) {
	if v, ok := c.data[section]; ok {
		return v.(map[string]string), nil
	}
	return nil, errors.New("nonexist section " + section)
}

//将配置信息数据保存到配置文件中
func (c *JSONConfigContainer) SaveConfigFile(filename string) (err error) {

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

//配置设置。
func (c *JSONConfigContainer) Set(key, val string) error {
	c.Lock()
	defer c.Unlock()
	c.data[key] = val
	return nil
}

//根据给定的Key自定义设置value
//支持":"定位。
/*如：json:
{
	"server1":
	{
	  "master":{"ip":"127.0.0.1","db":"go1"},
	  "slave":{"ip":"192.168.0.1","db":"go2"}
	}
}
获取时：cnf.GetVal("server1.master.ip")
*/
func (c *JSONConfigContainer) GetVal(key string) (v interface{}, err error) {
	val := c.getData(key)
	if val != nil {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("当前key[%s]未能获取到值！", key))
}

//根据值获取数据。路径分割符"."
func (c *JSONConfigContainer) getData(key string) interface{} {
	if len(key) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	sectionKeys := strings.Split(key, ".") //通过路径符进行分割
	if len(sectionKeys) >= 2 {
		curValue, ok := c.data[sectionKeys[0]]
		if !ok {
			return nil
		}
		for _, key := range sectionKeys[1:] {
			if v, ok := curValue.(map[string]interface{}); ok {
				if curValue, ok = v[key]; !ok {
					return nil
				}
			}
		}
		return curValue
	}
	if v, ok := c.data[key]; ok {
		return v
	}
	return nil
}

func (c *JSONConfigContainer) GetKeys() []string {
	var keys []string
	if len(c.data) > 0 {
		for k, _ := range c.data {
			keys = append(keys, k)
		}
	}
	return keys
}

//配置文件初始化
func init() {
	Register("json", &JSONConfig{})
}
