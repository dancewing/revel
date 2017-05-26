package orm

type Select struct {
	selectClause         string
	fromClause           string
	outerJoinsAfterFrom  string
	whereClause          string
	outerJoinsAfterWhere string
	orderByClause        string
	groupByClause        string
}

func (s Select) ToStatementString() (sql string) {

	sql += "select " + s.selectClause + " from " + s.fromClause

	if s.outerJoinsAfterFrom != "" {
		sql += s.outerJoinsAfterFrom
	}

	if s.whereClause != "" || s.outerJoinsAfterWhere != "" {
		sql += " where "

		if s.outerJoinsAfterWhere != "" {
			sql += s.outerJoinsAfterWhere

			if s.whereClause != "" {
				sql += " and "
			}
		}

		if s.whereClause != "" {
			sql += s.whereClause
		}
	}

	if s.groupByClause != "" {
		sql += "  group by " + s.groupByClause
	}

	if s.orderByClause != "" {
		sql += "  order by  " + s.orderByClause
	}

	return
}
