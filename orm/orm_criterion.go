package orm

import "fmt"

//Criterion An object-oriented representation of a query criterion that may be used
//as a restriction in a <tt>Criteria</tt> query.
//Built-in criterion types are provided by the <tt>Restrictions</tt> factory
//class. This interface might be implemented by application classes that
//define custom restriction criteria.
type Criterion interface {
	ToSqlString(criteria Criteria, dbmap *DbMap) string
	GetValues(criteria Criteria, dbmap *DbMap) interface{}
}

var (
	Restrictions = Restriction{}
)

type Restriction struct{}

func (r Restriction) Like(filedName string, value string) Criterion {
	c := new(simpleExpression)
	c.fieldName = filedName
	c.value = "%" + value + "%"
	c.operator = " like "
	return c
}

//simpleExpression s
type simpleExpression struct {
	fieldName  string
	value      interface{}
	ignoreCase bool
	operator   string
}

func (s simpleExpression) ToSqlString(criteria Criteria, dbmap *DbMap) (sql string) {
	cols := dbmap.findColumns(criteria, s.fieldName)

	sql += fmt.Sprintf("%s %s %s", cols[0], s.operator, "?")

	return
}

func (s simpleExpression) GetValues(criteria Criteria, dbmap *DbMap) interface{} {
	return s.value
}
