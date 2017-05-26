package orm

import (
	"reflect"
)

type Criteria interface {
	Add(criterion Criterion) Criteria
	GetCriterions() []Criterion
	List() ([]interface{}, error)
	UniqueResult() interface{}
	GetAlias() string
	SetProjection(projection Projection) Criteria
	GetProjection() Projection
	GetEntityType() reflect.Type
	GetEntity() interface{}
}

var _ Criteria = new(criteriaImpl)

type criteriaImpl struct {
	rootAlias      string
	rootEntityType reflect.Type
	rootEntity     interface{}
	criterions     []Criterion
	projection     Projection
	dbmap          *DbMap
	tmap           *modelInfo
}

type CriteriaTranslator struct {
	criteria Criteria
	dbmap    *DbMap
}

func (ci criteriaImpl) Add(criterion Criterion) Criteria {
	ci.criterions = append(ci.criterions, criterion)
	return ci
}

func (ci criteriaImpl) GetCriterions() []Criterion {
	return ci.criterions
}

func (ci criteriaImpl) List() ([]interface{}, error) {
	ct := &CriteriaTranslator{
		criteria: ci,
		dbmap:    ci.dbmap,
	}
	return ct.List()
}

func (ci criteriaImpl) UniqueResult() interface{} {

	return nil
}

func (ci criteriaImpl) GetAlias() string {

	return ci.rootAlias
}

func (ci criteriaImpl) SetProjection(projection Projection) Criteria {
	ci.projection = projection
	return ci
}

func (ci criteriaImpl) GetProjection() Projection {
	return ci.projection
}

func (ci criteriaImpl) GetEntityType() reflect.Type {
	return ci.rootEntityType
}

func (ci criteriaImpl) GetEntity() interface{} {
	return ci.rootEntity
}

func newCriteria(dbmap *DbMap, tmap *modelInfo, m interface{}, typ reflect.Type) Criteria {
	c := new(criteriaImpl)
	c.dbmap = dbmap
	c.tmap = tmap
	c.rootEntity = m
	c.rootEntityType = typ
	c.criterions = make([]Criterion, 0)
	c.rootAlias = "this"
	return c
}

//List get results from criteria
func (ct CriteriaTranslator) List() ([]interface{}, error) {

	args := make([]interface{}, 0)

	var (
		selectClause         string
		fromClause           string
		outerJoinsAfterFrom  string
		whereClause          string
		outerJoinsAfterWhere string
		orderByClause        string
		groupByClause        string
	)

	if ct.criteria.GetProjection() == nil {
		selectClause = "*"
	} else {
		selectClause = ct.criteria.GetProjection().ToSqlString(ct.criteria, 0, ct.dbmap)
	}

	fromClause = ct.dbmap.getObjectSQLAlias(ct.criteria)

	for _, cr := range ct.criteria.GetCriterions() {
		whereClause += cr.ToSqlString(ct.criteria, ct.dbmap)

		args = append(args, cr.GetValues(ct.criteria, ct.dbmap))
	}

	//ct.dbmap.getSQLAlias(ct.criteria, nil)

	selectSQL := &Select{
		selectClause:         selectClause,
		fromClause:           fromClause,
		outerJoinsAfterFrom:  outerJoinsAfterFrom,
		whereClause:          whereClause,
		outerJoinsAfterWhere: outerJoinsAfterWhere,
		orderByClause:        orderByClause,
		groupByClause:        groupByClause,
	}

	return ct.dbmap.Select(ct.criteria.GetEntity(), selectSQL.ToStatementString(), args...)
}
