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

//判断当前路径的文件是否存在
func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	} else {
		return true
	}
}
