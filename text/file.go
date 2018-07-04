/*
	文件操作库
	@author : hyperion
	@since  : 2017-3-20
	@version: 1.0
*/
package Text

import (
	"os"
	"path/filepath"
	"strings"
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

//创建目录
func CreateDir(path string, filemode ...os.FileMode) bool {
	var fmode os.FileMode = 0777
	if len(filemode) > 0 {
		fmode = filemode[0]
	}
	err := os.MkdirAll(path, fmode)
	if err == nil {
		return true
	} else {
		return false
	}
}

//获取程序运行的路径
func GetAppDir() (appDir string) {
	var err error
	appDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err == nil {
		appDir = strings.Replace(appDir, "\\", "/", -1) //将\替换成/
	} else {
		appDir = ""
	}
	return appDir
}
