/*
	数据库操作类库，支持CURD方法，支持对象映射（struct<-->map）
	@author : hyperion
	@since  : 2016-12-29
	@version: 1.0.1
*/
package Db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "github.com/misgo/aresgo/data/mysql"
	"github.com/misgo/aresgo/text"
)

var (
	IsDebug bool = false
)

const (
	MethodInsert = "INSERT"
	MethodUpdate = "UPDATE"
	MethodDelete = "DELETE"
	MethodSelect = "SELECT"
)

type (
	DbModel struct {
		dbReader        *sql.DB
		dbWriter        *sql.DB
		TableName       string
		RowsNum         int //行数
		Offset          int
		WhereStr        string
		Param           []interface{}
		Order           string
		Column          string
		PrimaryKeys     map[string]interface{}
		Join            string
		GroupByStr      string
		HavingStr       string
		QuoteIdentifier string
		ParamIdentifier string
		ParamIteration  int
		FieldMap        map[string]interface{}
		fieldStructMap  map[string]string
		EnableTbPre     bool
		TbPre           string
	}

	DbSettings struct {
		Ip          string
		Port        string
		User        string
		Password    string
		Charset     string
		DefaultDb   string
		EnableTbPre bool
		TbPre       string
	}
)

//创建数据库对象
func NewDb(driver string, config map[string]*DbSettings) *DbModel {
	db := &DbModel{}
	db.ResetDbModel()
	//拼装SQL语句
	if dbWriterConfig, ok := config["master"]; ok {
		dbWriterConfigStr := Text.SpliceString(dbWriterConfig.User, ":", dbWriterConfig.Password,
			"@tcp(", dbWriterConfig.Ip, ":", dbWriterConfig.Port, ")/", dbWriterConfig.DefaultDb,
			"?charset=", dbWriterConfig.Charset)
		db.dbWriter = Init(driver, dbWriterConfigStr)
		db.EnableTbPre = dbWriterConfig.EnableTbPre
		db.TbPre = dbWriterConfig.TbPre
		//		fmt.Println(dbWriterConfigStr)
	} else {
		errMsg := fmt.Sprintf("无法连接到主数据库[Ip:%s;port:%s]", dbWriterConfig.Ip, dbWriterConfig.Port)
		panic(errMsg)
	}
	if dbReaderConfig, ok := config["slave"]; ok {
		dbReaderConfigStr := Text.SpliceString(dbReaderConfig.User, ":", dbReaderConfig.Password,
			"@tcp(", dbReaderConfig.Ip, ":", dbReaderConfig.Port, ")/", dbReaderConfig.DefaultDb,
			"?charset=", dbReaderConfig.Charset)
		db.dbReader = Init(driver, dbReaderConfigStr)
		//		fmt.Println(dbReaderConfigStr)
	} else {
		errMsg := fmt.Sprintf("无法连接到从数据库[Ip:%s;port:%s]", dbReaderConfig.Ip, dbReaderConfig.Port)
		panic(errMsg)
		//		fmt.Printf("%s", "没有找到从库")
	}
	return db
}

//初始化数据库
func Init(driver string, linkstr string) *sql.DB {
	db, err := sql.Open(driver, linkstr)
	checkErr(err)
	db.SetMaxOpenConns(2000) //设置最大打开的连接数，默认值为0表示不限制,可以避免并发太高导致连接mysql出现too many connections的错误
	db.SetMaxIdleConns(1000) //设置闲置的连接数,当开启的一个连接使用完成后可以放在池里等候下一次使用
	err = db.Ping()
	checkErr(err)
	return db
}

//数据库连接判断，连接未中断返回nil
func (m *DbModel) Ping() error {
	err := m.dbReader.Ping()
	if err == nil {
		err = m.dbWriter.Ping()
	}
	return err
}

//设定数据表名
func (m *DbModel) Table(tbname string) *DbModel {
	if m.EnableTbPre {
		m.TableName = fmt.Sprintf("%s%s", m.TbPre, tbname)
	} else {
		m.TableName = tbname
	}

	return m
}

//设置主键
func (m *DbModel) SetPK(pks ...string) *DbModel {
	for _, v := range pks {
		m.PrimaryKeys[v] = ""
	}
	return m
}

//设定选择的字段
func (m *DbModel) Field(fields ...string) *DbModel {
	m.Column = strings.Join(fields, ",")
	return m
}

//分页
func (m *DbModel) Limit(start int, size int) *DbModel {
	m.Offset = start
	m.RowsNum = size
	return m
}

//排序字符
func (m *DbModel) OrderBy(order ...string) *DbModel {
	m.Order = strings.Join(order, ",")
	return m
}

//分组
func (m *DbModel) GroupBy(groupstr ...string) *DbModel {
	m.GroupByStr = strings.Join(groupstr, ",")
	return m
}

//分组条件
func (m *DbModel) Having(havingstr ...string) *DbModel {
	m.HavingStr = strings.Join(havingstr, ",")
	return m
}

//查询条件
func (m *DbModel) Where(queryString string, args ...interface{}) *DbModel {
	if strings.Count(queryString, "?") != len(args) {
		err := fmt.Sprintf("查询条件[%s]与参数个数不对应", queryString)
		panic(err)
	}
	m.WhereStr = queryString
	m.Param = args
	return m
}

//从数据库中查询出列表并映射为一个struct列表
//@param structList 结构体对象数组
func (m *DbModel) FindList(structList interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(structList))
	rt := rv.Type().Elem()
	rvNew := reflect.New(rt)
	m.ConvertModelToMap(rvNew.Interface()) //将字段结构转换map
	res, err := m.Select()

	if err == nil && len(*res) > 0 { //有数据
		if rv.Kind() != reflect.Slice {
			return errors.New("不是对象指针")
		}
		for _, row := range *res { //遍历查询出来的数据map
			err := m.ConvertMapToModel(row, rvNew.Interface()) //将单条写进struct对象
			if err != nil {
				return err
			}
			rv.Set(reflect.Append(rv, reflect.Indirect(reflect.ValueOf(rvNew.Interface()))))
		}
		return nil
	} else {
		return errors.New("未能查找到数据")
	}
}

//从数据库中查询一条数据并映射到struct
//@param i struct对象
func (m *DbModel) Find(i interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(i))
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return errors.New("获取的数据类型必须为struct")
	}
	m.ConvertModelToMap(rv.Interface()) //将字段结构转换map
	res, err := m.Select()

	if err == nil && len(*res) > 0 {
		for _, row := range *res {
			err := m.ConvertMapToModel(row, i) //将单条写进struct对象
			if err != nil {
				return err
			}
			break
		}
		return nil
	} else {
		return errors.New("未能查找到数据")
	}

}

//根据主键查询数据
//@param i 查询出的Struct对象
//@param pkArgs 主键值（含多个）
func (m *DbModel) FindByPK(i interface{}, pkArgs ...interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(i))
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		return errors.New("获取的数据类型必须为struct")
	}
	m.ConvertModelToMap(rv.Interface()) //将字段结构转换map

	//构建主键查询条件
	var pkValLen int = len(pkArgs)
	var pkLen int = len(m.PrimaryKeys)
	var param []interface{}

	if pkValLen < 1 {
		panic("主键值不能为空")
	} else if pkValLen != pkLen {
		panic("主键与值不匹配")
	}

	sb := Text.NewString("1=1")

	for k, _ := range m.PrimaryKeys {
		sb.Append(" AND ")
		sb.Append(k)
		sb.Append(" = ?")
	}
	for _, v := range pkArgs {
		param = append(param, v)
	}

	m.WhereStr = sb.ToString()
	m.Param = param
	//查询数据
	res, err := m.Select()
	if err == nil && len(*res) > 0 {
		for _, row := range *res {
			err := m.ConvertMapToModel(row, i) //将单条写进struct对象
			if err != nil {
				return err
			}
			break
		}
		return nil
	} else {
		return errors.New("未能查找到数据")
	}

}

//用户CURD操作时,根据struct结构体查询出结果
func (m *DbModel) Select() (*[]map[string]string, error) {
	sql := Text.NewString("SELECT ")
	//column
	if m.Column != "" {
		sql.Append(m.Column)
	} else {
		sql.Append("*")
	}
	//table
	sql.Append(" FROM ")
	sql.Append(m.TableName)
	//where
	if m.WhereStr != "" {
		sql.Append(" WHERE ")
		sql.Append(m.WhereStr)
	}
	//group by
	if m.GroupByStr != "" {
		sql.Append(" GROUP BY ")
		sql.Append(m.GroupByStr)
		if m.HavingStr != "" {
			sql.Append(" HAVING ")
			sql.Append(m.HavingStr)
		}
	}
	//order by
	if m.Order != "" {
		sql.Append(" ORDER BY ")
		sql.Append(m.Order)
	}

	//limit 0,1
	if m.RowsNum > 0 {
		sql.Append(" Limit ")
		sql.Append(strconv.Itoa(m.Offset))
		sql.Append(",")
		sql.Append(strconv.Itoa(m.RowsNum))
	}
	//SQL调试
	//fmt.Printf("%s", sql.ToString())
	//	ret := make([]map[string]string, 0)
	//	return &ret, errors.New("debug")
	return m.Query(sql.ToString(), m.Param...)
}

//将数据库查询出的数据映射到struct
func (m *DbModel) ConvertMapToModel(s map[string]string, mStruct interface{}) error {
	model := reflect.Indirect(reflect.ValueOf(mStruct))
	modelType := model.Type()
	if model.Kind() != reflect.Struct {
		return errors.New("expected a pointer to a struct")
	}
	for i := 0; i < model.NumField(); i++ {
		fieldValue := model.Field(i)
		field := modelType.Field(i)
		m.convertToModelElem(fieldValue, field, s)
	}

	return nil
}

//转换为Struct对象的元素（单个元素值设置）
func (m *DbModel) convertToModelElem(fieldValue reflect.Value, field reflect.StructField, s map[string]string) error {
	var sKey string
	fieldTag := field.Tag.Get("field")
	if fieldTag != "" {
		sKey = fieldTag
	} else {
		sKey = field.Name
	}
	if field.Type.Kind() == reflect.Struct && field.Type.String() != "time.Time" {
		return errors.New("不支持除Time类型外的其他类型的转换")
	}
	if dbValue, ok := s[sKey]; ok {
		//值转换
		var newValue interface{}
		switch field.Type.Kind() {
		case reflect.String:
			newValue = dbValue
		case reflect.Bool:
			newValue = dbValue == "1"
		case reflect.Int:
			x, err := strconv.Atoi(dbValue)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Int，错误：", dbValue, err.Error()))
			}
			newValue = x
		case reflect.Int8:
			x, err := strconv.ParseInt(dbValue, 10, 8)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Int8，错误：", dbValue, err.Error()))
			}
			newValue = int8(x)
		case reflect.Int16:
			x, err := strconv.ParseInt(dbValue, 10, 16)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Int16，错误：", dbValue, err.Error()))
			}
			newValue = int16(x)
		case reflect.Int32:
			x, err := strconv.ParseInt(dbValue, 10, 32)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Int32，错误：", dbValue, err.Error()))
			}
			newValue = int32(x)
		case reflect.Int64:
			x, err := strconv.ParseInt(dbValue, 10, 64)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Int64，错误：", dbValue, err.Error()))
			}
			newValue = x
		case reflect.Float32:
			x, err := strconv.ParseFloat(dbValue, 32)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Float32，错误：", dbValue, err.Error()))
			}
			newValue = float32(x)
		case reflect.Float64:
			x, err := strconv.ParseFloat(dbValue, 64)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Float64，错误：", dbValue, err.Error()))
			}
			newValue = x
		case reflect.Uint8:
			x, err := strconv.ParseUint(dbValue, 10, 8)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Uint，错误：", dbValue, err.Error()))
			}
			newValue = uint8(x)
		case reflect.Uint16:
			x, err := strconv.ParseUint(dbValue, 10, 16)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Uint，错误：", dbValue, err.Error()))
			}
			newValue = uint16(x)
		case reflect.Uint32:
			x, err := strconv.ParseUint(dbValue, 10, 32)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Uint，错误：", dbValue, err.Error()))
			}
			newValue = uint32(x)
		case reflect.Uint64:
			x, err := strconv.ParseUint(dbValue, 10, 64)
			if err != nil {
				return errors.New(fmt.Sprintf("字段[%v]不能转换为Uint，错误：", dbValue, err.Error()))
			}
			newValue = x
		case reflect.Struct:
			x, err := time.Parse("2006-01-02 15:04:05", dbValue) //此处注意go日期转换只能与格式字符串保持一致，诸如2000/10/1 12:00:22这样的不识别

			if err != nil {
				x, err = time.Parse("2006-01-02 15:04:05.000 -0700", dbValue)

				if err != nil {
					intTime, err := strconv.ParseInt(dbValue, 10, 64)
					if err == nil {
						x = time.Unix(intTime, 0)
					} else {
						return errors.New("时间格式不支持: " + dbValue)
					}

				}
			}
			newValue = x
		default:
			return errors.New("未发现可以支持的类型: " + reflect.TypeOf(newValue).String())
		}
		//		fmt.Printf("%v:%v;type:%v\r\n", field.Name, newValue, reflect.TypeOf(newValue).String())
		fieldValue.Set(reflect.ValueOf(newValue)) //将字段写入struct
		return nil
	} else {
		return errors.New("未找到此字段值")
	}
}

//将struct对象转换为Map，获取Map中自定义标签属性
//field:数据库中字段名；key:主键是PK，其他是field，如果为notfield代表着个字段不是数据库字段值,auto代表此字段是数据库字段值但是属于系统生成的；table表名，取第一个定义的table
func (m *DbModel) ConvertModelToMap(s interface{}) *DbModel {
	if reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface()).Kind() == reflect.Slice {
		sliceValue := reflect.Indirect(reflect.ValueOf(s))

		sliceElementType := sliceValue.Type().Elem()

		for i := 0; i < sliceElementType.NumField(); i++ {
			field := sliceElementType.Field(i)
			m.setFieldMap(field, sliceValue)
		}
	} else {
		rt := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(s)).Interface())
		rv := reflect.Indirect(reflect.ValueOf(s))
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			m.setFieldMap(field, rv)

		}
	}
	return m
}

//获取struct的标签并设置DbModel.FieldMap及DbModel.TableName
func (m *DbModel) setFieldMap(field reflect.StructField, rv reflect.Value) {
	var tagField, tagTable, tagKey, tagIsAuto, tagType string
	tagField = field.Tag.Get("field")
	tagKey = strings.ToLower(field.Tag.Get("key"))
	tagTable = field.Tag.Get("table")
	tagType = field.Tag.Get("type")
	tagIsAuto = strings.Trim(field.Tag.Get("auto"), " ") //是否为数据库自动字段

	if tagKey != "notfield" { //不属于数据库字段不添加到字段值列表
		//构造字段表map
		var fmField string
		if tagField != "" {
			fmField = tagField
		} else {
			fmField = field.Name
		}
		val := rv.FieldByName(field.Name).Interface()

		if tagIsAuto != "1" { //字段赋值
			if field.Type.Kind() == reflect.Struct && field.Type.String() == "time.Time" { //time类型
				fieldTime := val.(time.Time)
				if !fieldTime.IsZero() {
					if tagType == "date" { //date方式
						m.FieldMap[fmField] = fieldTime.Format("2006-01-02")
					} else if tagType == "datetime" { //datetime方式
						m.FieldMap[fmField] = fieldTime.Format("2006-01-02 15:04:05")
					} else if tagType == "int" { //uinx时间戳方式
						m.FieldMap[fmField] = fieldTime.Unix()
					}
				}

			} else {
				m.FieldMap[fmField] = val
			}

		}

		//添加到主键列表
		if tagKey == "pk" {
			m.PrimaryKeys[fmField] = val
		}

	}

	//如果设置table标签用则采用table的值，如果已经执行过Table方法了，此标签失效
	if tagTable != "" && m.TableName == "" {
		if m.EnableTbPre {
			m.TableName = fmt.Sprintf("%s%s", m.TbPre, tagTable)
		} else {
			m.TableName = tagTable
		}
	}
}

//添加新的对象到数据库，可以将struct保存到数据库，字段不一致的通过struct tag来解决
func (m *DbModel) Add(s interface{}) (int64, error) {
	m.ConvertModelToMap(s)

	if len(m.FieldMap) > 0 {
		return m.Insert(m.FieldMap)
	}
	return -1, nil
}

//保存对象到数据库
func (m *DbModel) Save(s interface{}) (int64, error) {
	m.ConvertModelToMap(s)
	//有数据更新才调用更新方法
	if len(m.FieldMap) > 0 {
		//保存时如果未设置查询条件，则按照主键保存
		if m.WhereStr == "" {
			whereSb := Text.NewString("1=1")
			var values []interface{}
			for k, v := range m.PrimaryKeys {
				whereSb.Append(" AND ")
				whereSb.Append(k)
				whereSb.Append(" = ?")
				//添加查询args数组
				values = append(values, v)
			}
			m.WhereStr = whereSb.ToString()
			m.Param = values
		}
		return m.Update(m.FieldMap)
	}
	return -1, nil
}

//添加数据，用于CURD操作时的添加，通过构建map[string]interface{}添加数据
func (m *DbModel) Insert(fieldmap map[string]interface{}) (int64, error) {
	if m.TableName == "" || len(fieldmap) < 1 {
		panic("数据表名不能为空或字段列表不能为空")
	}
	var fields []string
	var placeholders []string
	var values []interface{}
	var whereStr string = ""
	for k, v := range fieldmap {
		fields = append(fields, k)
		placeholders = append(placeholders, "?")
		values = append(values, v)
	}
	sql := fmt.Sprintf("INSERT INTO %v (%v) VALUES (%v) %v", m.TableName, strings.Join(fields, ", "), strings.Join(placeholders, ", "), whereStr)
	//sql调试
	//fmt.Println(sql, "\r\n")
	//fmt.Println(values, "\r\n")
	//	return -1, nil
	return m.Execute(MethodInsert, sql, values...)
}

//更新数据，用户CURD操作时的更新，通过构建map[string]interface{}更新数据
func (m *DbModel) Update(fieldmap map[string]interface{}) (int64, error) {
	if m.TableName == "" || len(fieldmap) < 1 {
		panic("数据表名不能为空或字段列表不能为空")
	}
	var items []string
	var values []interface{}
	var whereStr string = ""
	for k, v := range fieldmap {
		item := fmt.Sprintf("%v = ?", k)
		values = append(values, v)
		items = append(items, item)
	}
	if m.WhereStr != "" {
		whereStr = fmt.Sprintf(" WHERE %v", m.WhereStr)
		if len(m.Param) > 0 {
			for _, v := range m.Param {
				values = append(values, v)
			}

		}
	}
	sql := fmt.Sprintf("UPDATE %v SET %v %v", m.TableName, strings.Join(items, ", "), whereStr)
	//sql调试
	//	fmt.Println(sql, "\r\n")
	//	fmt.Println(values, "\r\n")
	//	return -1, nil
	return m.Execute(MethodUpdate, sql, values...)
}

//删除,用户CURD操作时的删除
//@param pkArgs 对应的主键值，如果是多主键则此处值得个数为多个
func (m *DbModel) Delete(pkArgs ...interface{}) (int64, error) {
	if m.TableName == "" {
		panic("数据表名不能为空")
	}
	var pkValLen int = len(pkArgs)
	var pkLen int = len(m.PrimaryKeys)
	if pkValLen < 1 && m.WhereStr == "" {
		panic("删除条件不能为空，禁止全表删除")
	}
	if pkValLen > 0 && pkValLen != pkLen {
		panic("主键与值不匹配")
	}
	sb := Text.NewString(" WHERE ")
	if m.WhereStr != "" { //
		sb.Append(m.WhereStr)

	} else {
		sb.Append("1=1")
	}
	if pkValLen > 0 { //有主键删除方式
		for k, _ := range m.PrimaryKeys {
			sb.Append(" AND ")
			sb.Append(k)
			sb.Append(" = ?")
		}

		for _, v := range pkArgs {
			m.Param = append(m.Param, v)
		}
	}

	sql := fmt.Sprintf("DELETE FROM %v %v", m.TableName, sb.ToString())
	//sql调试
	//	fmt.Println(sql, "\r\n")
	//	fmt.Println(m.Param, "\r\n")
	//	return -1, nil
	return m.Execute(MethodDelete, sql, m.Param...)
}

//数据库修改操作（insert/update/delete）
func (m *DbModel) Execute(opt string, sqlstr string, args ...interface{}) (int64, error) {
	defer m.ResetDbModel()
	stmt, err := m.dbWriter.Prepare(sqlstr)
	checkErr(err)
	defer stmt.Close()
	res, err := stmt.Exec(args...)
	checkErr(err)
	var resnum int64
	if opt == MethodInsert {
		resnum, err = res.LastInsertId()
	} else {
		resnum, err = res.RowsAffected()
	}
	return resnum, err
}

//获取一行数据
func (m *DbModel) GetRow(sqlstr string, args ...interface{}) (map[string]string, error) {
	res, err := m.Query(sqlstr, args...)
	if len(*res) > 0 {
		for _, row := range *res {
			return row, err
			break
		}
	}
	return nil, err
}

//数据库查询操作（select）
func (m *DbModel) Query(sqlstr string, args ...interface{}) (*[]map[string]string, error) {
	defer m.ResetDbModel()
	sqlstr = checkSql(sqlstr)
	ret := make([]map[string]string, 0) //返回的结果集

	if m.dbReader == nil { //如果数据库实例未能获取到，返回空列表
		return &ret, nil
	}
	stmp, err := m.dbReader.Prepare(sqlstr)
	checkErr(err)
	defer stmp.Close()
	rows, err := stmp.Query(args...)
	checkErr(err)
	columns, err := rows.Columns()
	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))

	for i := range values {
		scanArgs[i] = &values[i]
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		checkErr(err)
		var val string
		vmap := make(map[string]string, len(scanArgs))
		for i, col := range values {
			//			fmt.Printf("type:%v\r\n", reflect.ValueOf(col).Kind())
			if col == nil {
				val = "NULL"
			} else {
				val = string(col)
			}
			vmap[columns[i]] = val
		}
		ret = append(ret, vmap)
	}
	return &ret, nil

}

func (m *DbModel) ResetDbModel() {
	m.Column = "*"
	m.TableName = ""
	m.GroupByStr = ""
	m.HavingStr = ""
	m.Join = ""
	m.RowsNum = 0
	m.Offset = 0
	m.Order = ""
	m.WhereStr = ""
	m.PrimaryKeys = make(map[string]interface{})
	m.FieldMap = make(map[string]interface{})
	m.fieldStructMap = make(map[string]string)
}

//sql语句检查
func checkSql(sqlstr string) string {
	return sqlstr
}

//检查错误
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func outputMsg(msg string) {
	panic(msg)
}
