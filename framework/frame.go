package frame

//结构体定义
type (
	App struct {
	}
)

//全局变量
var (
	Debug        bool   //是否为调试模式
	CacheMode    string //缓存模式
	CacheTimeout int64  //缓存失效时间
)
