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

var (
	LogList  map[string]*Logger = nil //Log对象列表
	LogPath  string             = ""  //日志存放路径
	LT_DAY   int                = 1
	LT_MONTH int                = 2
)

type (
	Logger struct {
		Dir      string
		FilePath string
		Type     int
		Flags    int
	}
)

//log日志。最多添加10种类型的日志
func Log(filename ...string) *Logger {
	defer func() { //捕捉panic错误避免崩溃
		if r := recover(); r != nil {
			fmt.Printf("log error:%s", r)
		}
	}()
	var fname string = "app"
	if len(filename) > 0 {
		fname = filename[0]
	}
	if LogList == nil {
		LogList = make(map[string]*Logger, 10)
	}
	if _, ok := LogList[fname]; !ok { //不存在日志对象进行实例化
		log := &Logger{}
		if LogPath == "" {
			appDir := GetAppDir()
			if appDir != "" {
				log.Dir = SpliceString(appDir, "/log/")
			} else {
				log.Dir = "/tmp/log/"
			}
		}
		if !IsExists(log.Dir) { //如果目录不存在，则创建此目录
			CreateDir(log.Dir)
		}
		log.FilePath = SpliceString(log.Dir, fname, ".log")
		LogList[fname] = log
	}
	return LogList[fname]
}

//添加log日志。logtype-->默认（0）：信息日志；1：警告日志；2：错误日志；3：崩溃日志；4：调试日志；
func (l *Logger) Add(content string, logtype int) error {
	var infoTag = "[INFO]"
	if logtype == 1 {
		infoTag = "[WARNING]"
	} else if logtype == 2 {
		infoTag = "[ERROR]"
	} else if logtype == 3 {
		infoTag = "[FATAL]"
	} else if logtype == 4 {
		infoTag = "[DEBUG]"
	}
	logFile, logErr := os.OpenFile(l.FilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if logErr != nil {
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
func (l *Logger) Info(content string) error {
	l.Flags = 0
	return l.Add(content, 0)
}

//警告信息
func (l *Logger) Warning(content string) error {
	l.Flags = log.Lshortfile | log.LstdFlags
	return l.Add(content, 1)
}

//错误信息
func (l *Logger) Error(content string) error {
	l.Flags = log.Lshortfile | log.LstdFlags
	return l.Add(content, 2)
}

//严重错误
func (l *Logger) Fatal(content string) error {
	l.Flags = log.Lshortfile | log.LstdFlags
	return l.Add(content, 3)
}

//调试信息
func (l *Logger) Debug(content string) error {
	l.Flags = log.Lshortfile | log.LstdFlags
	return l.Add(content, 4)
}
