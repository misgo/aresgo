/*
	加解密库
	@author : hyperion
	@since  : 2017-2-7
	@version: 1.0
*/
package Text

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	base64Header = "hyperion"
	base64Footer = "abcdef"
)

//将字符串进行Md5加密
func Md5(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	m := md5.New()
	m.Write([]byte(plaintext))
	return hex.EncodeToString(m.Sum(nil))
}

//构造密码
func BuildPwd(pwd string, salt string) string {
	return Md5(SpliceString("go", pwd, "love", salt))
}

//生成凭证--根据ID和扰码生成唯一Token
func CreateCert(id int64, salt string) string {
	nowtime := strconv.FormatInt(time.Now().Unix(), 10)
	idStr := strconv.FormatInt(id, 10)
	return Md5(SpliceString(idStr, salt, nowtime))
}

//获取Guid
func Guid() string {
	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return Md5(base64.URLEncoding.EncodeToString(b))
}

//Base64加密
func Base64Encode(str string, code ...string) string {
	var header string = ""
	var footer string = ""
	var codelen int = len(code)
	if codelen > 0 && codelen <= 1 {
		header = code[0]
	} else if codelen > 1 {
		footer = code[1]
	}
	var src []byte = []byte(header + str + footer)
	return string([]byte(base64.StdEncoding.EncodeToString(src)))
}

//Base64解密
func Base64Decode(str string, code ...string) (string, error) {
	var header string = ""
	var footer string = ""
	var codelen int = len(code)
	if codelen > 0 && codelen <= 1 {
		header = code[0]
	} else if codelen > 1 {
		footer = code[1]
	}
	var src []byte = []byte(str)
	b, err := base64.StdEncoding.DecodeString(string(src))
	return strings.Replace(strings.Replace(string(b), header, "", -1), footer, "", -1), err
}
