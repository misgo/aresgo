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
	mrand "math/rand"
	"strconv"
	"strings"
	"time"
)

const (
	base64Header = "hyperion"
	base64Footer = "abcdef"
	RAND_NUM     = 0 //纯数字
	RAND_LOWER   = 1 //小写字母
	RAND_UPPER   = 2 //	大写字母
	RAND_ALL     = 3 //数字、小写字母、大写字母
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

//构造身份验签
func BuildSign(publicKey string, privateKey string, paras []string) string {
	keylen := len(publicKey)
	if keylen > 15 {
		keylen = 15
	}
	key := SpliceString(CutStr(publicKey, 0, keylen), privateKey)
	sb := NewString(key)
	for _, v := range paras {
		//过滤特殊字符，待添加~~
		sb.Append(v)
	}
	return Md5(sb.ToString())
}

//生成凭证--根据ID和扰码生成唯一Token
func CreateCert(id int64, salt string) string {
	nowtime := strconv.FormatInt(time.Now().Unix(), 10)
	idStr := strconv.FormatInt(id, 10)
	return Md5(SpliceString(idStr, salt, nowtime))
}

//生成随机字符串
//0：纯数字；1：小写英文；2：大写英文；3:大小写英文；
func RandStr(size int, str_type ...int) []byte {
	var kind int
	if len(str_type) < 1 {
		kind = 3
	} else {
		kind = str_type[0]
	}
	ikind, kinds, result := kind, [][]int{[]int{10, 48}, []int{26, 97}, []int{26, 65}}, make([]byte, size)
	is_all := kind > 2 || kind < 0
	mrand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if is_all { // random ikind
			ikind = mrand.Intn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		result[i] = uint8(base + mrand.Intn(scope))
	}
	return result
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
