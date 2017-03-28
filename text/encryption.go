/*
	加解密库
	@author : hyperion
	@since  : 2017-2-7
	@version: 1.0
*/
package Text

import (
	"crypto/md5"
	"encoding/hex"
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
