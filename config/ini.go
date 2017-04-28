/*
	配置文件解析类---ini结构解析实现
	@author : hyperion
	@since  : 2017-01-17
	@version: 1.0
*/
package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	defaultSection = "default"   // default section means if some ini items not in a section, make them in default section,
	bNumComment    = []byte{'#'} // number signal
	bSemComment    = []byte{';'} // semicolon signal
	bEmpty         = []byte{}
	bEqual         = []byte{'='} // equal signal
	bDQuote        = []byte{'"'} // quote signal
	sectionStart   = []byte{'['} // section start signal
	sectionEnd     = []byte{']'} // section end signal
	lineBreak      = "\n"
)

//ini配置对象，是对configer接口的实现
type IniConfig struct {
}

// 根据文件地址返回一个配置文件接口器.
func (ini *IniConfig) Parse(name string) (Configer, error) {
	return ini.parseFile(name)
}

//转换文件
func (ini *IniConfig) parseFile(name string) (*IniConfigContainer, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	cfg := &IniConfigContainer{
		file.Name(),
		make(map[string]map[string]string),
		make(map[string]string),
		make(map[string]string),
		sync.RWMutex{},
	}
	cfg.Lock()
	defer cfg.Unlock()
	defer file.Close()

	var comment bytes.Buffer
	buf := bufio.NewReader(file)
	// check the BOM
	head, err := buf.Peek(3)
	if err == nil && head[0] == 239 && head[1] == 187 && head[2] == 191 {
		for i := 1; i <= 3; i++ {
			buf.ReadByte()
		}
	}
	section := defaultSection
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		//It might be a good idea to throw a error on all unknonw errors?
		if _, ok := err.(*os.PathError); ok {
			return nil, err
		}
		line = bytes.TrimSpace(line)
		if bytes.Equal(line, bEmpty) {
			continue
		}
		var bComment []byte
		switch {
		case bytes.HasPrefix(line, bNumComment):
			bComment = bNumComment
		case bytes.HasPrefix(line, bSemComment):
			bComment = bSemComment
		}
		if bComment != nil {
			line = bytes.TrimLeft(line, string(bComment))
			// Need append to a new line if multi-line comments.
			if comment.Len() > 0 {
				comment.WriteByte('\n')
			}
			comment.Write(line)
			continue
		}

		if bytes.HasPrefix(line, sectionStart) && bytes.HasSuffix(line, sectionEnd) {
			section = strings.ToLower(string(line[1 : len(line)-1])) // section name case insensitive
			if comment.Len() > 0 {
				cfg.sectionComment[section] = comment.String()
				comment.Reset()
			}
			if _, ok := cfg.data[section]; !ok {
				cfg.data[section] = make(map[string]string)
			}
			continue
		}

		if _, ok := cfg.data[section]; !ok {
			cfg.data[section] = make(map[string]string)
		}
		keyValue := bytes.SplitN(line, bEqual, 2)

		key := string(bytes.TrimSpace(keyValue[0])) // key name case insensitive
		key = strings.ToLower(key)

		// handle include "other.conf"
		if len(keyValue) == 1 && strings.HasPrefix(key, "include") {
			includefiles := strings.Fields(key)
			if includefiles[0] == "include" && len(includefiles) == 2 {
				otherfile := strings.Trim(includefiles[1], "\"")
				if !filepath.IsAbs(otherfile) {
					otherfile = filepath.Join(filepath.Dir(name), otherfile)
				}
				i, err := ini.parseFile(otherfile)
				if err != nil {
					return nil, err
				}
				for sec, dt := range i.data {
					if _, ok := cfg.data[sec]; !ok {
						cfg.data[sec] = make(map[string]string)
					}
					for k, v := range dt {
						cfg.data[sec][k] = v
					}
				}
				for sec, comm := range i.sectionComment {
					cfg.sectionComment[sec] = comm
				}
				for k, comm := range i.keyComment {
					cfg.keyComment[k] = comm
				}
				continue
			}
		}

		if len(keyValue) != 2 {
			return nil, errors.New("read the content error: \"" + string(line) + "\", should key = val")
		}
		val := bytes.TrimSpace(keyValue[1])
		if bytes.HasPrefix(val, bDQuote) {
			val = bytes.Trim(val, `"`)
		}

		cfg.data[section][key] = ExpandValueEnv(string(val))
		if comment.Len() > 0 {
			cfg.keyComment[section+"."+key] = comment.String()
			comment.Reset()
		}

	}
	return cfg, nil
}

// 解析ini结构的数据
func (ini *IniConfig) ParseData(data []byte) (Configer, error) {
	// Save memory data to temporary file
	tmpName := path.Join(os.TempDir(), "github.com/misgo/aresgo", fmt.Sprintf("%d", time.Now().Nanosecond()))
	os.MkdirAll(path.Dir(tmpName), os.ModePerm)
	if err := ioutil.WriteFile(tmpName, data, 0655); err != nil {
		return nil, err
	}
	return ini.Parse(tmpName)
}

//ini配置对象，当设置或获取值时，支持section:name这种类型的key
type IniConfigContainer struct {
	filename       string
	data           map[string]map[string]string // section=> key:val
	sectionComment map[string]string            // section : comment
	keyComment     map[string]string            // id: []{comment, key...}; id 1 is for main comment.
	sync.RWMutex
}

//通过Key获取一个布尔值
func (c *IniConfigContainer) Bool(key string) (bool, error) {
	return ParseBool(c.getdata(key))
}

//无错误时返回一个布尔值，否则返回一个默认值
func (c *IniConfigContainer) DefaultBool(key string, defaultval bool) bool {
	v, err := c.Bool(key)
	if err != nil {
		return defaultval
	}
	return v
}

//通过Key返回一个Int型值
func (c *IniConfigContainer) Int(key string) (int, error) {
	return strconv.Atoi(c.getdata(key))
}

//无错误时返回一个int，否则返回一个默认值
func (c *IniConfigContainer) DefaultInt(key string, defaultval int) int {
	v, err := c.Int(key)
	if err != nil {
		return defaultval
	}
	return v
}

//通过Key返回一个int64型值
func (c *IniConfigContainer) Int64(key string) (int64, error) {
	return strconv.ParseInt(c.getdata(key), 10, 64)
}

//无错误时返回一个int64，否则返回一个默认值
func (c *IniConfigContainer) DefaultInt64(key string, defaultval int64) int64 {
	v, err := c.Int64(key)
	if err != nil {
		return defaultval
	}
	return v
}

//通过Key返回一个float型值
func (c *IniConfigContainer) Float(key string) (float64, error) {
	return strconv.ParseFloat(c.getdata(key), 64)
}

//无错误时返回一个float，否则返回一个默认值
func (c *IniConfigContainer) DefaultFloat(key string, defaultval float64) float64 {
	v, err := c.Float(key)
	if err != nil {
		return defaultval
	}
	return v
}

//通过Key返回一个string型
func (c *IniConfigContainer) String(key string) string {

	return c.getdata(key)
}

//无错误时返回一个string，否则返回一个默认值
func (c *IniConfigContainer) DefaultString(key string, defaultval string) string {
	v := c.String(key)
	if v == "" {
		return defaultval
	}
	return v
}

//通过Key返回一个string数组
func (c *IniConfigContainer) Strings(key string) []string {
	v := c.String(key)
	if v == "" {
		return nil
	}
	return strings.Split(v, ";")
}

//无错误时返回一个string数组，否则返回一个默认值
func (c *IniConfigContainer) DefaultStrings(key string, defaultval []string) []string {
	v := c.Strings(key)
	if v == nil {
		return defaultval
	}
	return v
}

//通过片段字符串返回一个map（ini文件可用，json文件不可用）
func (c *IniConfigContainer) GetSection(section string) (map[string]string, error) {
	if v, ok := c.data[section]; ok {
		return v, nil
	}
	return nil, errors.New("未找到该节点数据！")
}

//获取所有数据列表
func (c *IniConfigContainer) GetKeys() []string {
	var keys []string
	if len(c.data) > 0 {
		for k, _ := range c.data {
			keys = append(keys, k)
		}
	}
	return keys
}

//将配置信息数据保存到配置文件中
func (c *IniConfigContainer) SaveConfigFile(filename string) (err error) {
	// Write configuration file by filename.
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Get section or key comments. Fixed #1607
	getCommentStr := func(section, key string) string {
		comment, ok := "", false
		if len(key) == 0 {
			comment, ok = c.sectionComment[section]
		} else {
			comment, ok = c.keyComment[section+"."+key]
		}

		if ok {
			// Empty comment
			if len(comment) == 0 || len(strings.TrimSpace(comment)) == 0 {
				return string(bNumComment)
			}
			prefix := string(bNumComment)
			// Add the line head character "#"
			return prefix + strings.Replace(comment, lineBreak, lineBreak+prefix, -1)
		}
		return ""
	}

	buf := bytes.NewBuffer(nil)
	// Save default section at first place
	if dt, ok := c.data[defaultSection]; ok {
		for key, val := range dt {
			if key != " " {
				// Write key comments.
				if v := getCommentStr(defaultSection, key); len(v) > 0 {
					if _, err = buf.WriteString(v + lineBreak); err != nil {
						return err
					}
				}

				// Write key and value.
				if _, err = buf.WriteString(key + string(bEqual) + val + lineBreak); err != nil {
					return err
				}
			}
		}

		// Put a line between sections.
		if _, err = buf.WriteString(lineBreak); err != nil {
			return err
		}
	}
	// Save named sections
	for section, dt := range c.data {
		if section != defaultSection {
			// Write section comments.
			if v := getCommentStr(section, ""); len(v) > 0 {
				if _, err = buf.WriteString(v + lineBreak); err != nil {
					return err
				}
			}

			// Write section name.
			if _, err = buf.WriteString(string(sectionStart) + section + string(sectionEnd) + lineBreak); err != nil {
				return err
			}

			for key, val := range dt {
				if key != " " {
					// Write key comments.
					if v := getCommentStr(section, key); len(v) > 0 {
						if _, err = buf.WriteString(v + lineBreak); err != nil {
							return err
						}
					}

					// Write key and value.
					if _, err = buf.WriteString(key + string(bEqual) + val + lineBreak); err != nil {
						return err
					}
				}
			}

			// Put a line between sections.
			if _, err = buf.WriteString(lineBreak); err != nil {
				return err
			}
		}
	}

	if _, err = buf.WriteTo(f); err != nil {
		return err
	}
	return nil
}

//配置设置(可以通过访问符"."进行操作)
func (c *IniConfigContainer) Set(key, value string) error {
	c.Lock()
	defer c.Unlock()
	if len(key) == 0 {
		return errors.New("key is empty")
	}

	var (
		section, k string
		sectionKey = strings.Split(key, ".")
	)

	if len(sectionKey) >= 2 {
		section = sectionKey[0]
		k = sectionKey[1]
	} else {
		section = defaultSection
		k = sectionKey[0]
	}

	if _, ok := c.data[section]; !ok {
		c.data[section] = make(map[string]string)
	}
	c.data[section][k] = value
	return nil
}

//根据给定的Key自定义设置value
//支持":"定位。
/*如：ini:

获取时：cnf.GetVal("server1.master.ip")
*/
func (c *IniConfigContainer) GetVal(key string) (v interface{}, err error) {
	if v, ok := c.data[strings.ToLower(key)]; ok {
		return v, nil
	}
	return v, errors.New("key not find")
}

func (c *IniConfigContainer) getdata(key string) string {
	if len(key) == 0 {
		return ""
	}
	c.RLock()
	defer c.RUnlock()

	var (
		section, k string
		sectionKey = strings.Split(strings.ToLower(key), "->")
	)
	if len(sectionKey) >= 2 {
		section = sectionKey[0]
		k = sectionKey[1]
	} else {
		section = defaultSection
		k = sectionKey[0]
	}

	if v, ok := c.data[section]; ok {
		if vv, ok := v[k]; ok {
			return vv
		}
	}
	return ""
}

func init() {
	Register("ini", &IniConfig{})
}
