package aresgo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/misgo/aresgo/cache"
	"github.com/misgo/aresgo/config"
	"github.com/misgo/aresgo/data"
	//	"github.com/misgo/aresgo/framework"
	//	"github.com/misgo/aresgo/text"
)

//常量定义
const (
	Version = "1.0"
	Banner  = "→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→\r\n" + `
　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　
　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　
　　　■■■　　　　■　■　　　■■■　　　　　■■■■　　　　■■■　　　　　　■■■　　　
　■■■■■■■　　■■■　　■■■■■　　　■■■■■　　　■■■■■　　　■■■■■■■　
　■■　　　　■　　■　　　■■　　　■■　■■　　　　　　■■　　　■■　　■　　　　■■　
　■　　　　　■　　■　　　■　　■■■　　　■■■　　　　■　　　　　■　　■　　　　　■　
　■　　　　　■　　■　　■■　■■　　　　　　■■■■　　■　　　　　■　　■　　　　　■　
　■　　　　　■　　■　　　■　　　　　■　　　　　　■■　■　　　　　■　　■　　　　　■　
　■■　　　　■　　■　　　■■　　　■■　　　　　　■■　■■　　　　■　　■■　　　■■　
　　■■■■　■　　■　　　　■■■■■　　■■■■■■　　　■■■■　■　　　■■■■■　　
　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　■　　　　　　　　　　
　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　■■　　　　　　　　　　
　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　■■■■■　　　　　　　　　　　
　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　■■■■　　　　　　　　　　　　　　　` + "    Version：❤️" + Version +
		"\r\n←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←      " + ` `

	//数据库定义
	DbInsert = "INSERT"
	DbUpdate = "UPDATE"
	DbDelete = "DELETE"
	DbSelect = "SELECT"
	//存储模式定义
	ModeRedis = "redis"
	ModeDb    = "db"
)

//初始化定义
var (
	StartTime string //程序启动时间
	//---数据库---
	DS           *Db.DbModel            = nil                          //当前数据库对象实例
	DbModels     map[string]*Db.DbModel = make(map[string]*Db.DbModel) //数据库对象列表
	DbConfigPath string                 = ""                           //数据库配置文件路径
	dbConfiger   config.Configer        = nil                          //数据库配置文件对象
	//---Redis缓存---
	RS              *Cache.RedisModel            = nil                                //当前Redis对象实例
	RedisModels     map[string]*Cache.RedisModel = make(map[string]*Cache.RedisModel) //Redis对象列表
	RedisConfigPath string                       = ""                                 //Redis配置文件路径
	CacheConfigPath string                       = ""                                 //缓存配置文件路径
	//---用户自定义---
	CustomVar     map[string]interface{} //用户自定义全局变量
	TemplatePaths map[string][]string    //用户自定义页面模板列表

)

//初始化数据库配置
func InitMysql(config map[string]*Db.DbSettings) {
	DS = Db.NewDb("mysql", config)
}

//通过Key获取数据库访问对象
func D(dbkey string) *Db.DbModel {
	var ds *Db.DbModel = nil
	if _, ok := DbModels[dbkey]; ok { //能取到数据库对象
		ds = DbModels[dbkey]
	}
	if ds == nil { //数据库对象不存在
		err := getDbModel(dbkey)
		if err != nil {
			return &Db.DbModel{}
		}
	}
	return DbModels[dbkey]
}

//获取数据对象
func getDbModel(dbkey string) error {
	//如果数据库配置文件未加载，则先加载配置文件
	if dbConfiger == nil {
		err := loadDbConfig()
		if err != nil {
			return err
		}
	}
	//设置数据库主从配置，从配置文件中获取
	var settings map[string]*Db.DbSettings = make(map[string]*Db.DbSettings, 2)
	//从库配置
	dbreader := &Db.DbSettings{
		Ip:        dbConfiger.DefaultString(fmt.Sprintf("%s.slave.ip", dbkey), "127.0.0.1"),
		Port:      dbConfiger.DefaultString(fmt.Sprintf("%s.slave.port", dbkey), "3306"),
		User:      dbConfiger.DefaultString(fmt.Sprintf("%s.slave.user", dbkey), "root"),
		Password:  dbConfiger.DefaultString(fmt.Sprintf("%s.slave.password", dbkey), ""),
		Charset:   dbConfiger.DefaultString(fmt.Sprintf("%s.slave.charset", dbkey), "utf8"),
		DefaultDb: dbConfiger.DefaultString(fmt.Sprintf("%s.slave.db", dbkey), ""),
	}
	//主库配置
	dbwriter := &Db.DbSettings{
		Ip:        dbConfiger.DefaultString(fmt.Sprintf("%s.master.ip", dbkey), "127.0.0.1"),
		Port:      dbConfiger.DefaultString(fmt.Sprintf("%s.master.port", dbkey), "3306"),
		User:      dbConfiger.DefaultString(fmt.Sprintf("%s.master.user", dbkey), "root"),
		Password:  dbConfiger.DefaultString(fmt.Sprintf("%s.master.password", dbkey), ""),
		Charset:   dbConfiger.DefaultString(fmt.Sprintf("%s.master.charset", dbkey), "utf8"),
		DefaultDb: dbConfiger.DefaultString(fmt.Sprintf("%s.master.db", dbkey), ""),
	}
	//表前缀只设置主库，主从公用相同的表前缀
	if dbConfiger.DefaultBool(fmt.Sprintf("%s.enable_tbpre", dbkey), false) {
		dbwriter.EnableTbPre = true
		dbwriter.TbPre = dbConfiger.DefaultString(fmt.Sprintf("%s.tbpre", dbkey), "")

	}
	settings["master"] = dbwriter
	settings["slave"] = dbreader
	db := Db.NewDb("mysql", settings)
	DbModels[dbkey] = db
	return nil
}

//加载数据库配置文件
func loadDbConfig() error {
	if DbConfigPath != "" {
		conf, err := config.NewConfig("json", DbConfigPath)
		if err == nil { //获取成功
			dbConfiger = conf
			return nil
		} else {
			return errors.New("数据配置文件解析错误！")
		}
	} else {
		return errors.New("未设置数据库配置文件路径！")
	}
}

//通过Key获取Redis访问对象
func R(redisKey string) *Cache.RedisModel {
	var rs *Cache.RedisModel = nil
	if _, ok := RedisModels[redisKey]; ok {
		rs = RedisModels[redisKey]
	}
	if rs == nil {
		err := getRedisModel(redisKey)
		if err != nil {
			return &Cache.RedisModel{}
		}
	}
	return RedisModels[redisKey]
}

//获取Redis实例
func getRedisModel(redisKey string) error {
	if RedisConfigPath != "" {
		redisConfiger, err := LoadConfig("json", RedisConfigPath)
		if err == nil {
			var settings map[string]*Cache.RedisSettings = make(map[string]*Cache.RedisSettings, 2)
			settings["master"] = &Cache.RedisSettings{
				IP:          redisConfiger.DefaultString(fmt.Sprintf("%s.master.ip", redisKey), "127.0.0.1"),
				Port:        redisConfiger.DefaultString(fmt.Sprintf("%s.master.port", redisKey), "6379"),
				Password:    redisConfiger.DefaultString(fmt.Sprintf("%s.master.password", redisKey), ""),
				DbNum:       redisConfiger.DefaultInt(fmt.Sprintf("%s.master.db", redisKey), 0),
				MaxIdle:     redisConfiger.DefaultInt(fmt.Sprintf("%s.master.maxidle", redisKey), 3),
				MaxActive:   redisConfiger.DefaultInt(fmt.Sprintf("%s.master.maxactive", redisKey), 1000),
				IdleTimeout: redisConfiger.DefaultInt(fmt.Sprintf("%s.master.idletimeout", redisKey), 180),
				KeyPre:      redisConfiger.DefaultString(fmt.Sprintf("%s.master.key_pre", redisKey), "misgo_"),
			}

			settings["slave"] = &Cache.RedisSettings{
				IP:          redisConfiger.DefaultString(fmt.Sprintf("%s.slave.ip", redisKey), "127.0.0.1"),
				Port:        redisConfiger.DefaultString(fmt.Sprintf("%s.slave.port", redisKey), "6379"),
				Password:    redisConfiger.DefaultString(fmt.Sprintf("%s.slave.password", redisKey), ""),
				DbNum:       redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.db", redisKey), 0),
				MaxIdle:     redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.maxidle", redisKey), 3),
				MaxActive:   redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.maxactive", redisKey), 1000),
				IdleTimeout: redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.idletimeout", redisKey), 180),
				KeyPre:      redisConfiger.DefaultString(fmt.Sprintf("%s.slave.key_pre", redisKey), "misgo_"),
			}
			RedisModels[redisKey] = Cache.NewRedis(settings)
			return nil
		} else {
			return err
		}
	} else {
		return errors.New("未设置Redis配置文件路径！")
	}

}

//加载配置文件
func LoadConfig(ctype string, filePath string) (config.Configer, error) {
	if filePath != "" {
		conf, err := config.NewConfig(ctype, filePath)
		if err == nil { //获取成功
			return conf, nil
		} else {
			return nil, errors.New("文件解析错误,请检查文件格式！")
		}
	} else {
		return nil, errors.New("未设置文件路径！")
	}
}

//获取程序运行的路径
func GetAppPath() (appPath string, appDir string) {
	var err error
	appPath, err = filepath.Abs(os.Args[0])
	if err == nil {
		appPath = strings.Replace(appPath, "\\", "/", -1)
		appDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
		if err == nil {
			appDir = strings.Replace(appDir, "\\", "/", -1) //将\替换成/
		} else {
			appDir = ""
		}
	} else {
		appPath = ""
	}
	return appPath, appDir
}

//访问远端接口，支持POST和GET
// func Curl(action string, url string, params map[string]interface{}) ([]byte, error) {
// 	var err error
// 	var bodystr []byte = nil
// 	if params != nil {
// 		if len(params) > 0 {
// 			sb := Text.NewString("")
// 			for k, v := range params {
// 				val := Text.GetBytes(v)
// 				if err == nil {
// 					sb.AppendBytes([]byte("&"))
// 					sb.AppendBytes([]byte(k))
// 					sb.AppendBytes([]byte("="))
// 					sb.AppendBytes(val)
// 				}
// 			}
// 			bodystr = sb.ToBytes()
// 		}
// 	}
// 	return frame.Curl(action, url, "form", bodystr)
// }
