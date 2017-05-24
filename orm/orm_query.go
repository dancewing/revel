package orm

import "fmt"

func (m *DbMap) Count(qs *querySet) (cnt int64, err error) {

	//tables.parseRelated(qs.related, qs.relDepth)

	where, args := qs.tmap.getCondSQL(qs.cond, false, DefaultTimeLoc)
	groupBy := qs.tmap.getGroupSQL(qs.groups)
	qs.tmap.getOrderSQL(qs.orders)
	//	join := qs.tmap.getJoinSQL()

	Q := m.Dialect.QuotedTableForQuery(qs.tmap.SchemaName, qs.tmap.TableName)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s %s %s", Q, where, groupBy)

	if groupBy != "" {
		query = fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS T", query)
	}

	//d.ins.ReplaceMarks(&query)

	row := m.QueryRow(query, args...)

	err = row.Scan(&cnt)
	return
}
