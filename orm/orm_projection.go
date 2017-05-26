package orm

type Projection interface {
	ToSqlString(criteria Criteria, position int, dbMap *DbMap) string
}
