/*
	缓存操作类库-Redis库，使用redisgo库，链接使用缓存池
	@author : hyperion
	@since  : 2017-01-20
	@version: 1.0
*/
package Cache

import (
	"aresgo/cache/redigo/redis"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type (
	RedisModel struct {
		//Redis缓存池,如果不用池进行管理，每当要操作redis时，建立连接，用完后再关闭，会导致大量的连接处于TIME_WAIT状态（TIME_WAIT，也叫TCP半连接状态，会继续占用本地端口）
		redisReader *redis.Pool
		redisWriter *redis.Pool

		readerSettings *RedisSettings
		writerSettings *RedisSettings
	}
	RedisSettings struct {
		IP          string //IP地址
		Port        string //端口号
		Password    string //密码
		MaxIdle     int
		IdleTimeout int
		MaxActive   int

		DbNum int //默认数据编号
	}
)

//实例化Redis
func NewRedis(settings map[string]*RedisSettings) *RedisModel {
	r := &RedisModel{}
	r.readerSettings = settings["slave"]
	r.writerSettings = settings["master"]

	r.redisReader = r.Connect(r.readerSettings)
	r.redisWriter = r.Connect(r.writerSettings)
	//	fmt.Printf("%v\r\n", r.redisReader)
	return r
}

//链接Redis
func (r *RedisModel) Connect(settings *RedisSettings) *redis.Pool {
	dailFunc := func() (rc redis.Conn, err error) {
		connectStr := fmt.Sprintf("%s:%s", settings.IP, settings.Port)
		rc, err = redis.Dial("tcp", connectStr)
		if err != nil {
			return nil, err
		}
		//如果有密码，需要进行权限认证
		if settings.Password != "" {
			if _, err := rc.Do("AUTH", settings.Password); err != nil {
				rc.Close()
				return nil, err
			}
		}
		//选择默认库
		_, selectErr := rc.Do("SELECT", settings.DbNum)
		if selectErr != nil {
			rc.Close()
			return nil, selectErr
		}

		return
	}

	testFunc := func(rc redis.Conn, t time.Time) error {
		//		if time.Since(t) < time.Minute {
		//			return nil
		//		}
		_, err := rc.Do("PING")
		return err
	}

	//创建redis访问池
	return &redis.Pool{
		MaxIdle:      settings.MaxIdle,
		IdleTimeout:  time.Duration(settings.IdleTimeout) * time.Second,
		MaxActive:    settings.MaxActive,
		Dial:         dailFunc,
		TestOnBorrow: testFunc,
	}

}

//Redis链接测试
func (r *RedisModel) Ping() error {
	var err error
	reader := r.redisReader.Get()
	err = r.redisReader.TestOnBorrow(reader, time.Now())
	if err == nil {
		writer := r.redisWriter.Get()
		err = r.redisWriter.TestOnBorrow(writer, time.Now())
	}

	return err
}

//选择库
func (r *RedisModel) Select(num int) bool {
	_, err := r.Do("SELECT", num)
	if err == nil {
		return true
	} else {
		return false
	}
}

//获取单个值-字符串
func (r *RedisModel) GetString(key string, hashKey ...string) string {
	val, _ := r.String(r.Get(key, hashKey...))
	return val
}

//获取单个值-整型
func (r *RedisModel) GetInt(key string, hashKey ...string) int {
	val, _ := r.Int(r.Get(key, hashKey...))
	return val
}

//获取单个值-整型
func (r *RedisModel) GetInt64(key string, hashKey ...string) int64 {
	val, _ := r.Int64(r.Get(key, hashKey...))
	return val
}

//获取单个值-布尔型
func (r *RedisModel) GetBool(key string, hashKey ...string) bool {
	val, _ := r.Bool(r.Get(key, hashKey...))
	return val
}

//获取单个值(兼容string和hash)
func (r *RedisModel) Get(key string, hashKey ...string) (interface{}, error) {
	if key == "" {
		return nil, errors.New("Key不可以为空")
	}

	if len(hashKey) > 0 {
		return r.Query("hget", key, hashKey[0])
	} else {
		return r.Query("get", key)
	}
}

//获取Hash结构的多条记录
//@param keys hash结构组，map[`redis key`][]`hash key`
//@return map[`redis key`]map[`hash key`]`hash value`
func (r *RedisModel) GetHashList(keys map[string][]string) (map[string]map[string]interface{}, error) {
	lkeys := len(keys)
	if lkeys < 1 {
		return nil, errors.New("Key不可以为空")
	}

	res := make(map[string]map[string]interface{}) //结果集
	var err error                                  //错误信息
	var c redis.Conn                               //redis链接
	c, err = r.getConn(r.redisReader, r.readerSettings)
	defer c.Close()
	if err != nil { //链接不通
		return nil, err
	}
	var allKeys [][]interface{}
	for k, v := range keys {
		var key []interface{}
		if len(v) > 0 { //取部分
			key = append(key, k)
			for _, str := range v {
				key = append(key, str)
			}
			err = c.Send("HMGET", key...)
			if err != nil {
				key = make([]interface{}, 0) //有错误将此值赋空值
			}

		}
		allKeys = append(allKeys, key)
	}
	if err = c.Flush(); err != nil {
		return nil, err
	}
	var reply interface{}
	for i := 0; i < lkeys; i++ {
		if len(allKeys[i]) > 0 {
			if reply, err = c.Receive(); err == nil {
				redisKey := allKeys[i][:1][0].(string)
				hashKeys := allKeys[i][1:]
				res[redisKey] = r.buildKeyVals(hashKeys, reply)
			}
		}
	}
	return res, nil
}

//获取多个值（string型）
func (r *RedisModel) GetStrList(keys ...interface{}) map[string]interface{} {
	lkeys := len(keys)
	res := make(map[string]interface{})
	if lkeys > 0 {
		vals, err := r.Query("MGET", keys...) //获取值
		if err == nil {
			res = r.buildKeyVals(keys, vals)
		}
	}
	return res
}

//将key与查询出的vals做对应关系
func (r *RedisModel) buildKeyVals(keys []interface{}, vals interface{}) map[string]interface{} {
	rv := reflect.Indirect(reflect.ValueOf(vals))
	rt := strings.Replace(rv.Type().String(), " ", "", -1)
	var res map[string]interface{} = make(map[string]interface{})
	if rt == "[]interface{}" {
		valsArr := vals.([]interface{})
		if len(keys) == len(valsArr) {
			for k, v := range keys {
				mKey, _ := r.String(v, nil)
				if mKey != "" {
					res[mKey] = valsArr[k]
				}
			}

		}
	}
	return res
}

//设置值-单条
//@param key 键
//@param val 值
//@param timeout 过期时间，单位：秒
func (r *RedisModel) Set(key string, val interface{}, timeout ...int64) bool {
	var expire int64 = 0 //超时时间
	var err error = errors.New("保存失败")
	//超时时间设定
	if len(timeout) > 0 {
		if timeout[0] > 0 {
			expire = timeout[0]
		}
	}
	if expire > 0 {
		_, err = r.Do("SETEX", key, expire, val)
	} else {
		_, err = r.Do("SET", key, val)
	}
	if err == nil {
		return true
	} else {
		return false
	}
}

//保存值多条(兼容string和hash)
//@param vals key-value列表
//@param hashKey hash表在redis中的Key（如果传输此参数，vals中的Key-Value对应Hash中的Key-Value）
//@return 成功的条数
func (r *RedisModel) SetValues(vals map[string]interface{}, hashKey ...string) int {
	valLen := len(vals)
	if valLen < 1 {
		return 0
	}
	var successNum int = 0 //处理成功个数
	var err error          //错误信息
	var c redis.Conn       //redis链接

	c, err = r.getConn(r.redisWriter, r.writerSettings)
	defer c.Close()
	if err != nil { //链接不通返回空
		return 0
	}
	if len(hashKey) > 0 {
		for k, v := range vals {
			err = c.Send("HSET", hashKey[0], k, v)
		}
	} else {
		for k, v := range vals {
			err = c.Send("SET", k, v)
		}
	}

	if err = c.Flush(); err != nil {
		return 0
	}

	for i := 0; i < valLen; i++ {
		if _, err = c.Receive(); err == nil {
			successNum += 1
		}
	}

	return successNum
}

//哈希表设置值-单条
func (r *RedisModel) HSet(hashKey string, key string, val interface{}) bool {
	if hashKey == "" || key == "" {
		return false
	}
	_, err := r.Do("hset", hashKey, key, val)
	if err != nil {
		return false
	} else {
		return true
	}
}

//删除键或Hash中的多个建（兼容hash）
//@param key redis中的键
//@param hashKey hash中的键列表，一个或多个（如果此处不填则删除整个键，否则只删除hash中的单个键）
func (r *RedisModel) Del(key string, hashKey ...string) bool {
	if key == "" {
		return false
	}
	var reply interface{}
	var err error
	if len(hashKey) > 0 {
		var keys []interface{}
		keys = append(keys, key)
		for _, v := range hashKey {
			keys = append(keys, v)
		}
		reply, err = r.Do("HDEL", keys...)
	} else {
		reply, err = r.Do("DEL", key)
	}
	if err != nil {
		return false
	} else {
		var b bool
		b, err = redis.Bool(reply, err)
		return b
	}
}

//设置键的失效时间
func (r *RedisModel) SetTimeout(key string, second int) bool {
	if key == "" {
		return false
	}
	reply, err := r.Do("EXPIRE", key, second)
	if err != nil {
		return false
	} else {
		var b bool
		b, err = redis.Bool(reply, err)
		return b
	}
}

//获取键的失效剩余时间
func (r *RedisModel) GetTimeout(key string) int {
	if key == "" {
		return 0
	}
	var err error
	var reply interface{}
	reply, err = r.Query("TTL", key)
	if err != nil {
		return 0
	} else {
		var time int
		time, err = redis.Int(reply, err)
		if err != nil || time < 0 {
			return 0
		} else {
			return time
		}

	}
}

//执行Redis命令-包含查询和修改操作（一般用主库方法）
//@param commandStr Redis命令
//@param args 参数数组
func (r *RedisModel) Do(commandStr string, args ...interface{}) (reply interface{}, err error) {
	c := r.redisWriter.Get()
	defer c.Close()
	if c.Err() != nil { //判断链接是否丢失，丢失后重建
		r.redisWriter = r.Connect(r.writerSettings)
		c = r.redisWriter.Get()
		if c.Err() != nil { //重建后链接还是失败，返回错误
			return nil, c.Err()
		}
	}

	return c.Do(commandStr, args...)
}

//执行Redis命令-查询操作（get等操作）
//@param commandStr Redis命令
//@param args 参数数组
func (r *RedisModel) Query(commandStr string, args ...interface{}) (reply interface{}, err error) {
	c := r.redisReader.Get()
	defer c.Close()
	if c.Err() != nil { //判断链接是否丢失，丢失后重建
		r.redisReader = r.Connect(r.readerSettings)
		c = r.redisReader.Get()
		if c.Err() != nil { //重建后链接还是失败，返回错误
			return nil, c.Err()
		}
	}
	return c.Do(commandStr, args...)
}

//获取redis链接
func (r *RedisModel) getConn(pool *redis.Pool, settings *RedisSettings) (redis.Conn, error) {
	c := pool.Get()
	if c.Err() != nil { //判断链接是否丢失，丢失后重建
		pool = r.Connect(settings)
		c = pool.Get()
		if c.Err() != nil { //重建后链接还是失败，返回错误
			return nil, c.Err()
		}
	}
	return c, nil
}

//redis统计结果
func (r *RedisModel) Stat() map[string]string {
	var res map[string]string = make(map[string]string, 0)
	reply, err := r.Do("INFO")
	if err == nil {
		fmt.Printf("%v", reply)
	}
	return res
}

//将数据转换为int
func (r *RedisModel) Int(i interface{}, err error) (int, error) {
	return redis.Int(i, err)
}

//将数据转换为int64
func (r *RedisModel) Int64(i interface{}, err error) (int64, error) {
	return redis.Int64(i, err)
}

//将数据转换为bool
func (r *RedisModel) Bool(i interface{}, err error) (bool, error) {
	return redis.Bool(i, err)
}

//将数据转换为string
func (r *RedisModel) String(i interface{}, err error) (string, error) {
	return redis.String(i, err)
}

//将数据转换为map[string]string
func (r *RedisModel) StringMap(i interface{}, err error) (map[string]string, error) {
	return redis.StringMap(i, err)
}
