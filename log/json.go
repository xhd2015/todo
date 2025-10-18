package log

import (
	"encoding/json"
	"fmt"
)

func JSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	j, ok := v.(JSONValue)
	if ok {
		return j
	}
	return JSONValue{Value: v}
}

type JSONValue struct {
	Value interface{}
}

func (c JSONValue) String() string {
	b, err := json.Marshal(c.Value)
	if err != nil {
		return fmt.Sprintf("json.Marshal error: %v", err)
	}
	return string(b)
}
