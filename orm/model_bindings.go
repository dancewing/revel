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
	"bytes"
	"fmt"
	"reflect"
	"sync"
)

// CustomScanner binds a database column value to a Go type
type CustomScanner struct {
	// After a row is scanned, Holder will contain the value from the database column.
	// Initialize the CustomScanner with the concrete Go type you wish the database
	// driver to scan the raw column into.
	Holder interface{}
	// Target typically holds a pointer to the target struct field to bind the Holder
	// value to.
	Target interface{}
	// Binder is a custom function that converts the holder value to the target type
	// and sets target accordingly.  This function should return error if a problem
	// occurs converting the holder to the target.
	Binder func(holder interface{}, target interface{}) error
}

// Used to filter columns when selectively updating
type ColumnFilter func(*fieldInfo) bool

func acceptAllFilter(col *fieldInfo) bool {
	return true
}

// Bind is called automatically by gorp after Scan()
func (me CustomScanner) Bind() error {
	return me.Binder(me.Holder, me.Target)
}

type bindPlan struct {
	query             string
	argFields         []string
	keyFields         []string
	versField         string
	autoIncrIdx       int
	autoIncrFieldName string
	once              sync.Once
	paramValues       []interface{}
}

func (plan *bindPlan) createBindInstance(elem reflect.Value, conv TypeConverter) (bindInstance, error) {
	bi := bindInstance{query: plan.query, autoIncrIdx: plan.autoIncrIdx, autoIncrFieldName: plan.autoIncrFieldName, versField: plan.versField}
	if plan.versField != "" {
		bi.existingVersion = elem.FieldByName(plan.versField).Int()
	}

	var err error

	for i := 0; i < len(plan.argFields); i++ {
		k := plan.argFields[i]

		if k == versFieldConst {
			newVer := bi.existingVersion + 1
			bi.args = append(bi.args, newVer)
			if bi.existingVersion == 0 {
				elem.FieldByName(plan.versField).SetInt(int64(newVer))
			}
		} else {
			val := elem.FieldByName(k).Interface()
			if conv != nil {
				val, err = conv.ToDb(val)
				if err != nil {
					return bindInstance{}, err
				}
			}
			bi.args = append(bi.args, val)
		}
	}

	for i := 0; i < len(plan.keyFields); i++ {
		k := plan.keyFields[i]
		val := elem.FieldByName(k).Interface()
		if conv != nil {
			val, err = conv.ToDb(val)
			if err != nil {
				return bindInstance{}, err
			}
		}
		bi.keys = append(bi.keys, val)
	}

	return bi, nil
}

type bindInstance struct {
	query             string
	args              []interface{}
	keys              []interface{}
	existingVersion   int64
	versField         string
	autoIncrIdx       int
	autoIncrFieldName string
}

func (t *modelInfo) bindInsert(elem reflect.Value) (bindInstance, error) {
	plan := &t.insertPlan
	plan.once.Do(func() {
		plan.autoIncrIdx = -1

		s := bytes.Buffer{}
		s2 := bytes.Buffer{}
		s.WriteString(fmt.Sprintf("insert into %s (", Database().Get().Dialect.QuotedTableForQuery(t.schemaName, t.table)))

		x := 0
		first := true
		for _, col := range t.fields.columns {
			//col := t.Columns[y]
			if !(col.auto && Database().Get().Dialect.AutoIncrBindValue() == "") {

				if col.transient || col.fieldType == RelManyToMany || col.fieldType == RelReverseMany {

				} else {
					if !first {
						s.WriteString(",")
						s2.WriteString(",")
					}
					s.WriteString(Database().Get().Dialect.QuoteField(col.column))

					if col.auto {
						s2.WriteString(Database().Get().Dialect.AutoIncrBindValue())
						plan.autoIncrIdx = x
						plan.autoIncrFieldName = col.name
					} else {
						if col.DefaultValue == "" {
							s2.WriteString(Database().Get().Dialect.BindVar(x))
							if col == t.version {
								plan.versField = col.name
								plan.argFields = append(plan.argFields, versFieldConst)
							} else {

								//TODO
								if col.fieldType == RelManyToMany || col.fieldType == RelReverseMany {

								} else {
									plan.argFields = append(plan.argFields, col.name)
								}

							}
							x++
						} else {
							s2.WriteString(col.DefaultValue)
						}
					}
					first = false
				}

			} else {
				plan.autoIncrIdx = x
				plan.autoIncrFieldName = col.name
			}
			x++
		}
		s.WriteString(") values (")
		s.WriteString(s2.String())
		s.WriteString(")")
		if plan.autoIncrIdx > -1 {
			s.WriteString(Database().Get().Dialect.AutoIncrInsertSuffix(t.fields.GetByIndex(plan.autoIncrIdx)))
		}
		s.WriteString(Database().Get().Dialect.QuerySuffix())

		plan.query = s.String()
	})

	return plan.createBindInstance(elem, Database().Get().TypeConverter)
}

func (t *modelInfo) bindUpdate(elem reflect.Value, colFilter ColumnFilter) (bindInstance, error) {
	if colFilter == nil {
		colFilter = acceptAllFilter
	}

	plan := &t.updatePlan
	plan.once.Do(func() {
		s := bytes.Buffer{}
		s.WriteString(fmt.Sprintf("update %s set ", Database().Get().Dialect.QuotedTableForQuery(t.schemaName, t.table)))
		x := 0

		for _, col := range t.fields.columns {
			//col := t.Columns[y]
			if !col.auto && !col.transient && colFilter(col) {
				if x > 0 {
					s.WriteString(", ")
				}
				s.WriteString(Database().Get().Dialect.QuoteField(col.column))
				s.WriteString("=")
				s.WriteString(Database().Get().Dialect.BindVar(x))

				if col == t.version {
					plan.versField = col.name
					plan.argFields = append(plan.argFields, versFieldConst)
				} else {
					plan.argFields = append(plan.argFields, col.name)
				}
				x++
			}
		}

		s.WriteString(" where ")
		var y = 0
		for _, col := range t.fields.keys {
			//col := t.keys[y]
			if y > 0 {
				s.WriteString(" and ")
			}
			s.WriteString(Database().Get().Dialect.QuoteField(col.column))
			s.WriteString("=")

			s.WriteString(Database().Get().Dialect.BindVar(y))
			plan.argFields = append(plan.argFields, col.name)
			plan.keyFields = append(plan.keyFields, col.name)
			//x++
			y++
		}
		if plan.versField != "" {
			s.WriteString(" and ")
			s.WriteString(Database().Get().Dialect.QuoteField(t.version.column))
			s.WriteString("=")
			s.WriteString(Database().Get().Dialect.BindVar(x))
			plan.argFields = append(plan.argFields, plan.versField)
		}
		s.WriteString(Database().Get().Dialect.QuerySuffix())

		plan.query = s.String()
	})

	return plan.createBindInstance(elem, Database().Get().TypeConverter)
}

func (t *modelInfo) bindDelete(elem reflect.Value) (bindInstance, error) {
	plan := &t.deletePlan
	plan.once.Do(func() {
		s := bytes.Buffer{}
		s.WriteString(fmt.Sprintf("delete from %s", Database().Get().Dialect.QuotedTableForQuery(t.schemaName, t.table)))

		for _, col := range t.fields.columns {
			//col := t.Columns[y]
			if !col.transient {
				if col == t.version {
					plan.versField = col.name
				}
			}
		}

		s.WriteString(" where ")
		var x = 0
		for _, k := range t.fields.keys {
			//k := t.keys[x]
			if x > 0 {
				s.WriteString(" and ")
			}
			s.WriteString(Database().Get().Dialect.QuoteField(k.column))
			s.WriteString("=")
			s.WriteString(Database().Get().Dialect.BindVar(x))

			plan.keyFields = append(plan.keyFields, k.name)
			plan.argFields = append(plan.argFields, k.name)
			x++
		}
		if plan.versField != "" {
			s.WriteString(" and ")
			s.WriteString(Database().Get().Dialect.QuoteField(t.version.column))
			s.WriteString("=")
			s.WriteString(Database().Get().Dialect.BindVar(len(plan.argFields)))

			plan.argFields = append(plan.argFields, plan.versField)
		}
		s.WriteString(Database().Get().Dialect.QuerySuffix())

		plan.query = s.String()
	})

	return plan.createBindInstance(elem, Database().Get().TypeConverter)
}

func (t *modelInfo) bindGet() *bindPlan {
	plan := &t.getPlan
	plan.once.Do(func() {
		s := bytes.Buffer{}
		s.WriteString("select ")

		x := 0
		for _, col := range t.fields.columns {
			if !col.transient {
				if x > 0 {
					s.WriteString(",")
				}
				s.WriteString(Database().Get().Dialect.QuoteField(col.column))
				plan.argFields = append(plan.argFields, col.name)
				x++
			}
		}
		s.WriteString(" from ")
		s.WriteString(Database().Get().Dialect.QuotedTableForQuery(t.schemaName, t.table))
		s.WriteString(" where ")
		var y = 0
		for _, col := range t.fields.keys {
			//col := t.keys[x]
			if y > 0 {
				s.WriteString(" and ")
			}
			s.WriteString(Database().Get().Dialect.QuoteField(col.column))
			s.WriteString("=")
			s.WriteString(Database().Get().Dialect.BindVar(y))

			plan.keyFields = append(plan.keyFields, col.name)
			y++
		}
		s.WriteString(Database().Get().Dialect.QuerySuffix())

		plan.query = s.String()
	})

	return plan
}
