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
)

func (plan *bindPlan) createM2MBindInstance(conv TypeConverter) (bindInstance, error) {
	bi := bindInstance{query: plan.query, autoIncrIdx: plan.autoIncrIdx, autoIncrFieldName: plan.autoIncrFieldName, versField: plan.versField}

	var err error

	for d := 0; d < len(plan.paramValues); d++ {

		val := plan.paramValues[d]

		if conv != nil {
			val, err = conv.ToDb(val)
			if err != nil {
				return bindInstance{}, err
			}
		}
		bi.args = append(bi.args, val)
	}

	return bi, nil
}

func (t *modelInfo) bindM2MInsert(elem reflect.Value, field string, args []interface{}) (bindInstance, error) {

	plan := &t.m2mInsertPlan

	plan.once.Do(func() {

		plan.autoIncrIdx = -1

		s := bytes.Buffer{}

		relField := t.fields.GetByName(field)

		if relField == nil {
			panic(fmt.Sprintf("Can't find relation field :%s", field))
		}

		relThroughModelInfo := relField.relThroughModelInfo
		relModelInfo := relField.relModelInfo

		s.WriteString(fmt.Sprintf("insert into %s ", Database().Get().Dialect.QuotedTableForQuery(t.schemaName, relThroughModelInfo.table)))

		plan.paramValues = make([]interface{}, 0)

		mPK := t.fields.GetOnePrimaryKey()
		fPK := relModelInfo.fields.GetOnePrimaryKey()

		reveseKeyValue := getFieldValue(elem.Interface(), mPK.name)

		loop := 0
		for index := range args {

			if loop == 0 {
				s.WriteString("(")
			}

			x := 0

			av := args[index]

			s2 := bytes.Buffer{}
			s3 := bytes.Buffer{}

			for _, col := range relThroughModelInfo.fields.columns {

				//col := t.Columns[y]
				if !(col.auto && Database().Get().Dialect.AutoIncrBindValue() == "") {

					if col.transient || col.fieldType == RelManyToMany || col.fieldType == RelReverseMany {

					} else {

						if x > 0 {
							s2.WriteString(",")
							s3.WriteString(",")
						}

						s3.WriteString(Database().Get().Dialect.QuoteField(col.column))

						if col.auto {
							s2.WriteString(Database().Get().Dialect.AutoIncrBindValue())
							plan.autoIncrIdx = x
							plan.autoIncrFieldName = col.column
						} else {
							if col.DefaultValue == "" {
								s2.WriteString(Database().Get().Dialect.BindVar(x))
								if col == t.version {
									plan.versField = col.name
									if loop == 0 {
										plan.argFields = append(plan.argFields, versFieldConst)
									}
								} else {
									if col.fieldType == RelManyToMany || col.fieldType == RelReverseMany {

									} else {
										if loop == 0 {
											plan.argFields = append(plan.argFields, col.column)
										}
									}

								}
								x++
							} else {
								s2.WriteString(col.DefaultValue)
							}
						}

						if mPK.column == col.column {
							plan.paramValues = append(plan.paramValues, reveseKeyValue)
						} else {
							plan.paramValues = append(plan.paramValues, getFieldValue(av, fPK.name))
						}

					}

				} else {
					plan.autoIncrIdx = x
					plan.autoIncrFieldName = col.name
				}
				x++

			}

			if loop == 0 {
				s.WriteString(s3.String())
				s.WriteString(") values (")
				s.WriteString(s2.String())
				s.WriteString(")")
			} else {
				s.WriteString(", (")
				s.WriteString(s2.String())
				s.WriteString(")")
			}

			loop++

		}

		if plan.autoIncrIdx > -1 {
			s.WriteString(Database().Get().Dialect.AutoIncrInsertSuffix(t.fields.GetByIndex(plan.autoIncrIdx)))
		}
		s.WriteString(Database().Get().Dialect.QuerySuffix())

		plan.query = s.String()

	})

	return plan.createM2MBindInstance(Database().Get().TypeConverter)
}

func (t *modelInfo) bindM2MQuery(elem reflect.Value, field string) (bindInstance, error) {

	plan := &t.m2mQueryPlan

	plan.once.Do(func() {

		pk := t.fields.GetOnePrimaryKey()
		pkName := pk.name

		reveseKeyValue := getFieldValue(elem.Interface(), pkName)

		if reveseKeyValue == nil {
			panic(fmt.Sprintf("can't find m2m as %s 's key(%s) is null", t.name, pkName))
		}

		plan.paramValues = make([]interface{}, 0)

		plan.paramValues = append(plan.paramValues, reveseKeyValue)

		plan.autoIncrIdx = -1

		s := bytes.Buffer{}

		relField := t.fields.GetByName(field)

		if relField == nil {
			panic(fmt.Sprintf("Can't find relation field :%s", field))
		}

		relThroughModelInfo := relField.relThroughModelInfo

		relModelInfo := relField.relModelInfo

		joinColumn := relModelInfo.fields.GetOnePrimaryKey().column

		targetTable := Database().Get().Dialect.QuotedTableForQuery(relModelInfo.schemaName, relModelInfo.table)
		joinTable := Database().Get().Dialect.QuotedTableForQuery(relThroughModelInfo.schemaName, relThroughModelInfo.table)

		//Select
		s.WriteString(fmt.Sprintf("select %s.* from %s left join %s on %s.%s = %s.%s ", targetTable, targetTable, joinTable, targetTable, Database().Get().Dialect.QuoteField(joinColumn), joinTable, Database().Get().Dialect.QuoteField(joinColumn)))
		//Where
		s.WriteString(fmt.Sprintf("where %s.%s = ? ", joinTable, Database().Get().Dialect.QuoteField(pk.column)))

		s.WriteString(Database().Get().Dialect.QuerySuffix())

		plan.query = s.String()

	})

	return plan.createM2MBindInstance(Database().Get().TypeConverter)
}
