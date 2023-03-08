package data

import (
	"fmt"
	"sync"

	"github.com/ohler55/ojg/jp"
)

type Data struct {
	sync.RWMutex
	Value any
}

func (d *Data) Write(val any) {
	d.Lock()
	defer d.Unlock()
	d.Value = val
}

func (d *Data) Get() any {
	d.RLock()
	defer d.RUnlock()
	return d.Value
}

func (d *Data) GetObject(path string) any {
	dp := jp.MustParseString(path)
	result := dp.Get(d.Get())
	if len(result) > 0 {
		return result[0]
	}
	return nil
}

func (d *Data) GetArray(path string) []any {
	dp := jp.MustParseString(path)
	return dp.Get(d.Get())
}

func (d *Data) GetInt64(path string, errors *Errors) int64 {
	val, ok := d.GetObject(path).(int64)
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q", path)))
		}
		return 0
	}
	return val
}

func (d *Data) GetFloat64(path string, errors *Errors) float64 {
	val, ok := d.GetObject(path).(float64)
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q", path)))
		}
		return 0
	}
	return val
}

func (d *Data) GetString(path string, errors *Errors) string {
	val, ok := d.GetObject(path).(string)
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q", path)))
		}
		return ""
	}
	return val
}

func (d *Data) GetBool(path string, errors *Errors) *bool {
	val, ok := d.GetObject(path).(bool)
	if !ok {
		if errors != nil {
			*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q", path)))
		}
		return nil
	}
	return &val
}
