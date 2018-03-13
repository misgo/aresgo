/*
	验证库
	@author : hyperion
	@since  : 2018-3-13
	@version: 1.0
*/
package Text

import (
	"regexp"
)

type Error struct {
	Messsage, Field string
	Value           interface{}
}

type Validation struct {
	Errors    []*Error
	ErrorsMap map[string]*Error
}
