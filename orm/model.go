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
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

var errSkipField = errors.New("skip field")

// modelInfo represents a mapping between a Go struct and a database table
// Use dbmap.AddTable() or dbmap.AddTableWithName() to create these
type modelInfo struct {
	// Name of database table.
	//TableName  string
	schemaName string
	//FullName   string
	gotype reflect.Type
	//Columns        []*fieldInfo
	//keys           []*fieldInfo
	indexes        []*IndexMap
	uniqueTogether [][]string
	version        *fieldInfo
	insertPlan     bindPlan
	updatePlan     bindPlan
	deletePlan     bindPlan
	getPlan        bindPlan
	m2mInsertPlan  bindPlan
	m2mQueryPlan   bindPlan

	pkg       string
	name      string
	fullName  string
	table     string
	model     interface{}
	fields    *fields
	manual    bool          // true, model created by code, false for many-to-many tables
	addrField reflect.Value //store the original struct value
	uniques   []string
	isThrough bool
}

// new model info
func newModelInfo(val reflect.Value) (mi *modelInfo) {
	mi = &modelInfo{}
	mi.fields = newFields()
	ind := reflect.Indirect(val)
	mi.addrField = val
	mi.name = ind.Type().Name()
	mi.fullName = getFullName(ind.Type())
	addModelFields(mi, ind, "", []int{})
	return
}

// index: FieldByIndex returns the nested field corresponding to index
func addModelFields(mi *modelInfo, ind reflect.Value, mName string, index []int) {
	var (
		err error
		fi  *fieldInfo
		sf  reflect.StructField
	)

	for i := 0; i < ind.NumField(); i++ {
		field := ind.Field(i)
		sf = ind.Type().Field(i)
		// if the field is unexported skip
		if sf.PkgPath != "" {
			continue
		}
		// add anonymous struct fields
		if sf.Anonymous {
			addModelFields(mi, field, mName+"."+sf.Name, append(index, i))
			continue
		}

		fi, err = newFieldInfo(mi, field, sf, mName)
		if err == errSkipField {
			err = nil
			continue
		} else if err != nil {
			break
		}
		//record current field index
		fi.fieldIndex = append(index, i)
		fi.mi = mi
		fi.gotype = field.Type()
		fi.inModel = true
		if !mi.fields.Add(fi) {
			err = fmt.Errorf("duplicate column name: %s", fi.column)
			break
		}
		if fi.pk {
			// if mi.fields.pk != nil {
			// 	err = fmt.Errorf("one model must have one pk field only")
			// 	break
			// } else {
			// 	mi.fields.pk = fi
			// }

			mi.fields.keys[fi.name] = fi
		}
	}

	if err != nil {
		fmt.Println(fmt.Errorf("field: %s.%s, %s", ind.Type(), sf.Name, err))
		os.Exit(2)
	}
}

// ResetSql removes cached insert/update/select/delete SQL strings
// associated with this modelInfo.  Call this if you've modified
// any column names or the table name itself.
func (t *modelInfo) ResetSql() {
	t.insertPlan = bindPlan{}
	t.updatePlan = bindPlan{}
	t.deletePlan = bindPlan{}
	t.getPlan = bindPlan{}
	t.m2mInsertPlan = bindPlan{}
	t.m2mQueryPlan = bindPlan{}
}

// SetKeys lets you specify the fields on a struct that map to primary
// key columns on the table.  If isAutoIncr is set, result.LastInsertId()
// will be used after INSERT to bind the generated id to the Go struct.
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
//
// Panics if isAutoIncr is true, and fieldNames length != 1
//
func (t *modelInfo) SetKeys(isAutoIncr bool, fieldNames ...string) *modelInfo {
	if isAutoIncr && len(fieldNames) != 1 {
		panic(fmt.Sprintf(
			"gorp: SetKeys: fieldNames length must be 1 if key is auto-increment. (Saw %v fieldNames)",
			len(fieldNames)))
	}
	// t.keys = make([]*fieldInfo, 0)
	for _, name := range fieldNames {
		colmap := t.ColMap(name)
		colmap.pk = true
		colmap.auto = isAutoIncr
		t.fields.keys[name] = colmap
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
func (t *modelInfo) SetUniqueTogether(fieldNames ...string) *modelInfo {
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

// ColMap returns the fieldInfo pointer matching the given struct field
// name.  It panics if the struct does not contain a field matching this
// name.
func (t *modelInfo) ColMap(field string) *fieldInfo {
	col := colMapOrNil(t, field)
	if col == nil {
		e := fmt.Sprintf("No fieldInfo in table %s type %s with field %s",
			t.table, t.gotype.Name(), field)

		panic(e)
	}
	return col
}

// GetByAny return fieldInfo
func (t *modelInfo) GetByAny(name string) (*fieldInfo, bool) {
	if fi, ok := t.fields.fields[name]; ok {
		return fi, ok
	}
	if fi, ok := t.fields.fieldsLow[strings.ToLower(name)]; ok {
		return fi, ok
	}
	// if fi, ok := t.fieldInfo[name]; ok {
	// 	return fi, ok
	// }
	// if fi, ok := t.ColumnLowMap[name]; ok {
	// 	return fi, ok
	// }
	return nil, false
}

func colMapOrNil(t *modelInfo, field string) *fieldInfo {
	for _, col := range t.fields.columns {
		if col.name == field || col.column == field {
			return col
		}
	}
	return nil
}

// IdxMap returns the IndexMap pointer matching the given index name.
func (t *modelInfo) IdxMap(field string) *IndexMap {
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
func (t *modelInfo) AddIndex(name string, idxtype string, columns []string) *IndexMap {
	// check if we have a index with this name already
	for _, idx := range t.indexes {
		if idx.IndexName == name {
			return idx
		}
	}
	for _, icol := range columns {
		if res := t.ColMap(icol); res == nil {
			e := fmt.Sprintf("No ColumnName in table %s to create index on", t.table)
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
func (t *modelInfo) SetVersionCol(field string) *fieldInfo {
	c := t.ColMap(field)
	t.version = c
	t.ResetSql()
	return c
}

// SqlForCreateTable gets a sequence of SQL commands that will create
// the specified table and any associated schema
func (t *modelInfo) SqlForCreate(ifNotExists bool) string {

	s := bytes.Buffer{}
	dialect := Database().Get().Dialect

	if strings.TrimSpace(t.schemaName) != "" {
		schemaCreate := "create schema"
		if ifNotExists {
			s.WriteString(dialect.IfSchemaNotExists(schemaCreate, t.schemaName))
		} else {
			s.WriteString(schemaCreate)
		}
		s.WriteString(fmt.Sprintf(" %s;", t.schemaName))
	}

	tableCreate := "create table"
	if ifNotExists {
		s.WriteString(dialect.IfTableNotExists(tableCreate, t.schemaName, t.table))
	} else {
		s.WriteString(tableCreate)
	}
	s.WriteString(fmt.Sprintf(" %s (", dialect.QuotedTableForQuery(t.schemaName, t.table)))

	x := 0
	for _, col := range t.fields.columns {

		if col.transient || !col.dbcol {
			continue
		}

		if x > 0 {
			s.WriteString(", ")
		}

		stype := ""

		if col.rel {
			if col.fieldType == RelForeignKey || col.fieldType == RelOneToOne {
				stype = dialect.ToSqlType(col.relModelInfo.fields.GetOnePrimaryKey().gotype, col.relModelInfo.fields.GetOnePrimaryKey().size, false)
			}

		} else {
			stype = dialect.ToSqlType(col.gotype, col.size, col.auto)
		}
		//stype := dialect.ToSqlType(col.gotype, col.size, col.auto)

		s.WriteString(fmt.Sprintf("%s %s", dialect.QuoteField(col.column), stype))

		if col.pk || col.isNotNull {
			s.WriteString(" not null")
		}
		if col.pk && len(t.fields.keys) == 1 {
			s.WriteString(" primary key")
		}
		if col.unique {
			s.WriteString(" unique")
		}
		if col.auto {
			s.WriteString(fmt.Sprintf(" %s", dialect.AutoIncrStr()))
		}

		x++

	}
	if len(t.fields.keys) > 1 {
		s.WriteString(", primary key (")

		var index = 0
		for _, f := range t.fields.keys {
			if index > 0 {
				s.WriteString(", ")
			}
			s.WriteString(dialect.QuoteField(f.column))
			index++
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
func (t *modelInfo) parseExprs(exprs []string) (index, name string, info *fieldInfo, success bool) {

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

// new field info
func newFieldInfo(mi *modelInfo, field reflect.Value, sf reflect.StructField, mName string) (fi *fieldInfo, err error) {
	var (
		tag       string
		tagValue  string
		initial   StrTo // store the default value
		fieldType int
		attrs     map[string]bool
		tags      map[string]string
		addrField reflect.Value
	)

	fi = new(fieldInfo)

	// if field which CanAddr is the follow type
	//  A value is addressable if it is an element of a slice,
	//  an element of an addressable array, a field of an
	//  addressable struct, or the result of dereferencing a pointer.
	addrField = field
	if field.CanAddr() && field.Kind() != reflect.Ptr {
		addrField = field.Addr()
		if _, ok := addrField.Interface().(Fielder); !ok {
			if field.Kind() == reflect.Slice {
				addrField = field
			}
		}
	}

	attrs, tags = parseStructTag(sf.Tag.Get(defaultStructTagName))

	if _, ok := attrs["-"]; ok {
		return nil, errSkipField
	}

	digits := tags["digits"]
	decimals := tags["decimals"]
	size := tags["size"]
	onDelete := tags["on_delete"]

	initial.Clear()
	if v, ok := tags["default"]; ok {
		initial.Set(v)
	}

checkType:
	switch f := addrField.Interface().(type) {
	case Fielder:
		fi.isFielder = true
		if field.Kind() == reflect.Ptr {
			err = fmt.Errorf("the model Fielder can not be use ptr")
			goto end
		}
		fieldType = f.FieldType()
		if fieldType&IsRelField > 0 {
			err = fmt.Errorf("unsupport type custom field, please refer to https://github.com/astaxie/beego/blob/master/orm/models_fields.go#L24-L42")
			goto end
		}
	default:
		tag = "rel"
		tagValue = tags[tag]
		if tagValue != "" {
			switch tagValue {
			case "fk":
				fieldType = RelForeignKey
				break checkType
			case "one":
				fieldType = RelOneToOne
				break checkType
			case "m2m":
				fieldType = RelManyToMany
				if tv := tags["rel_table"]; tv != "" {
					fi.relTable = tv
				} else if tv := tags["rel_through"]; tv != "" {
					fi.relThrough = tv
				}
				break checkType
			default:
				err = fmt.Errorf("rel only allow these value: fk, one, m2m")
				goto wrongTag
			}
		}
		tag = "reverse"
		tagValue = tags[tag]
		if tagValue != "" {
			switch tagValue {
			case "one":
				fieldType = RelReverseOne
				break checkType
			case "many":
				fieldType = RelReverseMany
				if tv := tags["rel_table"]; tv != "" {
					fi.relTable = tv
				} else if tv := tags["rel_through"]; tv != "" {
					fi.relThrough = tv
				}
				break checkType
			default:
				err = fmt.Errorf("reverse only allow these value: one, many")
				goto wrongTag
			}
		}

		fieldType, err = getFieldType(addrField)
		if err != nil {
			goto end
		}
		if fieldType == TypeCharField {
			switch tags["type"] {
			case "text":
				fieldType = TypeTextField
			case "json":
				fieldType = TypeJSONField
			case "jsonb":
				fieldType = TypeJsonbField
			}
		}
		if fieldType == TypeFloatField && (digits != "" || decimals != "") {
			fieldType = TypeDecimalField
		}
		if fieldType == TypeDateTimeField && tags["type"] == "date" {
			fieldType = TypeDateField
		}
		if fieldType == TypeTimeField && tags["type"] == "time" {
			fieldType = TypeTimeField
		}
	}

	// check the rel and reverse type
	// rel should Ptr
	// reverse should slice []*struct
	switch fieldType {
	case RelForeignKey, RelOneToOne, RelReverseOne:
		if field.Kind() != reflect.Ptr {
			err = fmt.Errorf("rel/reverse:one field must be *%s", field.Type().Name())
			goto end
		}
	case RelManyToMany, RelReverseMany:
		if field.Kind() != reflect.Slice {
			err = fmt.Errorf("rel/reverse:many field must be slice")
			goto end
		} else {
			if field.Type().Elem().Kind() != reflect.Ptr {
				err = fmt.Errorf("rel/reverse:many slice must be []*%s", field.Type().Elem().Name())
				goto end
			}
		}
	}

	if fieldType&IsFieldType == 0 {
		err = fmt.Errorf("wrong field type")
		goto end
	}

	fi.fieldType = fieldType
	fi.name = sf.Name
	fi.column = getColumnName(fieldType, addrField, sf, tags["column"])
	fi.addrValue = addrField
	fi.sf = sf
	fi.fullName = mi.fullName + mName + "." + sf.Name

	fi.null = attrs["null"]
	fi.index = attrs["index"]
	fi.auto = attrs["auto"]
	fi.pk = attrs["pk"]
	fi.unique = attrs["unique"]

	// Mark object property if there is attribute "default" in the orm configuration
	if _, ok := tags["default"]; ok {
		fi.colDefault = true
	}

	switch fieldType {
	case RelManyToMany, RelReverseMany, RelReverseOne:
		fi.null = false
		fi.index = false
		fi.auto = false
		fi.pk = false
		fi.unique = false
	default:
		fi.dbcol = true
	}

	switch fieldType {
	case RelForeignKey, RelOneToOne, RelManyToMany:
		fi.rel = true
		if fieldType == RelOneToOne {
			fi.unique = true
		}
	case RelReverseMany, RelReverseOne:
		fi.reverse = true
	}

	if fi.rel && fi.dbcol {
		switch onDelete {
		case odCascade, odDoNothing:
		case odSetDefault:
			if !initial.Exist() {
				err = errors.New("on_delete: set_default need set field a default value")
				goto end
			}
		case odSetNULL:
			if !fi.null {
				err = errors.New("on_delete: set_null need set field null")
				goto end
			}
		default:
			if onDelete == "" {
				onDelete = odCascade
			} else {
				err = fmt.Errorf("on_delete value expected choice in `cascade,set_null,set_default,do_nothing`, unknown `%s`", onDelete)
				goto end
			}
		}

		fi.onDelete = onDelete
	}

	switch fieldType {
	case TypeBooleanField:
	case TypeCharField, TypeJSONField, TypeJsonbField:
		if size != "" {
			v, e := StrTo(size).Int32()
			if e != nil {
				err = fmt.Errorf("wrong size value `%s`", size)
			} else {
				fi.size = int(v)
			}
		} else {
			fi.size = 255
			fi.toText = true
		}
	case TypeTextField:
		fi.index = false
		fi.unique = false
	case TypeTimeField, TypeDateField, TypeDateTimeField:
		if attrs["auto_now"] {
			fi.autoNow = true
		} else if attrs["auto_now_add"] {
			fi.autoNowAdd = true
		}
	case TypeFloatField:
	case TypeDecimalField:
		d1 := digits
		d2 := decimals
		v1, er1 := StrTo(d1).Int8()
		v2, er2 := StrTo(d2).Int8()
		if er1 != nil || er2 != nil {
			err = fmt.Errorf("wrong digits/decimals value %s/%s", d2, d1)
			goto end
		}
		fi.digits = int(v1)
		fi.decimals = int(v2)
	default:
		switch {
		case fieldType&IsIntegerField > 0:
		case fieldType&IsRelField > 0:
		}
	}

	if fieldType&IsIntegerField == 0 {
		if fi.auto {
			err = fmt.Errorf("non-integer type cannot set auto")
			goto end
		}
	}

	if fi.auto || fi.pk {
		if fi.auto {
			switch addrField.Elem().Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
			default:
				err = fmt.Errorf("auto primary key only support int, int32, int64, uint, uint32, uint64 but found `%s`", addrField.Elem().Kind())
				goto end
			}
			fi.pk = true
		}
		fi.null = false
		fi.index = false
		fi.unique = false
	}

	if fi.unique {
		fi.index = false
	}

	// can not set default for these type
	if fi.auto || fi.pk || fi.unique || fieldType == TypeTimeField || fieldType == TypeDateField || fieldType == TypeDateTimeField {
		initial.Clear()
	}

	if initial.Exist() {
		v := initial
		switch fieldType {
		case TypeBooleanField:
			_, err = v.Bool()
		case TypeFloatField, TypeDecimalField:
			_, err = v.Float64()
		case TypeBitField:
			_, err = v.Int8()
		case TypeSmallIntegerField:
			_, err = v.Int16()
		case TypeIntegerField:
			_, err = v.Int32()
		case TypeBigIntegerField:
			_, err = v.Int64()
		case TypePositiveBitField:
			_, err = v.Uint8()
		case TypePositiveSmallIntegerField:
			_, err = v.Uint16()
		case TypePositiveIntegerField:
			_, err = v.Uint32()
		case TypePositiveBigIntegerField:
			_, err = v.Uint64()
		}
		if err != nil {
			tag, tagValue = "default", tags["default"]
			goto wrongTag
		}
	}

	fi.initial = initial
end:
	if err != nil {
		return nil, err
	}
	return
wrongTag:
	return nil, fmt.Errorf("wrong tag format: `%s:\"%s\"`, %s", tag, tagValue, err)
}

// combine related model info to new model info.
// prepare for relation models query.
func newM2MModelInfo(m1, m2 *modelInfo) (mi *modelInfo) {

	if len(m1.fields.keys) != 1 || len(m2.fields.keys) != 1 {
		panic(fmt.Errorf("Many-to-Many Models (%s,%s) must have one key", m1.table, m2.table))
	}

	var (
		m1key, m2key *fieldInfo
		i            int
	)
	i = 0
	for _, f := range m1.fields.keys {
		if i == 0 {
			m1key = f
		}
		i++
	}
	i = 0
	for _, f := range m2.fields.keys {
		if i == 0 {
			m2key = f
		}
		i++
	}

	mi = new(modelInfo)

	mi.manual = false

	mi.fields = newFields()
	mi.table = m1.table + "_" + m2.table
	mi.name = camelString(mi.table)
	mi.fullName = m1.pkg + "." + mi.name

	//	fa := new(fieldInfo) // pk
	f1 := new(fieldInfo) // m1 table RelForeignKey
	f2 := new(fieldInfo) // m2 table RelForeignKey

	f1.dbcol = true
	f2.dbcol = true

	f1.gotype = m1key.gotype
	f2.gotype = m2key.gotype

	f1.fieldType = RelForeignKey
	f2.fieldType = RelForeignKey
	f1.name = camelString(m1.table)
	f2.name = camelString(m2.table)
	f1.fullName = mi.fullName + "." + f1.name
	f2.fullName = mi.fullName + "." + f2.name
	f1.column = m1.fields.GetOnePrimaryKey().column
	f2.column = m2.fields.GetOnePrimaryKey().column
	f1.rel = true
	f2.rel = true
	f1.relTable = m1.table
	f2.relTable = m2.table
	f1.relModelInfo = m1
	f2.relModelInfo = m2
	f1.mi = mi
	f2.mi = mi

	mi.fields.Add(f1)
	mi.fields.Add(f2)

	mi.fields.keys[f1.name] = f1
	mi.fields.keys[f2.name] = f2

	mi.uniques = []string{f1.column, f2.column}
	return
}
