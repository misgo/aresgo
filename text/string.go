/*
	字符串操作包，针对一些常用的字符串处理且Go本身不提供的应用封装了一些方法
	字符串的处理，[]byte操作要比string快
	使用方法：import ("github.com/misgo/aresgo/text")
	func main(){
		t = Text.FirstCharToUpper("abcd")
		fmt.Println(t)
	}
	@author : hyperion
	@since  : 2016-12-26
	@version: 1.0
*/
package Text

import (
	"bytes"
	"strconv"
	//	"strings"
)

type (
	StringBuilder struct {
		buffer *bytes.Buffer
	}
)

//高效拼接字符串-----start---
//经网上测试：拼接字符使用buffer，100W字符大约耗时77ms;拼接字符使用+=，10W字符大约耗时3312ms

//拼接字符串
func SpliceString(chars ...string) string {
	sb := stringBuilderInit(chars)
	return sb.ToString()
}

//拼接[]byte型字符串
func SpliceByte(bs ...[]byte) []byte {
	newByte := []byte("")
	if len(bs) > 0 {
		buffer := bytes.NewBuffer(newByte)
		for _, v := range bs {
			buffer.Write(v)
		}
		return buffer.Bytes()
	} else {
		return newByte
	}

}

//初始化新的字符串构造器StringBuilder
func NewString(chars ...string) *StringBuilder {
	return stringBuilderInit(chars)
}

//初始化
func stringBuilderInit(chars []string) *StringBuilder {
	if len(chars) > 0 {
		firstChar := chars[0]
		buffer := &StringBuilder{buffer: bytes.NewBuffer([]byte(firstChar))}
		for i := 1; i < len(chars); i++ {
			buffer.Append(chars[i])
		}
		return buffer
	} else {
		return &StringBuilder{
			buffer: bytes.NewBuffer([]byte("")),
		}
	}
}

//向字符串末尾添加字符串
func (sb *StringBuilder) Append(str string) int {
	len, _ := sb.buffer.WriteString(str)
	return len
}

//向末尾添加bytes
func (sb *StringBuilder) AppendBytes(b []byte) int {
	len, _ := sb.buffer.Write(b)
	return len
}

//获取拼接的字符串
func (sb *StringBuilder) ToString() string {
	return sb.buffer.String()
}

//获取拼接的字符串的Bytes
func (sb *StringBuilder) ToBytes() []byte {
	return sb.buffer.Bytes()
}

//高效拼接字符串-----end---

//将interface{}转换为[]byte
func GetBytes(i interface{}) []byte {
	var v []byte
	switch val := i.(type) {
	case string:
		v = []byte(val)
	case []byte:
		v = val
	case int:
		v = []byte(Int64ToString(int64(val)))
	case int32:
		v = []byte(Int64ToString(int64(val)))
	case int64:
		v = []byte(Int64ToString(val))
	case float64:
		v = []byte(strconv.FormatFloat(val, 'f', -1, 64))
	case float32:
		v = []byte(strconv.FormatFloat(float64(val), 'f', -1, 64))
	case bool:
		v = []byte(strconv.FormatBool(val))
	default:
		v = []byte("")
	}
	return v
}

//首字母大写其他转换成小写
func FirstCharToUpper(str string) string {
	strByte := []byte(str)
	strRes := FirstCharToUpperBytes(strByte)
	return string(strRes)
}

//首字母大写其他转换成小写([]byte模式)
func FirstCharToUpperBytes(strByte []byte) []byte {
	if len(strByte) > 0 {
		firstChar := bytes.ToUpper(strByte[:1])
		otherChar := bytes.ToLower(strByte[1:])
		strBytes := bytes.NewBuffer(firstChar)
		strBytes.Write(otherChar)
		return strBytes.Bytes()
	} else {
		emptyByte := make([]byte, 0)
		return bytes.NewBuffer(emptyByte).Bytes()
	}

}

/*
  根据开始和结束字符来截取字符串（截取开始与结束字符中间的部分，不包括开始与结束字符）
  @param strByte 待处理的字节数组
  @param startChar 截取的开始字符组
  @param endChar 截取的结束字符组(不包含结束字符)
  @return 新字节组
*/
func SubStrBytes(strByte []byte, startChar []byte, endChar []byte) []byte {
	var strLen int = len(strByte)
	var startIndex int = 0
	var endIndex int = strLen
	startCharLen := len(startChar)
	if startCharLen > 0 {
		startIndex = bytes.Index(strByte, startChar)
		if startIndex < 0 {
			startIndex = 0
		} else {
			startIndex = startIndex + startCharLen
			if startIndex > strLen {
				startIndex = strLen
			}
		}
	}
	if len(endChar) > 0 {
		if endIndex = bytes.LastIndex(strByte, endChar); endIndex < 0 {
			endIndex = strLen
		}
	}
	if startIndex > endIndex {
		startIndex = endIndex
	}
	return strByte[startIndex:endIndex]
}

/**
  截取字符串(截取开始于结束字符中间的部分，不包括开始与结束字符)
  @param str 待处理的字符串
  @param cutstr 截取字符串标识（多参数，1个参数：开始字符；2个参数：第一个是开始字符，第二个是结束字符）
  @return 新字符
*/
func SubStr(str string, cutstr ...string) string {
	if str == "" {
		return ""
	}
	var startChar string
	var endChar string
	if len(cutstr) > 0 {
		for i := 0; i < len(cutstr); i++ {
			if i == 0 {
				startChar = cutstr[i]
			} else if i == 1 {
				endChar = cutstr[i]
			}
		}
	}
	strBytes := SubStrBytes([]byte(str), []byte(startChar), []byte(endChar))

	return string(strBytes)
}

//根据索引截取字符串
func CutStr(str string, start int, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0
	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}
	return string(rs[start:end])
}

//string转换为int32
func StringToInt32(str string) int32 {
	i, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0
	}
	return int32(i)
}

//string转换为int
func StringToInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return i
}

//int64转换为字符串
func Int64ToString(n int64) string {
	buf := [11]byte{}
	pos := len(buf)
	signed := n < 0
	if signed {
		n = -n
	}
	for {
		pos--
		buf[pos], n = '0'+byte(n%10), n/10
		if n == 0 {
			if signed {
				pos--
				buf[pos] = '-'
			}
			return string(buf[pos:])
		}
	}
}
