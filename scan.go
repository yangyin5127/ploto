package ploto

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// valuesToMap  sql values转化为map
func valuesToMap(mapValue map[string]interface{}, values []interface{}, columns []string) {
	for idx, column := range columns {

		reflectValue := reflect.ValueOf(values[idx])
		if !reflectValue.IsValid() {
			mapValue[column] = nil
			continue
		}
		mapValue[column] = reflect.Indirect(reflectValue).Interface()

		if valuer, ok := mapValue[column].(driver.Valuer); ok {
			mapValue[column], _ = valuer.Value()
		} else if b, ok := mapValue[column].(sql.RawBytes); ok {
			mapValue[column] = string(b)
		} else {
			// mapValue[column] = string(b)
		}

	}
}

// initStructFieldTags
func initStructFieldTags(item reflect.Value, fieldTagMap *map[string]reflect.Value) {

	typ := item.Type() //value’s type
	for i := 0; i < item.NumField(); i++ {
		field := typ.Field(i)
		// support anonymous field, struct embedding
		if field.Anonymous {
			initStructFieldTags(item.Field(i), fieldTagMap)
			continue
		}
		tag, ok := typ.Field(i).Tag.Lookup("db")

		if ok && tag != "" {
			(*fieldTagMap)[tag] = item.Field(i)
		}
	}
}

// initStructValues
func initStructValues(item reflect.Value, columns []string, values []interface{}) {
	fieldTagMap := make(map[string]reflect.Value, len(columns))
	initStructFieldTags(item, &fieldTagMap)

	scannerType := reflect.TypeOf((*sql.Scanner)(nil)).Elem()

	for i, column := range columns {
		var fieldValue reflect.Value
		if v, ok := fieldTagMap[column]; ok {
			fieldValue = v
		} else {
			fieldValue = item.FieldByName(cases.Title(language.Und, cases.NoLower).String(column))
		}

		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			values[i] = new(interface{})
		} else {
			if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Implements(scannerType) {
				values[i] = fieldValue.Interface()
			} else if fieldValue.Type().Implements(scannerType) {
				values[i] = fieldValue.Addr().Interface()
			} else {
				values[i] = fieldValue.Addr().Interface()
			}
		}
	}

}

func ScanResult(rows *sql.Rows, dest interface{}) error {

	defer rows.Close()

	destType := reflect.TypeOf(dest)

	if k := destType.Kind(); k != reflect.Ptr {
		return fmt.Errorf("%s must be a pointer:", k.String())
	}

	sliceType := destType.Elem()

	if reflect.Slice == sliceType.Kind() {
		//slice
		err := ScanSlice(rows, dest)
		return err

	} else {
		//other
		if rows.Next() {
			err := Scan(rows, dest)
			return err
		}
	}

	return nil

}

func ScanSlice(rows *sql.Rows, dest interface{}) error {

	//columns
	columns, _ := rows.Columns()

	destType := reflect.TypeOf(dest)
	sliceType := destType.Elem()

	//slice
	isPtr := false
	itemType := sliceType.Elem()
	if itemType.Kind() == reflect.Ptr {
		itemType = itemType.Elem()
		isPtr = true
	}

	sliceVal := reflect.Indirect(reflect.ValueOf(dest))

	for rows.Next() {

		values := make([]interface{}, len(columns))

		sliceItem := reflect.New(itemType).Elem()

		initStructValues(sliceItem, columns, values)

		err := rows.Scan(values...)

		if err != nil {
			return err
		}

		if isPtr {
			sliceVal.Set(reflect.Append(sliceVal, sliceItem.Addr()))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, sliceItem))
		}

	}

	return nil

}

// Scan
func Scan(rows *sql.Rows, dest interface{}) error {
	//columns
	columns, _ := rows.Columns()
	values := make([]interface{}, len(columns))
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	// destType := reflect.TypeOf(dest)

	// if k := destType.Kind(); k != reflect.Ptr {
	// 	return fmt.Errorf("%s must be a pointer:", k.String())
	// }

	switch dType := dest.(type) {

	case *int, *int8, *int16, *int32, *int64,
		*uint, *uint8, *uint16, *uint32, *uint64, *uintptr,
		*float32, *float64,
		*bool, *string, *time.Time,
		*sql.NullInt32, *sql.NullInt64, *sql.NullFloat64,
		*sql.NullBool, *sql.NullString, *sql.NullTime:

		err := rows.Scan(dType)

		return err
	case *map[string]interface{}:
		for i := 0; i < len(columns); i++ {
			//*T
			if columnTypes[i].ScanType() != nil {
				values[i] = reflect.New(columnTypes[i].ScanType()).Interface()
			} else {
				values[i] = new(interface{})
			}
		}
		err := rows.Scan(values...)
		mValue := dest.(*map[string]interface{})
		valuesToMap(*mValue, values, columns)

		return err
	default:
		//scan to struct
		destValue := reflect.ValueOf(dest).Elem() //destType.Elem()
		initStructValues(destValue, columns, values)

		err := rows.Scan(values...)
		return err

	}
}
