/*
	文件操作库
	@author : hyperion
	@since  : 2017-3-20
	@version: 1.0
*/
package Text

import (
	"os"
)

//将字符串进行Md5加密
func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	} else {
		return true
	}
}
