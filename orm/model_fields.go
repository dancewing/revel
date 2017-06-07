// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// package orm provides a simple way to marshal Go structs to and from
// SQL databases.  It uses the database/sql package, and should work with any
// compliant database/sql driver.
//
// Source code and project home:
// https://github.com/dancewing/revel/orm

package orm

import (
	"reflect"
	"strings"
)

// field info collection
type fields struct {
	//pk            *fieldInfo
	keys    map[string]*fieldInfo
	indexes []*IndexMap

	columns       map[string]*fieldInfo
	fields        map[string]*fieldInfo
	fieldsLow     map[string]*fieldInfo
	fieldsByType  map[int][]*fieldInfo
	fieldsRel     []*fieldInfo
	fieldsReverse []*fieldInfo
	fieldsDB      []*fieldInfo
	rels          []*fieldInfo
	orders        []string
	dbcols        []string
}

// add field info
func (f *fields) Add(fi *fieldInfo) (added bool) {
	if f.fields[fi.name] == nil && f.columns[fi.column] == nil {
		f.columns[fi.column] = fi
		f.fields[fi.name] = fi
		f.fieldsLow[strings.ToLower(fi.name)] = fi
	} else {
		return
	}
	if _, ok := f.fieldsByType[fi.fieldType]; !ok {
		f.fieldsByType[fi.fieldType] = make([]*fieldInfo, 0)
	}
	f.fieldsByType[fi.fieldType] = append(f.fieldsByType[fi.fieldType], fi)
	f.orders = append(f.orders, fi.column)
	if fi.dbcol {
		f.dbcols = append(f.dbcols, fi.column)
		f.fieldsDB = append(f.fieldsDB, fi)
	}
	if fi.rel {
		f.fieldsRel = append(f.fieldsRel, fi)
	}
	if fi.reverse {
		f.fieldsReverse = append(f.fieldsReverse, fi)
	}
	return true
}

// get field info by name
func (f *fields) GetByName(name string) *fieldInfo {
	return f.fields[name]
}

// get field info by name
func (f *fields) GetOnePrimaryKey() *fieldInfo {
	if len(f.keys) > 0 {
		i := 0
		for _, f := range f.keys {
			if i == 0 {
				return f
			}
			i++
		}
	}
	return nil
}

// get field info by column name
func (f *fields) GetByColumn(column string) *fieldInfo {
	return f.columns[column]
}

// get field info by string, name is prior
func (f *fields) GetByAny(name string) (*fieldInfo, bool) {
	if fi, ok := f.fields[name]; ok {
		return fi, ok
	}
	if fi, ok := f.fieldsLow[strings.ToLower(name)]; ok {
		return fi, ok
	}
	if fi, ok := f.columns[name]; ok {
		return fi, ok
	}
	return nil, false
}

func (f *fields) GetByIndex(index int) *fieldInfo {
	i := 0
	for _, f := range f.columns {
		if index == i {
			return f
		}
		i++
	}
	return nil
}

// create new field info collection
func newFields() *fields {
	f := new(fields)
	f.keys = make(map[string]*fieldInfo)
	f.fields = make(map[string]*fieldInfo)
	f.fieldsLow = make(map[string]*fieldInfo)
	f.columns = make(map[string]*fieldInfo)
	f.fieldsByType = make(map[int][]*fieldInfo)
	return f
}

// fieldInfo represents a mapping between a Go struct field and a single
// column in a table.
// Unique and MaxSize only inform the
// CreateTables() function and are not used by Insert/Update/Delete/Get.
type fieldInfo struct {
	// Column name in db table
	//ColumnName string

	// If true, this column is skipped in generated SQL statements
	transient bool

	// If true, " unique" is added to create table statements.
	// Not used elsewhere
	//Unique bool

	// Query used for getting generated id after insert
	GeneratedIdQuery string

	// Passed to Dialect.ToSqlType() to assist in informing the
	// correct column type to map to in CreateTables()
	//MaxSize int

	DefaultValue string

	//fieldName string
	gotype reflect.Type
	//isAutoIncr bool
	isNotNull bool

	mi                  *modelInfo
	fieldIndex          []int
	fieldType           int
	dbcol               bool // if RelType is RelManyToMany, RelReverseMany, RelReverseOne value will be false
	inModel             bool
	name                string // fieldName
	fullName            string
	column              string // column name
	addrValue           reflect.Value
	sf                  reflect.StructField
	auto                bool
	pk                  bool //PK
	null                bool //is null
	index               bool
	unique              bool
	colDefault          bool  // whether has default tag
	initial             StrTo // store the default value
	size                int
	toText              bool
	autoNow             bool
	autoNowAdd          bool
	rel                 bool // if type equal to RelForeignKey, RelOneToOne, RelManyToMany then true
	reverse             bool
	reverseField        string
	reverseFieldInfo    *fieldInfo
	reverseFieldInfoTwo *fieldInfo
	reverseFieldInfoM2M *fieldInfo
	relTable            string
	relThrough          string
	relThroughModelInfo *modelInfo
	relModelInfo        *modelInfo
	digits              int
	decimals            int
	isFielder           bool // implement Fielder interface
	onDelete            string
}

// Rename allows you to specify the column name in the table
//
// Example:  table.ColMap("Updated").Rename("date_updated")
//
func (c *fieldInfo) Rename(colname string) *fieldInfo {
	c.column = colname
	return c
}

// SetTransient allows you to mark the column as transient. If true
// this column will be skipped when SQL statements are generated
func (c *fieldInfo) SetTransient(b bool) *fieldInfo {
	c.transient = b
	return c
}

// SetUnique adds "unique" to the create table statements for this
// column, if b is true.
func (c *fieldInfo) SetUnique(b bool) *fieldInfo {
	c.unique = b
	return c
}

// SetNotNull adds "not null" to the create table statements for this
// column, if nn is true.
func (c *fieldInfo) SetNotNull(nn bool) *fieldInfo {
	c.isNotNull = nn
	return c
}

// SetMaxSize specifies the max length of values of this column. This is
// passed to the dialect.ToSqlType() function, which can use the value
// to alter the generated type for "create table" statements
func (c *fieldInfo) SetMaxSize(size int) *fieldInfo {
	c.size = size
	return c
}
