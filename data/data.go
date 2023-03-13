package data

import (
	"fmt"
	"sync"

	"github.com/ohler55/ojg/jp"
)

type Data struct {
	sync.RWMutex
	value DataValue
}

type DataValue struct {
	val any
}

func NewDataValue(val any) DataValue {
	return DataValue{val}
}

func (d *Data) Write(val any) {
	d.Lock()
	defer d.Unlock()
	d.value.val = val
}

func (d *Data) Get() DataValue {
	d.RLock()
	defer d.RUnlock()
	return d.value
}

func (d *DataValue) GetObject(path string) any {
	dp := jp.MustParseString(path)
	result := dp.Get(d.val)
	if len(result) > 0 {
		return result[0]
	}
	return nil
}

func (d *DataValue) GetArray(path string) []any {
	dp := jp.MustParseString(path)
	return dp.Get(d.val)
}

func (d *DataValue) GetInt64(path string, errors *Errors) int64 {
	val, ok := d.GetObject(path).(int64)
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q, typeof val: %T", path, d.GetObject(path))))
		}
		return 0
	}
	return val
}

func (d *DataValue) GetFloat64(path string, errors *Errors) float64 {
	val, ok := d.GetObject(path).(float64)
	if !ok {
		// check if we can get it as int64 instead
		var iv int64
		iv, ok = d.GetObject(path).(int64)
		if ok {
			val = float64(iv)
		}
	}
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q, typeof val: %T", path, d.GetObject(path))))
		}
		return 0
	}
	return val
}

func (d *DataValue) GetString(path string, errors *Errors) string {
	val, ok := d.GetObject(path).(string)
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q, typeof val: %T", path, d.GetObject(path))))
		}
		return ""
	}
	return val
}

func (d *DataValue) GetBool(path string, errors *Errors) *bool {
	val, ok := d.GetObject(path).(bool)
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q, typeof val: %T", path, d.GetObject(path))))
		}
		return nil
	}
	return &val
}
