/*
	验证库
	@author : hyperion
	@since  : 2018-3-13
	@version: 1.0
*/
package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"time"
	"unicode/utf8"
)

var (
	//Email验证
	emailPattern = regexp.MustCompile("[\\w!#$%&'*+/=?^_`{|}~-]+(?:\\.[\\w!#$%&'*+/=?^_`{|}~-]+)*@(?:[\\w](?:[\\w-]*[\\w])?\\.)+[a-zA-Z0-9](?:[\\w-]*[\\w])?")
	//IP验证
	ipPattern = regexp.MustCompile("^((2[0-4]\\d|25[0-5]|[01]?\\d\\d?)\\.){3}(2[0-4]\\d|25[0-5]|[01]?\\d\\d?)$")
	//base64验证
	base64Pattern = regexp.MustCompile("^(?:[A-Za-z0-99+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$")
	//手机验证
	mobilePattern = regexp.MustCompile("^((\\+86)|(86))?(1(([35][0-9])|[8][0-9]|[7][06789]|[4][579]))\\d{8}$")
	//固定电话验证
	telPattern = regexp.MustCompile("^(0\\d{2,3}(\\-)?)?\\d{7,8}$")
	//英文和数字组合验证
	enAndNumericPattern = regexp.MustCompile("^[A-Za-z0-9]+$")
	//账号是否合法验证,允许5-20字节，允许字母数字下划线
	validAccountPattern = regexp.MustCompile("^[a-zA-Z0-9_]{4,20}$")
	//弱密码验证,以字母开头，长度在6~18之间，只能包含字母、数字和下划线
	weakPwdPattern = regexp.MustCompile("^[a-zA-Z]\\w{5,17}$")
	//强密码验证,必须包含大小写字母和数字的组合，不能使用特殊字符，长度在8个字符以上
	strongPwdPattern = regexp.MustCompile("^[A-Z]+[a-z0-9]{7,}$")
	//中文字符,必须为中文字符
	cnCharPattern = regexp.MustCompile("^[\u4e00-\u9fa5]{0,}$")
	//日期检验，已考虑平闰年，日期格式：yyyy-mm-dd
	//datePattern = regexp.MustCompile("^(?:(?!0000)[0-9]{4}-(?:(?:0[1-9]|1[0-2])-(?:0[1-9]|1[0-9]|2[0-8])|(?:0[13-9]|1[0-2])-(?:29|30)|(?:0[13578]|1[02])-31)|(?:[0-9]{2}(?:0[48]|[2468][048]|[13579][26])|(?:0[48]|[2468][048]|[13579][26])00)-02-29)$")
	datePattern = regexp.MustCompile("^[A-Za-z0-9]+$")
	//普通名称验证,中文英文数字，不能包含空格和标点
	commonName = regexp.MustCompile("^[0-9a-zA-Z\u4E00-\u9FA5]+$")
	//错误消息模板
	MessageTpl = map[string]string{
		"Required":   "值不可为空",
		"Range":      "取值范围必须在%d至%d之间",
		"Numeric":    "必须为数字类型",
		"Length":     "长度必须为%d",
		"Mobile":     "手机号有误",
		"IP":         "您输入的IP地址有误",
		"Email":      "您输入的Email地址有误",
		"Min":        "最小值必须为%d",
		"Max":        "最大值必须为%d",
		"MaxLen":     "最大长度必须为%d",
		"MinLen":     "最小长度必须为%d",
		"Phone":      "固定电话格式有误，格式：(010)81122333或010-811255 .88",
		"EnNumeric":  "输入的值只能包含英文字母和数字",
		"Account":    "只能包含字母、数字或下划线，请保持在5-20个字符之间",
		"CommonName": "名称只能包含中英文或数字",
		"WeakPwd":    "不符合要求，必须以字母开头，长度在6~18字符之间，只能包含字母、数字和下划线",
		"StrongPwd":  "不符合要求，必须包含大小写字母和数字的组合，不能使用特殊字符，长度在8个字符以上",
		"CnChar":     "必须包含中文字符",
		"Date":       "日期不符合要求，日期格式：yyyy-mm-dd",
	}
)

type (
	//验证器接口，新建某种规则的验证器必须实现以下方法
	Validator interface {
		IsSatisfied(data interface{}) bool //是否满足条件
		SetMessage() string                //设置错误信息
		GetLimitValue() interface{}        //获取限制值
		GetKey() string                    //获取键值
	}
	//正则匹配类
	Match struct {
		Regexp *regexp.Regexp
		Key    string
		TplKey string
	}
)

//===必填验证===start======
type Required struct{ Key string }

//判断是否满足，要看类型中是否存在值
func (r Required) IsSatisfied(data interface{}) bool {
	if data == nil {
		return false
	}
	if str, ok := data.(string); ok {
		return len(str) > 0
	}
	if _, ok := data.(bool); ok {
		return true
	}
	if i, ok := data.(int); ok {
		return i != 0
	}
	if i, ok := data.(uint); ok {
		return i != 0
	}
	if i, ok := data.(int8); ok {
		return i != 0
	}
	if i, ok := data.(uint8); ok {
		return i != 0
	}
	if i, ok := data.(int16); ok {
		return i != 0
	}
	if i, ok := data.(uint16); ok {
		return i != 0
	}
	if i, ok := data.(uint32); ok {
		return i != 0
	}
	if i, ok := data.(int32); ok {
		return i != 0
	}
	if i, ok := data.(int64); ok {
		return i != 0
	}
	if i, ok := data.(uint64); ok {
		return i != 0
	}
	if t, ok := data.(time.Time); ok {
		return !t.IsZero()
	}
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice {
		return v.Len() > 0
	}
	return true
}

//设置错误信息
func (r Required) SetMessage() string {
	return fmt.Sprint(MessageTpl["Required"])
}

//获取限制值
func (r Required) GetLimitValue() interface{} {
	return nil
}
func (r Required) GetKey() string {
	return r.Key
}

//===必填验证===end======
//===是否为数字验证===start======
type Numeric struct {
	Key string
}

//判断字符串是否为数字
func (n Numeric) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		for _, v := range str {
			if '9' < v || v < '0' {
				return false
			}
		}
		return true
	}
	return false
}

func (n Numeric) SetMessage() string {
	return fmt.Sprint(MessageTpl["Numeric"])
}

func (n Numeric) GetKey() string {
	return n.Key
}

func (n Numeric) GetLimitValue() interface{} {
	return nil
}

//===是否为数字验证===end======
//===最小值验证===start======
type Min struct {
	Min int
	Key string
}

//是否满足条件
func (m Min) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num >= m.Min
	}
	return false
}

//提示信息
func (m Min) SetMessage() string {
	return fmt.Sprintf(MessageTpl["Min"], m.Min)
}

func (m Min) GetKey() string {
	return m.Key
}

func (m Min) GetLimitValue() interface{} {
	return m.Min
}

//===最小值验证===end======
//===最大值验证===start======
type Max struct {
	Max int
	Key string
}

//是否满足条件
func (m Max) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num <= m.Max
	}
	return false
}

//提示信息
func (m Max) SetMessage() string {
	return fmt.Sprintf(MessageTpl["Max"], m.Max)
}
func (m Max) GetKey() string {
	return m.Key
}
func (m Max) GetLimitValue() interface{} {
	return m.Max
}

//===最大值验证===end======
//===取值范围验证===start======
//取值范围需要介于最大最小值之间
type Range struct {
	Min
	Max
	Key string
}

func (r Range) IsSatisfied(obj interface{}) bool {
	return r.Min.IsSatisfied(obj) && r.Max.IsSatisfied(obj)
}

func (r Range) SetMessage() string {
	return fmt.Sprintf(MessageTpl["Range"], r.Min.Min, r.Max.Max)
}

func (r Range) GetKey() string {
	return r.Key
}

func (r Range) GetLimitValue() interface{} {
	return []int{r.Min.Min, r.Max.Max}
}

//===取值范围验证===end======
//===最小值长度验证===start======
type MinLen struct {
	Min int
	Key string
}

func (m MinLen) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) >= m.Min
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() >= m.Min
	}
	return false
}

func (m MinLen) SetMessage() string {
	return fmt.Sprintf(MessageTpl["MinLen"], m.Min)
}

func (m MinLen) GetKey() string {
	return m.Key
}

func (m MinLen) GetLimitValue() interface{} {
	return m.Min
}

//===最小值长度验证===end======

//===最大值长度验证===start======
type MaxLen struct {
	Max int
	Key string
}

func (m MaxLen) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) <= m.Max
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() <= m.Max
	}
	return false
}

func (m MaxLen) SetMessage() string {
	return fmt.Sprintf(MessageTpl["MaxLen"], m.Max)
}

func (m MaxLen) GetKey() string {
	return m.Key
}

func (m MaxLen) GetLimitValue() interface{} {
	return m.Max
}

//===最大值长度验证===end======

//===正则验证===start===
func (m Match) IsSatisfied(data interface{}) bool {
	return m.Regexp.MatchString(fmt.Sprintf("%v", data))
}

//设置错误信息
func (m Match) SetMessage() string {
	return fmt.Sprintf(MessageTpl[m.TplKey])
}

//获取限制值
func (m Match) GetLimitValue() interface{} {
	return m.Regexp.String()
}

func (m Match) GetKey() string {
	return m.Key
}

//===正则验证===end===
