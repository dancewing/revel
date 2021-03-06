// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orm

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// StrTo is the target string
type StrTo string

// Set string
func (f *StrTo) Set(v string) {
	if v != "" {
		*f = StrTo(v)
	} else {
		f.Clear()
	}
}

// Clear string
func (f *StrTo) Clear() {
	*f = StrTo(0x1E)
}

// Exist check string exist
func (f StrTo) Exist() bool {
	return string(f) != string(0x1E)
}

// Bool string to bool
func (f StrTo) Bool() (bool, error) {
	return strconv.ParseBool(f.String())
}

// Float32 string to float32
func (f StrTo) Float32() (float32, error) {
	v, err := strconv.ParseFloat(f.String(), 32)
	return float32(v), err
}

// Float64 string to float64
func (f StrTo) Float64() (float64, error) {
	return strconv.ParseFloat(f.String(), 64)
}

// Int string to int
func (f StrTo) Int() (int, error) {
	v, err := strconv.ParseInt(f.String(), 10, 32)
	return int(v), err
}

// Int8 string to int8
func (f StrTo) Int8() (int8, error) {
	v, err := strconv.ParseInt(f.String(), 10, 8)
	return int8(v), err
}

// Int16 string to int16
func (f StrTo) Int16() (int16, error) {
	v, err := strconv.ParseInt(f.String(), 10, 16)
	return int16(v), err
}

// Int32 string to int32
func (f StrTo) Int32() (int32, error) {
	v, err := strconv.ParseInt(f.String(), 10, 32)
	return int32(v), err
}

// Int64 string to int64
func (f StrTo) Int64() (int64, error) {
	v, err := strconv.ParseInt(f.String(), 10, 64)
	if err != nil {
		i := new(big.Int)
		ni, ok := i.SetString(f.String(), 10) // octal
		if !ok {
			return v, err
		}
		return ni.Int64(), nil
	}
	return v, err
}

// Uint string to uint
func (f StrTo) Uint() (uint, error) {
	v, err := strconv.ParseUint(f.String(), 10, 32)
	return uint(v), err
}

// Uint8 string to uint8
func (f StrTo) Uint8() (uint8, error) {
	v, err := strconv.ParseUint(f.String(), 10, 8)
	return uint8(v), err
}

// Uint16 string to uint16
func (f StrTo) Uint16() (uint16, error) {
	v, err := strconv.ParseUint(f.String(), 10, 16)
	return uint16(v), err
}

// Uint32 string to uint31
func (f StrTo) Uint32() (uint32, error) {
	v, err := strconv.ParseUint(f.String(), 10, 32)
	return uint32(v), err
}

// Uint64 string to uint64
func (f StrTo) Uint64() (uint64, error) {
	v, err := strconv.ParseUint(f.String(), 10, 64)
	if err != nil {
		i := new(big.Int)
		ni, ok := i.SetString(f.String(), 10)
		if !ok {
			return v, err
		}
		return ni.Uint64(), nil
	}
	return v, err
}

// String string to string
func (f StrTo) String() string {
	if f.Exist() {
		return string(f)
	}
	return ""
}

// ToStr interface to string
func ToStr(value interface{}, args ...int) (s string) {
	switch v := value.(type) {
	case bool:
		s = strconv.FormatBool(v)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', argInt(args).Get(0, -1), argInt(args).Get(1, 32))
	case float64:
		s = strconv.FormatFloat(v, 'f', argInt(args).Get(0, -1), argInt(args).Get(1, 64))
	case int:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int8:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int16:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int32:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int64:
		s = strconv.FormatInt(v, argInt(args).Get(0, 10))
	case uint:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint8:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint16:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint32:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint64:
		s = strconv.FormatUint(v, argInt(args).Get(0, 10))
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		s = fmt.Sprintf("%v", v)
	}
	return s
}

// ToInt64 interface to int64
func ToInt64(value interface{}) (d int64) {
	val := reflect.ValueOf(value)
	switch value.(type) {
	case int, int8, int16, int32, int64:
		d = val.Int()
	case uint, uint8, uint16, uint32, uint64:
		d = int64(val.Uint())
	default:
		panic(fmt.Errorf("ToInt64 need numeric not `%T`", value))
	}
	return
}

// snake string, XxYy to xx_yy , XxYY to xx_yy
func snakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 {
			t := s[i-1]
			if (isCaptial(d)) && !isCaptial(t) {
				data = append(data, '_')
			}
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

func isCaptial(d byte) bool {
	if d >= 'A' && d <= 'Z' {
		return true
	}
	return false
}

// camel string, xx_yy to XxYy
func camelString(s string) string {
	data := make([]byte, 0, len(s))
	flag, num := true, len(s)-1
	for i := 0; i <= num; i++ {
		d := s[i]
		if d == '_' {
			flag = true
			continue
		} else if flag {
			if d >= 'a' && d <= 'z' {
				d = d - 32
			}
			flag = false
		}
		data = append(data, d)
	}
	return string(data[:])
}

type argString []string

// get string by index from string slice
func (a argString) Get(i int, args ...string) (r string) {
	if i >= 0 && i < len(a) {
		r = a[i]
	} else if len(args) > 0 {
		r = args[0]
	}
	return
}

type argInt []int

// get int by index from int slice
func (a argInt) Get(i int, args ...int) (r int) {
	if i >= 0 && i < len(a) {
		r = a[i]
	}
	if len(args) > 0 {
		r = args[0]
	}
	return
}

// parse time to string with location
func timeParse(dateString, format string) (time.Time, error) {
	tp, err := time.ParseInLocation(format, dateString, DefaultTimeLoc)
	return tp, err
}

// get pointer indirect type
func indirectType(v reflect.Type) reflect.Type {
	switch v.Kind() {
	case reflect.Ptr:
		return indirectType(v.Elem())
	default:
		return v
	}
}

// get fields description as flatted string.
func getFlatParams(fi *fieldInfo, args []interface{}, tz *time.Location) (params []interface{}) {

outFor:
	for _, arg := range args {
		val := reflect.ValueOf(arg)

		if arg == nil {
			params = append(params, arg)
			continue
		}

		kind := val.Kind()
		if kind == reflect.Ptr {
			val = val.Elem()
			kind = val.Kind()
			arg = val.Interface()
		}

		switch kind {
		case reflect.String:
			v := val.String()
			if fi != nil {
				if fi.fieldType == TypeTimeField || fi.fieldType == TypeDateField || fi.fieldType == TypeDateTimeField {
					var t time.Time
					var err error
					if len(v) >= 19 {
						s := v[:19]
						t, err = time.ParseInLocation(formatDateTime, s, DefaultTimeLoc)
					} else if len(v) >= 10 {
						s := v
						if len(v) > 10 {
							s = v[:10]
						}
						t, err = time.ParseInLocation(formatDate, s, tz)
					} else {
						s := v
						if len(s) > 8 {
							s = v[:8]
						}
						t, err = time.ParseInLocation(formatTime, s, tz)
					}
					if err == nil {
						if fi.fieldType == TypeDateField {
							v = t.In(tz).Format(formatDate)
						} else if fi.fieldType == TypeDateTimeField {
							v = t.In(tz).Format(formatDateTime)
						} else {
							v = t.In(tz).Format(formatTime)
						}
					}
				}
			}
			arg = v
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			arg = val.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			arg = val.Uint()
		case reflect.Float32:
			arg, _ = StrTo(ToStr(arg)).Float64()
		case reflect.Float64:
			arg = val.Float()
		case reflect.Bool:
			arg = val.Bool()
		case reflect.Slice, reflect.Array:
			if _, ok := arg.([]byte); ok {
				continue outFor
			}

			var args []interface{}
			for i := 0; i < val.Len(); i++ {
				v := val.Index(i)

				var vu interface{}
				if v.CanInterface() {
					vu = v.Interface()
				}

				if vu == nil {
					continue
				}

				args = append(args, vu)
			}

			if len(args) > 0 {
				p := getFlatParams(fi, args, tz)
				params = append(params, p...)
			}
			continue outFor
		case reflect.Struct:
			if v, ok := arg.(time.Time); ok {
				if fi != nil && fi.fieldType == TypeDateField {
					arg = v.In(tz).Format(formatDate)
				} else if fi != nil && fi.fieldType == TypeDateTimeField {
					arg = v.In(tz).Format(formatDateTime)
				} else if fi != nil && fi.fieldType == TypeTimeField {
					arg = v.In(tz).Format(formatTime)
				} else {
					arg = v.In(tz).Format(formatDateTime)
				}
			} else {
				typ := val.Type()
				name := getFullName(typ)
				var value interface{}
				if mmi, err := Database().Get().TableFor(typ, true); err != nil {
					if _, vu, exist := getExistPk(mmi, val); exist {
						value = vu
					}
				}

				arg = value

				if arg == nil {
					panic(fmt.Errorf("need a valid args value, unknown table or value `%s`", name))
				}
			}
		}

		params = append(params, arg)
	}
	return
}

// get pk column info.
func getExistPk(mi *modelInfo, ind reflect.Value) (column string, value interface{}, exist bool) {

	if len(mi.fields.keys) > 1 {
		panic(fmt.Errorf("only one primary key can be set in %s", mi.name))
	}

	fi := mi.fields.GetOnePrimaryKey()

	v := ind.FieldByIndex(fi.fieldIndex)
	if fi.fieldType&IsPositiveIntegerField > 0 {
		vu := v.Uint()
		exist = vu > 0
		value = vu
	} else if fi.fieldType&IsIntegerField > 0 {
		vu := v.Int()
		exist = true
		value = vu
	} else if fi.fieldType&IsRelField > 0 {
		_, value, exist = getExistPk(fi.relModelInfo, reflect.Indirect(v))
	} else {
		vu := v.String()
		exist = vu != ""
		value = vu
	}

	column = fi.name
	return
}

func getFieldValue(m interface{}, field string) (arg interface{}) {
	e := reflect.ValueOf(m)
	rk := e.Kind()

	if rk == reflect.Ptr {
		e = e.Elem()
	}

	val := e.FieldByName(field)

	//	val := reflect.ValueOf(m)

	kind := val.Kind()
	if kind == reflect.Ptr {
		val = val.Elem()
		kind = val.Kind()
		arg = val.Interface()
	}

	switch kind {
	case reflect.String:
		v := val.String()
		arg = v
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		arg = val.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		arg = val.Uint()
	case reflect.Float32:
		arg, _ = StrTo(ToStr(arg)).Float64()
	case reflect.Float64:
		arg = val.Float()
	case reflect.Bool:
		arg = val.Bool()
	case reflect.Slice, reflect.Array:
		if _, ok := arg.([]byte); ok {
			//continue outFor
		}

		var args []interface{}
		for i := 0; i < val.Len(); i++ {
			v := val.Index(i)

			var vu interface{}
			if v.CanInterface() {
				vu = v.Interface()
			}

			if vu == nil {
				continue
			}

			args = append(args, vu)
		}

		// if len(args) > 0 {
		// 	p := getFlatParams(fi, args, tz)
		// 	params = append(params, p...)
		// }
		// continue outFor
	case reflect.Struct:
		if _, ok := arg.(time.Time); ok {

		} else {
			typ := val.Type()
			name := getFullName(typ)
			var value interface{}
			if mmi, err := Database().Get().TableFor(typ, true); err != nil {
				if _, vu, exist := getExistPk(mmi, val); exist {
					value = vu
				}
			}

			arg = value

			if arg == nil {
				panic(fmt.Errorf("need a valid args value, unknown table or value `%s`", name))
			}
		}
	}

	return

}
