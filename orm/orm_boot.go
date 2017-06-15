package orm

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

func RegisterModel(i interface{}) {
	RegisterModelWithSchema(i, "")
}

// RegisterModelWithSchema , RegisterModel with schema name.
func RegisterModelWithSchema(model interface{}, schema string) {
	val := reflect.ValueOf(model)
	typ := reflect.Indirect(val).Type()

	if val.Kind() != reflect.Ptr {
		panic(fmt.Errorf("<orm.RegisterModel> cannot use non-ptr model struct `%s`", getFullName(typ)))
	}
	// For this case:
	// u := &User{}
	// registerModel(&u)
	if typ.Kind() == reflect.Ptr {
		panic(fmt.Errorf("<orm.RegisterModel> only allow ptr model struct, it looks you use two reference to the struct `%s`", typ))
	}

	//t := reflect.TypeOf(i)
	table := getTableName(val)

	// check if we have a table for this type already
	// if so, update the name and return the existing pointer

	// models's fullname is pkgpath + struct name
	name := getFullName(typ)
	if _, ok := modelCache.getByFullName(name); ok {
		fmt.Printf("<orm.RegisterModel> model `%s` repeat register, must be unique\n", name)
		os.Exit(2)
	}

	if _, ok := modelCache.get(table); ok {
		fmt.Printf("<orm.RegisterModel> table name `%s` repeat register, must be unique\n", table)
		os.Exit(2)
	}

	mi := newModelInfo(val)
	mi.gotype = typ
	//keys := getTableKeys(val)

	//mi := initialmodelInfo(typ, table, schema, keys)

	mi.table = table
	mi.pkg = typ.PkgPath()
	mi.model = model
	mi.manual = true
	modelCache.set(table, mi)

}

// BootStrap bootrap models.
// make all model parsed and can not add more models
func BootStrap() {
	if modelCache.done {
		return
	}
	modelCache.Lock()
	defer modelCache.Unlock()
	bootStrap()
	modelCache.done = true
}

// boostrap models
func bootStrap() {
	if modelCache.done {
		return
	}
	var (
		err    error
		models map[string]*modelInfo
	)
	// if dataBaseCache.getDefault() == nil {
	// 	err = fmt.Errorf("must have one register DataBase alias named `default`")
	// 	goto end
	// }

	// set rel and reverse model
	// RelManyToMany set the relTable
	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.columns {

			if fi.rel || fi.reverse {
				elm := fi.addrValue.Type().Elem()
				if fi.fieldType == RelReverseMany || fi.fieldType == RelManyToMany {
					elm = elm.Elem()
				}
				// check the rel or reverse model already register
				name := getFullName(elm)
				mii, ok := modelCache.getByFullName(name)
				if !ok || mii.pkg != elm.PkgPath() {
					err = fmt.Errorf("can not find rel in field `%s`, `%s` may be miss register", fi.fullName, elm.String())
					goto end
				}
				fi.relModelInfo = mii

				switch fi.fieldType {
				case RelManyToMany:
					if fi.relThrough != "" {
						if i := strings.LastIndex(fi.relThrough, "."); i != -1 && len(fi.relThrough) > (i+1) {
							pn := fi.relThrough[:i]
							rmi, ok := modelCache.getByFullName(fi.relThrough)
							if !ok || pn != rmi.pkg {
								err = fmt.Errorf("field `%s` wrong rel_through value `%s` cannot find table", fi.fullName, fi.relThrough)
								goto end
							}
							fi.relThroughModelInfo = rmi
							fi.relTable = rmi.table
						} else {
							err = fmt.Errorf("field `%s` wrong rel_through value `%s`", fi.fullName, fi.relThrough)
							goto end
						}
					} else {
						i := newM2MModelInfo(mi, mii)
						if fi.relTable != "" {
							i.table = fi.relTable
						}
						if v := modelCache.set(i.table, i); v != nil {
							err = fmt.Errorf("the rel table name `%s` already registered, cannot be use, please change one", fi.relTable)
							goto end
						}
						fi.relTable = i.table
						fi.relThroughModelInfo = i
					}

					fi.relThroughModelInfo.isThrough = true
				}
			}
		}
	}

	// check the rel filed while the relModelInfo also has filed point to current model
	// if not exist, add a new field to the relModelInfo
	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.fieldsRel {
			switch fi.fieldType {
			case RelForeignKey, RelOneToOne, RelManyToMany:
				inModel := false
				for _, ffi := range fi.relModelInfo.fields.fieldsReverse {
					if ffi.relModelInfo == mi {
						inModel = true
						break
					}
				}
				if !inModel {
					rmi := fi.relModelInfo
					ffi := new(fieldInfo)
					ffi.name = mi.name
					ffi.column = ffi.name
					ffi.fullName = rmi.fullName + "." + ffi.name
					ffi.reverse = true
					ffi.relModelInfo = mi
					ffi.mi = rmi
					if fi.fieldType == RelOneToOne {
						ffi.fieldType = RelReverseOne
					} else {
						ffi.fieldType = RelReverseMany
					}
					if !rmi.fields.Add(ffi) {
						added := false
						for cnt := 0; cnt < 5; cnt++ {
							ffi.name = fmt.Sprintf("%s%d", mi.name, cnt)
							ffi.column = ffi.name
							ffi.fullName = rmi.fullName + "." + ffi.name
							if added = rmi.fields.Add(ffi); added {
								break
							}
						}
						if !added {
							panic(fmt.Errorf("cannot generate auto reverse field info `%s` to `%s`", fi.fullName, ffi.fullName))
						}
					}
				}
			}
		}
	}

	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.fieldsRel {
			switch fi.fieldType {
			case RelManyToMany:
				for _, ffi := range fi.relThroughModelInfo.fields.fieldsRel {
					switch ffi.fieldType {
					case RelOneToOne, RelForeignKey:
						if ffi.relModelInfo == fi.relModelInfo {
							fi.reverseFieldInfoTwo = ffi
						}
						if ffi.relModelInfo == mi {
							fi.reverseField = ffi.name
							fi.reverseFieldInfo = ffi
						}
					}
				}
				if fi.reverseFieldInfoTwo == nil {
					err = fmt.Errorf("can not find m2m field for m2m model `%s`, ensure your m2m model defined correct",
						fi.relThroughModelInfo.fullName)
					goto end
				}
			}
		}
	}

	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.fieldsReverse {
			switch fi.fieldType {
			case RelReverseOne:
				found := false
			mForA:
				for _, ffi := range fi.relModelInfo.fields.fieldsByType[RelOneToOne] {
					if ffi.relModelInfo == mi {
						found = true
						fi.reverseField = ffi.name
						fi.reverseFieldInfo = ffi

						ffi.reverseField = fi.name
						ffi.reverseFieldInfo = fi
						break mForA
					}
				}
				if !found {
					err = fmt.Errorf("reverse field `%s` not found in model `%s`", fi.fullName, fi.relModelInfo.fullName)
					goto end
				}
			case RelReverseMany:
				found := false
			mForB:
				for _, ffi := range fi.relModelInfo.fields.fieldsByType[RelForeignKey] {
					if ffi.relModelInfo == mi {
						found = true
						fi.reverseField = ffi.name
						fi.reverseFieldInfo = ffi

						ffi.reverseField = fi.name
						ffi.reverseFieldInfo = fi

						break mForB
					}
				}
				if !found {
				mForC:
					for _, ffi := range fi.relModelInfo.fields.fieldsByType[RelManyToMany] {
						conditions := fi.relThrough != "" && fi.relThrough == ffi.relThrough ||
							fi.relTable != "" && fi.relTable == ffi.relTable ||
							fi.relThrough == "" && fi.relTable == ""
						if ffi.relModelInfo == mi && conditions {
							found = true

							fi.reverseField = ffi.reverseFieldInfoTwo.name
							fi.reverseFieldInfo = ffi.reverseFieldInfoTwo
							fi.relThroughModelInfo = ffi.relThroughModelInfo
							fi.reverseFieldInfoTwo = ffi.reverseFieldInfo
							fi.reverseFieldInfoM2M = ffi
							ffi.reverseFieldInfoM2M = fi

							break mForC
						}
					}
				}
				if !found {
					err = fmt.Errorf("reverse field for `%s` not found in model `%s`", fi.fullName, fi.relModelInfo.fullName)
					goto end
				}
			}
		}
	}

end:
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}