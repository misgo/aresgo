/*
	验证库
	@author : hyperion
	@since  : 2018-3-13
	@version: 1.0
*/
package validation

import (
	//"regexp"
	"fmt"
	"strings"
)

type (
	Error struct {
		Message, Key, Name, Field, Tpl string
		Value, LimitValue              interface{}
	}
	//验证类
	Validation struct {
		Errors    []*Error
		ErrorsMap map[string]*Error
	}

	//返回的验证类结果
	Result struct {
		Error *Error
		Ok    bool
	}
)

//Error方法--返回错误信息
func (e *Error) String() string {
	if e == nil {
		return ""
	}
	return e.Message
}

//Result方法--生成错误信息
func (r *Result) Message(msg string, args ...interface{}) *Result {
	if r.Error != nil {
		if len(args) == 0 {
			r.Error.Message = msg
		} else {
			r.Error.Message = fmt.Sprintf(msg, args...)
		}
	}
	return r
}

//Validation方法---是否有错误
func (v *Validation) HasErrors() bool {
	return len(v.Errors) > 0
}

//Validation方法---清除所有错误
func (v *Validation) Clear() {
	v.Errors = []*Error{}
	v.ErrorsMap = nil
}

//Validation方法---为验证器添加错误内容
func (v *Validation) Error(message string, args ...interface{}) *Result {
	result := (&Result{
		Ok:    false,
		Error: &Error{},
	}).Message(message, args...)
	v.Errors = append(v.Errors, result.Error)
	return result
}

//Validation方法---是否为必填项
func (v *Validation) Required(data interface{}, key string) *Result {
	return v.validate(Required{Key: key}, data)
}

//Validation方法---是否为数字类型
func (v *Validation) Numeric(obj interface{}, key string) *Result {
	return v.validate(Numeric{key}, obj)
}

//Validation方法---最小值验证，给定值必须比最小值大
func (v *Validation) Min(obj interface{}, min int, key string) *Result {
	return v.validate(Min{min, key}, obj)
}

//Validation方法---最大值验证，给定值必须比最小值小
func (v *Validation) Max(obj interface{}, max int, key string) *Result {
	return v.validate(Max{max, key}, obj)
}

//Validatio 方法---取值范围验证，必须介入最大最小值之间
func (v *Validation) Range(obj interface{}, min, max int, key string) *Result {
	return v.validate(Range{Min{Min: min}, Max{Max: max}, key}, obj)
}

//Validatio 方法---最小长度验证，类型为：string or slice
func (v *Validation) MinLen(obj interface{}, min int, key string) *Result {
	return v.validate(MinLen{min, key}, obj)
}

//Validatio 方法---最大长度验证，类型为：string or slice
func (v *Validation) MaxLen(obj interface{}, max int, key string) *Result {
	return v.validate(MaxLen{max, key}, obj)
}

//Validation方法---Email验证
func (v *Validation) Email(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: emailPattern, Key: key, TplKey: "Email"}, data)
}

//Validation方法---IP验证
func (v *Validation) IP(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: ipPattern, Key: key, TplKey: "IP"}, data)
}

//Validation方法---手机号验证
func (v *Validation) Mobile(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: mobilePattern, Key: key, TplKey: "Mobile"}, data)
}

//Validation方法---固定电话号验证
func (v *Validation) Phone(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: telPattern, Key: key, TplKey: "Phone"}, data)
}

//Validation方法---英文跟数字组合验证
func (v *Validation) EnNumeric(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: enAndNumericPattern, Key: key, TplKey: "EnNumeric"}, data)
}

//Validation方法---账号是否合法验证
func (v *Validation) ValidAccount(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: validAccountPattern, Key: key, TplKey: "Account"}, data)
}

//Validation方法---弱密码验证
func (v *Validation) WeakPassword(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: weakPwdPattern, Key: key, TplKey: "WeakPwd"}, data)
}

//Validation方法---强密码验证
func (v *Validation) StrongPassword(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: strongPwdPattern, Key: key, TplKey: "StrongPwd"}, data)
}

//Validation方法---中文字符验证
func (v *Validation) ChineseChar(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: cnCharPattern, Key: key, TplKey: "cnCharPattern"}, data)
}

//Validation方法---日期验证
func (v *Validation) ValidDate(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: datePattern, Key: key, TplKey: "Date"}, data)
}

//Validation方法---普通名称验证
func (v *Validation) CommonName(data interface{}, key string) *Result {
	return v.validate(Match{Regexp: commonName, Key: key, TplKey: "CommonName"}, data)
}

//验证是否满足条件
func (v *Validation) validate(valObj Validator, data interface{}) *Result {
	if valObj.IsSatisfied(data) {
		return &Result{Ok: true}
	}
	key := valObj.GetKey()
	Name := key
	Field := ""
	parts := strings.Split(key, ".")
	if len(parts) == 2 {
		Field = parts[0]
		Name = parts[1]
	}
	err := &Error{
		Message:    valObj.SetMessage(),
		Key:        key,
		Name:       Name,
		Field:      Field,
		Value:      data,
		Tpl:        MessageTpl[Name],
		LimitValue: valObj.GetLimitValue(),
	}
	v.setError(err)
	return &Result{
		Ok:    false,
		Error: err,
	}
}

//添加错误信息到信息列表中
func (v *Validation) setError(err *Error) {
	v.Errors = append(v.Errors, err)
	if v.ErrorsMap == nil {
		v.ErrorsMap = make(map[string]*Error)
	}
	if _, ok := v.ErrorsMap[err.Field]; !ok {
		v.ErrorsMap[err.Field] = err
	}
}
