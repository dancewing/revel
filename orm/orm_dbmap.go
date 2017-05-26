package orm

//getSQLAlias
func (m *DbMap) getFieldSQLAlias(criteria Criteria, fieldName string) string {

	tmap, err := m.TableFor(criteria.GetEntityType(), true)

	if err != nil {

	} else {
		cls, d := tmap.GetByAny(fieldName)
		if d {
			return criteria.GetAlias() + "_" + "." + cls.column
		}
	}
	return ""
}

func (m *DbMap) getObjectSQLAlias(criteria Criteria) string {

	tmap, err := m.TableFor(criteria.GetEntityType(), true)

	if err != nil {

	} else {
		return tmap.table + " " + criteria.GetAlias() + "_"
	}

	return ""
}

func (m *DbMap) findColumns(criteria Criteria, fieldName string) []string {
	columns := make([]string, 0)

	tmap, err := m.TableFor(criteria.GetEntityType(), true)

	if err != nil {

	} else {
		cls, d := tmap.GetByAny(fieldName)
		if d {
			columns = append(columns, cls.column)
		}
	}
	return columns
}
