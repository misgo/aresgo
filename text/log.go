/*
	日志库
	@author : hyperion
	@since  : 2017-3-8
	@version: 1.0
*/
package Text

import (
	"fmt"
	"log"
	"os"
)

var LT_DAY int = 1
var LT_MONTH int = 2

type (
	Log struct {
		Dir      string
		FilePath string
		Type     int
		Flags    int
	}
)

//将字符串进行Md5加密
func (l *Log) Add(content string, logtype int) error {
	var infoTag = "[INFO]"
	if logtype == 1 {
		infoTag = "[WARNING]"
	} else if logtype == 2 {
		infoTag = "[ERROR]"
	} else if logtype == 3 {
		infoTag = "[FATAL]"
	}

	logFile, logErr := os.OpenFile(l.FilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if logErr != nil {
		fmt.Println("Fail to find", *logFile, "Server start Failed")
		os.Exit(1)
		return logErr
	}
	log.SetOutput(logFile)
	if l.Flags != 0 {
		log.SetFlags(l.Flags)
	}

	log.SetPrefix(infoTag)
	log.Println(content)
	return nil

}

//普通信息
func (l *Log) Info(content string) error {
	l.Flags = 0
	return l.Add(content, 0)
}

//警告信息
func (l *Log) Warning(content string) error {
	l.Flags = log.Lshortfile | log.LstdFlags
	return l.Add(content, 1)
}

//错误信息
func (l *Log) Error(content string) error {
	l.Flags = log.Llongfile | log.LstdFlags
	return l.Add(content, 2)
}

//严重错误
func (l *Log) Fatal(content string) error {
	l.Flags = log.Llongfile | log.LstdFlags
	return l.Add(content, 3)
}
