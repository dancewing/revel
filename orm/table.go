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
	"strings"
	"time"
)

// TableMap represents a mapping between a Go struct and a database table
// Use dbmap.AddTable() or dbmap.AddTableWithName() to create these
type TableMap struct {
	// Name of database table.
	TableName      string
	SchemaName     string
	gotype         reflect.Type
	Columns        []*ColumnMap
	keys           []*ColumnMap
	indexes        []*IndexMap
	uniqueTogether [][]string
	version        *ColumnMap
	insertPlan     bindPlan
	updatePlan     bindPlan
	deletePlan     bindPlan
	getPlan        bindPlan
	dbmap          *DbMap
	FieldMap       map[string]*ColumnMap // contains all column maps, includes keys
	FieldLowMap    map[string]*ColumnMap
	ColumnMap      map[string]*ColumnMap
	ColumnLowMap   map[string]*ColumnMap
}

func (t *TableMap) String() string {
	return fmt.Sprintf("%v", t)
}

// ResetSql removes cached insert/update/select/delete SQL strings
// associated with this TableMap.  Call this if you've modified
// any column names or the table name itself.
func (t *TableMap) ResetSql() {
	t.insertPlan = bindPlan{}
	t.updatePlan = bindPlan{}
	t.deletePlan = bindPlan{}
	t.getPlan = bindPlan{}
}

// SetKeys lets you specify the fields on a struct that map to primary
// key columns on the table.  If isAutoIncr is set, result.LastInsertId()
// will be used after INSERT to bind the generated id to the Go struct.
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
//
// Panics if isAutoIncr is true, and fieldNames length != 1
//
func (t *TableMap) SetKeys(isAutoIncr bool, fieldNames ...string) *TableMap {
	if isAutoIncr && len(fieldNames) != 1 {
		panic(fmt.Sprintf(
			"gorp: SetKeys: fieldNames length must be 1 if key is auto-increment. (Saw %v fieldNames)",
			len(fieldNames)))
	}
	t.keys = make([]*ColumnMap, 0)
	for _, name := range fieldNames {
		colmap := t.ColMap(name)
		colmap.isPK = true
		colmap.isAutoIncr = isAutoIncr
		t.keys = append(t.keys, colmap)
	}
	t.ResetSql()

	return t
}

// SetUniqueTogether lets you specify uniqueness constraints across multiple
// columns on the table. Each call adds an additional constraint for the
// specified columns.
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
//
// Panics if fieldNames length < 2.
//
func (t *TableMap) SetUniqueTogether(fieldNames ...string) *TableMap {
	if len(fieldNames) < 2 {
		panic(fmt.Sprintf(
			"gorp: SetUniqueTogether: must provide at least two fieldNames to set uniqueness constraint."))
	}

	columns := make([]string, 0)
	for _, name := range fieldNames {
		columns = append(columns, name)
	}
	t.uniqueTogether = append(t.uniqueTogether, columns)
	t.ResetSql()

	return t
}

// ColMap returns the ColumnMap pointer matching the given struct field
// name.  It panics if the struct does not contain a field matching this
// name.
func (t *TableMap) ColMap(field string) *ColumnMap {
	col := colMapOrNil(t, field)
	if col == nil {
		e := fmt.Sprintf("No ColumnMap in table %s type %s with field %s",
			t.TableName, t.gotype.Name(), field)

		panic(e)
	}
	return col
}

// GetByAny return ColumnMap
func (t *TableMap) GetByAny(name string) (*ColumnMap, bool) {
	if fi, ok := t.FieldMap[name]; ok {
		return fi, ok
	}
	if fi, ok := t.FieldLowMap[strings.ToLower(name)]; ok {
		return fi, ok
	}
	if fi, ok := t.ColumnMap[name]; ok {
		return fi, ok
	}
	if fi, ok := t.ColumnLowMap[name]; ok {
		return fi, ok
	}
	return nil, false
}

func colMapOrNil(t *TableMap, field string) *ColumnMap {
	for _, col := range t.Columns {
		if col.fieldName == field || col.ColumnName == field {
			return col
		}
	}
	return nil
}

// IdxMap returns the IndexMap pointer matching the given index name.
func (t *TableMap) IdxMap(field string) *IndexMap {
	for _, idx := range t.indexes {
		if idx.IndexName == field {
			return idx
		}
	}
	return nil
}

// AddIndex registers the index with gorp for specified table with given parameters.
// This operation is idempotent. If index is already mapped, the
// existing *IndexMap is returned
// Function will panic if one of the given for index columns does not exists
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
//
func (t *TableMap) AddIndex(name string, idxtype string, columns []string) *IndexMap {
	// check if we have a index with this name already
	for _, idx := range t.indexes {
		if idx.IndexName == name {
			return idx
		}
	}
	for _, icol := range columns {
		if res := t.ColMap(icol); res == nil {
			e := fmt.Sprintf("No ColumnName in table %s to create index on", t.TableName)
			panic(e)
		}
	}

	idx := &IndexMap{IndexName: name, Unique: false, IndexType: idxtype, columns: columns}
	t.indexes = append(t.indexes, idx)
	t.ResetSql()
	return idx
}

// SetVersionCol sets the column to use as the Version field.  By default
// the "Version" field is used.  Returns the column found, or panics
// if the struct does not contain a field matching this name.
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
func (t *TableMap) SetVersionCol(field string) *ColumnMap {
	c := t.ColMap(field)
	t.version = c
	t.ResetSql()
	return c
}

// SqlForCreateTable gets a sequence of SQL commands that will create
// the specified table and any associated schema
func (t *TableMap) SqlForCreate(ifNotExists bool) string {
	s := bytes.Buffer{}
	dialect := t.dbmap.Dialect

	if strings.TrimSpace(t.SchemaName) != "" {
		schemaCreate := "create schema"
		if ifNotExists {
			s.WriteString(dialect.IfSchemaNotExists(schemaCreate, t.SchemaName))
		} else {
			s.WriteString(schemaCreate)
		}
		s.WriteString(fmt.Sprintf(" %s;", t.SchemaName))
	}

	tableCreate := "create table"
	if ifNotExists {
		s.WriteString(dialect.IfTableNotExists(tableCreate, t.SchemaName, t.TableName))
	} else {
		s.WriteString(tableCreate)
	}
	s.WriteString(fmt.Sprintf(" %s (", dialect.QuotedTableForQuery(t.SchemaName, t.TableName)))

	x := 0
	for _, col := range t.Columns {
		if !col.Transient {
			if x > 0 {
				s.WriteString(", ")
			}
			stype := dialect.ToSqlType(col.gotype, col.MaxSize, col.isAutoIncr)
			s.WriteString(fmt.Sprintf("%s %s", dialect.QuoteField(col.ColumnName), stype))

			if col.isPK || col.isNotNull {
				s.WriteString(" not null")
			}
			if col.isPK && len(t.keys) == 1 {
				s.WriteString(" primary key")
			}
			if col.Unique {
				s.WriteString(" unique")
			}
			if col.isAutoIncr {
				s.WriteString(fmt.Sprintf(" %s", dialect.AutoIncrStr()))
			}

			x++
		}
	}
	if len(t.keys) > 1 {
		s.WriteString(", primary key (")
		for x := range t.keys {
			if x > 0 {
				s.WriteString(", ")
			}
			s.WriteString(dialect.QuoteField(t.keys[x].ColumnName))
		}
		s.WriteString(")")
	}
	if len(t.uniqueTogether) > 0 {
		for _, columns := range t.uniqueTogether {
			s.WriteString(", unique (")
			for i, column := range columns {
				if i > 0 {
					s.WriteString(", ")
				}
				s.WriteString(dialect.QuoteField(column))
			}
			s.WriteString(")")
		}
	}
	s.WriteString(") ")
	s.WriteString(dialect.CreateTableSuffix())
	s.WriteString(dialect.QuerySuffix())
	return s.String()
}

// parse orm model struct field tag expression.
func (t *TableMap) parseExprs(exprs []string) (index, name string, info *ColumnMap, success bool) {

	index = ""
	name = ""
	success = false

	if len(exprs) == 1 {
		name = exprs[0]
	} else if len(exprs) == 2 {
		name = exprs[1]
	}

	if name != "" {
		c, ok := t.GetByAny(name)
		if ok {
			info = c
			success = true
		}
	}

	return
}

// generate condition sql.
func (t *TableMap) getCondSQL(cond *Condition, sub bool, tz *time.Location) (where string, params []interface{}) {
	if cond == nil || cond.IsEmpty() {
		return
	}

	//Q := t.dbmap.Dialect.QuotedTableForQuery(t.SchemaName, t.TableName)

	for i, p := range cond.params {
		if i > 0 {
			if p.isOr {
				where += "OR "
			} else {
				where += "AND "
			}
		}
		if p.isNot {
			where += "NOT "
		}
		if p.isCond {
			w, ps := t.getCondSQL(p.cond, true, tz)
			if w != "" {
				w = fmt.Sprintf("( %s) ", w)
			}
			where += w
			params = append(params, ps...)
		} else {
			exprs := p.exprs

			fmt.Println("exprs :", exprs)

			num := len(exprs) - 1
			operator := ""
			if operators[exprs[num]] {
				operator = exprs[num]
				exprs = exprs[:num]
			}

			_, _, fi, suc := t.parseExprs(exprs)
			if !suc {
				panic(fmt.Errorf("unknown field/column name `%s`", strings.Join(p.exprs, ExprSep)))
			}

			if operator == "" {
				operator = "exact"
			}

			operSQL, args := t.GenerateOperatorSQL(fi, operator, p.args, tz)

			leftCol := fmt.Sprintf("%s", fi.ColumnName)

			t.GenerateOperatorLeftCol(fi, operator, &leftCol)

			where += fmt.Sprintf("%s %s ", leftCol, operSQL)
			params = append(params, args...)

		}
	}

	if !sub && where != "" {
		where = "WHERE " + where
	}

	return
}

// generate group sql.
func (t *TableMap) getGroupSQL(groups []string) (groupSQL string) {
	if len(groups) == 0 {
		return
	}

	Q := t.dbmap.Dialect.QuotedTableForQuery(t.SchemaName, t.TableName)

	groupSqls := make([]string, 0, len(groups))
	for _, group := range groups {
		exprs := strings.Split(group, ExprSep)

		index, _, fi, suc := t.parseExprs(exprs)
		if !suc {
			panic(fmt.Errorf("unknown field/column name `%s`", strings.Join(exprs, ExprSep)))
		}

		groupSqls = append(groupSqls, fmt.Sprintf("%s.%s%s%s", index, Q, fi.ColumnName, Q))
	}

	groupSQL = fmt.Sprintf("GROUP BY %s ", strings.Join(groupSqls, ", "))
	return
}

// generate order sql.
func (t *TableMap) getOrderSQL(orders []string) (orderSQL string) {
	if len(orders) == 0 {
		return
	}

	Q := t.dbmap.Dialect.QuotedTableForQuery(t.SchemaName, t.TableName)

	orderSqls := make([]string, 0, len(orders))
	for _, order := range orders {
		asc := "ASC"
		if order[0] == '-' {
			asc = "DESC"
			order = order[1:]
		}
		exprs := strings.Split(order, ExprSep)

		index, _, fi, suc := t.parseExprs(exprs)
		if !suc {
			panic(fmt.Errorf("unknown field/column name `%s`", strings.Join(exprs, ExprSep)))
		}

		orderSqls = append(orderSqls, fmt.Sprintf("%s.%s%s%s %s", index, Q, fi.ColumnName, Q, asc))
	}

	orderSQL = fmt.Sprintf("ORDER BY %s ", strings.Join(orderSqls, ", "))
	return
}

// generate join string.
func (t *TableMap) getJoinSQL() (join string) {

	//Q := t.dbmap.Dialect.QuotedTableForQuery(t.SchemaName, t.TableName)

	join = ""
	//
	//for _, jt := range t.tables {
	//	if jt.inner {
	//		join += "INNER JOIN "
	//	} else {
	//		join += "LEFT OUTER JOIN "
	//	}
	//	var (
	//		table  string
	//		t1, t2 string
	//		c1, c2 string
	//	)
	//	t1 = "T0"
	//	if jt.jtl != nil {
	//		t1 = jt.jtl.index
	//	}
	//	t2 = jt.index
	//	table = jt.mi.table
	//
	//	switch {
	//	case jt.fi.fieldType == RelManyToMany || jt.fi.fieldType == RelReverseMany || jt.fi.reverse && jt.fi.reverseFieldInfo.fieldType == RelManyToMany:
	//		c1 = jt.fi.mi.fields.pk.column
	//		for _, ffi := range jt.mi.fields.fieldsRel {
	//			if jt.fi.mi == ffi.relModelInfo {
	//				c2 = ffi.column
	//				break
	//			}
	//		}
	//	default:
	//		c1 = jt.fi.column
	//		c2 = jt.fi.relModelInfo.fields.pk.column
	//
	//		if jt.fi.reverse {
	//			c1 = jt.mi.fields.pk.column
	//			c2 = jt.fi.reverseFieldInfo.column
	//		}
	//	}
	//
	//	join += fmt.Sprintf("%s%s%s %s ON %s.%s%s%s = %s.%s%s%s ", Q, table, Q, t2,
	//		t2, Q, c2, Q, t1, Q, c1, Q)
	//}
	return
}

// generate sql with replacing operator string placeholders and replaced values.
func (d *TableMap) GenerateOperatorSQL(fi *ColumnMap, operator string, args []interface{}, tz *time.Location) (string, []interface{}) {
	var sql string

	params := getFlatParams(fi, args, tz)

	if len(params) == 0 {
		panic(fmt.Errorf("operator `%s` need at least one args", operator))
	}
	arg := params[0]

	switch operator {
	case "in":
		marks := make([]string, len(params))
		for i := range marks {
			marks[i] = "?"
		}
		sql = fmt.Sprintf("IN (%s)", strings.Join(marks, ", "))
	case "between":
		if len(params) != 2 {
			panic(fmt.Errorf("operator `%s` need 2 args not %d", operator, len(params)))
		}
		sql = "BETWEEN ? AND ?"
	default:
		if len(params) > 1 {
			panic(fmt.Errorf("operator `%s` need 1 args not %d", operator, len(params)))
		}
		sql = d.dbmap.Dialect.OperatorSQL(operator)
		switch operator {
		case "exact":
			if arg == nil {
				params[0] = "IS NULL"
			}
		case "iexact", "contains", "icontains", "startswith", "endswith", "istartswith", "iendswith":
			param := strings.Replace(ToStr(arg), `%`, `\%`, -1)
			switch operator {
			case "iexact":
			case "contains", "icontains":
				param = fmt.Sprintf("%%%s%%", param)
			case "startswith", "istartswith":
				param = fmt.Sprintf("%s%%", param)
			case "endswith", "iendswith":
				param = fmt.Sprintf("%%%s", param)
			}
			params[0] = param
		case "isnull":
			if b, ok := arg.(bool); ok {
				if b {
					sql = "IS NULL"
				} else {
					sql = "IS NOT NULL"
				}
				params = nil
			} else {
				panic(fmt.Errorf("operator `%s` need a bool value not `%T`", operator, arg))
			}
		}
	}
	return sql, params
}

// gernerate sql string with inner function, such as UPPER(text).
func (d *TableMap) GenerateOperatorLeftCol(*ColumnMap, string, *string) {
	// default not use
}
