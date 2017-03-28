// frame
package aresgo

import (
	"aresgo/cache"
	"aresgo/config"
	"aresgo/data"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　　■■■■　　　　　　　　　　　　　　　` + "    Version：" + Version +
		"\r\n←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←←      " + ` `

	//数据库定义
	DbInsert = "INSERT"
	DbUpdate = "UPDATE"
	DbDelete = "DELETE"
	DbSelect = "SELECT"
)
//结构体定义
type (
	FrameWork struct{}
)

//初始化定义
var (
	StartTime string
	//数据库实例
	DS              *Db.DbModel                  = nil
	DbModels        map[string]*Db.DbModel       = make(map[string]*Db.DbModel)       //数据库对象列表
	DbConfigPath    string                       = ""                                 //数据库配置文件路径
	dbConfiger      config.Configer              = nil                                //数据库配置文件对象
	RedisModels     map[string]*Cache.RedisModel = make(map[string]*Cache.RedisModel) //Redis对象列表
	RedisConfigPath string                       = ""                                 //Redis配置文件路径
	CacheConfigPath string                       = ""                                 //缓存配置文件路径
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
			//			fmt.Printf("%v\r\n", err.Error())
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
		//		dbConfiger, err := LoadConfig("json", DbConfigPath)
		if err != nil {
			return err
		}
	}
	//设置数据库主从配置，从配置文件中获取
	var settings map[string]*Db.DbSettings = make(map[string]*Db.DbSettings)
	dbreader := &Db.DbSettings{
		Ip:        dbConfiger.DefaultString(fmt.Sprintf("%s.slave.ip", dbkey), "127.0.0.1"),
		Port:      dbConfiger.DefaultString(fmt.Sprintf("%s.slave.port", dbkey), "3306"),
		User:      dbConfiger.DefaultString(fmt.Sprintf("%s.slave.user", dbkey), "root"),
		Password:  dbConfiger.DefaultString(fmt.Sprintf("%s.slave.password", dbkey), ""),
		Charset:   dbConfiger.DefaultString(fmt.Sprintf("%s.slave.charset", dbkey), "utf8"),
		DefaultDb: dbConfiger.DefaultString(fmt.Sprintf("%s.slave.db", dbkey), ""),
	}
	dbwriter := &Db.DbSettings{
		Ip:        dbConfiger.DefaultString(fmt.Sprintf("%s.master.ip", dbkey), "127.0.0.1"),
		Port:      dbConfiger.DefaultString(fmt.Sprintf("%s.master.port", dbkey), "3306"),
		User:      dbConfiger.DefaultString(fmt.Sprintf("%s.master.user", dbkey), "root"),
		Password:  dbConfiger.DefaultString(fmt.Sprintf("%s.master.password", dbkey), ""),
		Charset:   dbConfiger.DefaultString(fmt.Sprintf("%s.master.charset", dbkey), "utf8"),
		DefaultDb: dbConfiger.DefaultString(fmt.Sprintf("%s.master.db", dbkey), ""),
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
			var settings map[string]*Cache.RedisSettings = make(map[string]*Cache.RedisSettings)

			settings["master"] = &Cache.RedisSettings{
				IP:          redisConfiger.DefaultString(fmt.Sprintf("%s.master.ip", redisKey), "127.0.0.1"),
				Port:        redisConfiger.DefaultString(fmt.Sprintf("%s.master.port", redisKey), "6379"),
				Password:    redisConfiger.DefaultString(fmt.Sprintf("%s.master.password", redisKey), ""),
				DbNum:       redisConfiger.DefaultInt(fmt.Sprintf("%s.master.db", redisKey), 0),
				MaxIdle:     redisConfiger.DefaultInt(fmt.Sprintf("%s.master.maxidle", redisKey), 3),
				MaxActive:   redisConfiger.DefaultInt(fmt.Sprintf("%s.master.maxactive", redisKey), 1000),
				IdleTimeout: redisConfiger.DefaultInt(fmt.Sprintf("%s.master.idletimeout", redisKey), 180),
			}

			settings["slave"] = &Cache.RedisSettings{
				IP:          redisConfiger.DefaultString(fmt.Sprintf("%s.slave.ip", redisKey), "127.0.0.1"),
				Port:        redisConfiger.DefaultString(fmt.Sprintf("%s.slave.port", redisKey), "6379"),
				Password:    redisConfiger.DefaultString(fmt.Sprintf("%s.slave.password", redisKey), ""),
				DbNum:       redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.db", redisKey), 0),
				MaxIdle:     redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.maxidle", redisKey), 3),
				MaxActive:   redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.maxactive", redisKey), 1000),
				IdleTimeout: redisConfiger.DefaultInt(fmt.Sprintf("%s.slave.idletimeout", redisKey), 180),
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

//加载数据库配置文件
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
	} else {
		appPath = ""
	}
	appDir, err = os.Getwd()
	if err == nil {
		appDir = strings.Replace(appDir, "\\", "/", -1)
	} else {
		appDir = ""
	}
	return appPath, appDir
}
