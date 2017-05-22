// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orm

import ()

// Params stores the Params
type Params map[string]interface{}

// ParamsList stores paramslist
type ParamsList []interface{}

type Inserter interface {
	Insert(interface{}) (int64, error)
	Close() error
}

// QuerySeter query seter
type QuerySeter interface {
	// add condition expression to QuerySeter.
	// for example:
	//	filter by UserName == 'slene'
	//	qs.Filter("UserName", "slene")
	//	sql : left outer join profile on t0.id1==t1.id2 where t1.age == 28
	//	Filter("profile__Age", 28)
	// 	 // time compare
	//	qs.Filter("created", time.Now())
	Filter(string, ...interface{}) QuerySeter
	// add NOT condition to querySeter.
	// have the same usage as Filter
	Exclude(string, ...interface{}) QuerySeter
	// set condition to QuerySeter.
	// sql's where condition
	//	cond := orm.NewCondition()
	//	cond1 := cond.And("profile__isnull", false).AndNot("status__in", 1).Or("profile__age__gt", 2000)
	//	//sql-> WHERE T0.`profile_id` IS NOT NULL AND NOT T0.`Status` IN (?) OR T1.`age` >  2000
	//	num, err := qs.SetCond(cond1).Count()
	SetCond(*Condition) QuerySeter
	// get condition from QuerySeter.
	// sql's where condition
	//  cond := orm.NewCondition()
	//  cond = cond.And("profile__isnull", false).AndNot("status__in", 1)
	//  qs = qs.SetCond(cond)
	//  cond = qs.GetCond()
	//  cond := cond.Or("profile__age__gt", 2000)
	//  //sql-> WHERE T0.`profile_id` IS NOT NULL AND NOT T0.`Status` IN (?) OR T1.`age` >  2000
	//  num, err := qs.SetCond(cond).Count()
	GetCond() *Condition
	// add LIMIT value.
	// args[0] means offset, e.g. LIMIT num,offset.
	// if Limit <= 0 then Limit will be set to default limit ,eg 1000
	// if QuerySeter doesn't call Limit, the sql's Limit will be set to default limit, eg 1000
	//  for example:
	//	qs.Limit(10, 2)
	//	// sql-> limit 10 offset 2
	Limit(limit interface{}, args ...interface{}) QuerySeter
	// add OFFSET value
	// same as Limit function's args[0]
	Offset(offset interface{}) QuerySeter
	// add GROUP BY expression
	// for example:
	//	qs.GroupBy("id")
	GroupBy(exprs ...string) QuerySeter
	// add ORDER expression.
	// "column" means ASC, "-column" means DESC.
	// for example:
	//	qs.OrderBy("-status")
	OrderBy(exprs ...string) QuerySeter
	// set relation model to query together.
	// it will query relation models and assign to parent model.
	// for example:
	//	// will load all related fields use left join .
	// 	qs.RelatedSel().One(&user)
	//	// will  load related field only profile
	//	qs.RelatedSel("profile").One(&user)
	//	user.Profile.Age = 32
	RelatedSel(params ...interface{}) QuerySeter
	// Set Distinct
	// for example:
	//  o.QueryTable("policy").Filter("Groups__Group__Users__User", user).
	//    Distinct().
	//    All(&permissions)
	Distinct() QuerySeter
	// return QuerySeter execution result number
	// for example:
	//	num, err = qs.Filter("profile__age__gt", 28).Count()
	Count() (int64, error)
	// check result empty or not after QuerySeter executed
	// the same as QuerySeter.Count > 0
	Exist() bool
	// execute update with parameters
	// for example:
	//	num, err = qs.Filter("user_name", "slene").Update(Params{
	//		"Nums": ColValue(Col_Minus, 50),
	//	}) // user slene's Nums will minus 50
	//	num, err = qs.Filter("UserName", "slene").Update(Params{
	//		"user_name": "slene2"
	//	}) // user slene's  name will change to slene2
	Update(values Params) (int64, error)
	// delete from table
	//for example:
	//	num ,err = qs.Filter("user_name__in", "testing1", "testing2").Delete()
	// 	//delete two user  who's name is testing1 or testing2
	Delete() (int64, error)
	// return a insert queryer.
	// it can be used in times.
	// example:
	// 	i,err := sq.PrepareInsert()
	// 	num, err = i.Insert(&user1) // user table will add one record user1 at once
	//	num, err = i.Insert(&user2) // user table will add one record user2 at once
	//	err = i.Close() //don't forget call Close
	PrepareInsert() (Inserter, error)
	// query all data and map to containers.
	// cols means the columns when querying.
	// for example:
	//	var users []*User
	//	qs.All(&users) // users[0],users[1],users[2] ...
	All(container interface{}, cols ...string) (int64, error)
	// query one row data and map to containers.
	// cols means the columns when querying.
	// for example:
	//	var user User
	//	qs.One(&user) //user.UserName == "slene"
	One(container interface{}, cols ...string) error
	// query all data and map to []map[string]interface.
	// expres means condition expression.
	// it converts data to []map[column]value.
	// for example:
	//	var maps []Params
	//	qs.Values(&maps) //maps[0]["UserName"]=="slene"
	Values(results *[]Params, exprs ...string) (int64, error)
	// query all data and map to [][]interface
	// it converts data to [][column_index]value
	// for example:
	//	var list []ParamsList
	//	qs.ValuesList(&list) // list[0][1] == "slene"
	ValuesList(results *[]ParamsList, exprs ...string) (int64, error)
	// query all data and map to []interface.
	// it's designed for one column record set, auto change to []value, not [][column]value.
	// for example:
	//	var list ParamsList
	//	qs.ValuesFlat(&list, "UserName") // list[0] == "slene"
	ValuesFlat(result *ParamsList, expr string) (int64, error)
	// query all rows into map[string]interface with specify key and value column name.
	// keyCol = "name", valueCol = "value"
	// table data
	// name  | value
	// total | 100
	// found | 200
	// to map[string]interface{}{
	// 	"total": 100,
	// 	"found": 200,
	// }
	RowsToMap(result *Params, keyCol, valueCol string) (int64, error)
	// query all rows into struct with specify key and value column name.
	// keyCol = "name", valueCol = "value"
	// table data
	// name  | value
	// total | 100
	// found | 200
	// to struct {
	// 	Total int
	// 	Found int
	// }
	RowsToStruct(ptrStruct interface{}, keyCol, valueCol string) (int64, error)
}
