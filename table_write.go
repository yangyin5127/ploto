package ploto

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

var errCreateValueMustBeStruct = errors.New("value must be a struct or pointer to struct")
var errUpdateWhereRequired = errors.New("update where clause is required")

type createField struct {
	column     string
	value      any
	fieldValue reflect.Value
	primary    bool
	auto       bool
	omitInsert bool
}

// Create inserts value into table using db tags on exported struct fields.
//
// Example:
//
//	type User struct {
//		ID   int64  `db:"id"`
//		Name string `db:"name"`
//	}
//
//	user := &User{Name: "alice"}
//	_, err := db.Create("users", user)
func (db *DB) Create(table string, value any) (sql.Result, error) {
	return db.CreateContext(context.Background(), table, value)
}

// CreateContext inserts value into table using db tags on exported struct fields.
//
// Example:
//
//	ctx := context.Background()
//	_, err := db.CreateContext(ctx, "users", &User{Name: "alice"})
func (db *DB) CreateContext(ctx context.Context, table string, value any) (sql.Result, error) {
	query, args, autoPrimary, err := buildCreateStatement(db.dialector, table, value)
	if err != nil {
		return nil, err
	}
	return executeCreate(ctx, db.dialector, autoPrimary, query, args, db.ExecContext, db.QueryRowContext)
}

// Create inserts value into table within the current transaction.
//
// Example:
//
//	_, err := tx.Create("users", &User{Name: "alice"})
func (tx *Tx) Create(table string, value any) (sql.Result, error) {
	return tx.CreateContext(context.Background(), table, value)
}

// CreateContext inserts value into table within the current transaction.
//
// Example:
//
//	ctx := context.Background()
//	_, err := tx.CreateContext(ctx, "users", &User{Name: "alice"})
func (tx *Tx) CreateContext(ctx context.Context, table string, value any) (sql.Result, error) {
	dialect := ""
	if tx.DB != nil {
		dialect = tx.DB.dialector
	}
	query, args, autoPrimary, err := buildCreateStatement(dialect, table, value)
	if err != nil {
		return nil, err
	}
	return executeCreate(ctx, dialect, autoPrimary, query, args, tx.ExecContext, tx.QueryRowContext)
}

// Update updates table columns from value using db tags on exported struct fields.
// where is required to avoid updating all rows.
//
// Example:
//
//	user := &User{ID: 1, Name: "alice"}
//	_, err := db.Update("users", user, "id=?", user.ID)
func (db *DB) Update(table string, value any, where string, whereArgs ...any) (sql.Result, error) {
	return db.UpdateContext(context.Background(), table, value, where, whereArgs...)
}

// UpdateContext updates table columns from value using db tags on exported struct fields.
// where is required to avoid updating all rows.
//
// Example:
//
//	ctx := context.Background()
//	_, err := db.UpdateContext(ctx, "users", user, "id=?", user.ID)
func (db *DB) UpdateContext(ctx context.Context, table string, value any, where string, whereArgs ...any) (sql.Result, error) {
	query, args, err := buildUpdateStatement(db.dialector, table, value, where, whereArgs...)
	if err != nil {
		return nil, err
	}
	return db.ExecContext(ctx, query, args...)
}

// Update updates table columns from value within the current transaction.
// where is required to avoid updating all rows.
//
// Example:
//
//	_, err := tx.Update("users", user, "id=?", user.ID)
func (tx *Tx) Update(table string, value any, where string, whereArgs ...any) (sql.Result, error) {
	return tx.UpdateContext(context.Background(), table, value, where, whereArgs...)
}

// UpdateContext updates table columns from value within the current transaction.
// where is required to avoid updating all rows.
//
// Example:
//
//	ctx := context.Background()
//	_, err := tx.UpdateContext(ctx, "users", user, "id=?", user.ID)
func (tx *Tx) UpdateContext(ctx context.Context, table string, value any, where string, whereArgs ...any) (sql.Result, error) {
	dialect := ""
	if tx.DB != nil {
		dialect = tx.DB.dialector
	}
	query, args, err := buildUpdateStatement(dialect, table, value, where, whereArgs...)
	if err != nil {
		return nil, err
	}
	return tx.ExecContext(ctx, query, args...)
}

func (db *DB) UpdateColumns(table string, columns map[string]any, where string, whereArgs ...any) (sql.Result, error) {
	return db.UpdateColumnsContext(context.Background(), table, columns, where, whereArgs...)
}

func (db *DB) UpdateColumnsContext(ctx context.Context, table string, columns map[string]any, where string, whereArgs ...any) (sql.Result, error) {
	query, args, err := buildUpdateColumnsStatement(db.dialector, table, columns, where, whereArgs...)
	if err != nil {
		return nil, err
	}
	return db.ExecContext(ctx, query, args...)
}

func (tx *Tx) UpdateColumns(table string, columns map[string]any, where string, whereArgs ...any) (sql.Result, error) {
	return tx.UpdateColumnsContext(context.Background(), table, columns, where, whereArgs...)
}

func (tx *Tx) UpdateColumnsContext(ctx context.Context, table string, columns map[string]any, where string, whereArgs ...any) (sql.Result, error) {
	dialect := ""
	if tx.DB != nil {
		dialect = tx.DB.dialector
	}
	query, args, err := buildUpdateColumnsStatement(dialect, table, columns, where, whereArgs...)
	if err != nil {
		return nil, err
	}
	return tx.ExecContext(ctx, query, args...)
}

func buildCreateStatement(dialect, table string, value any) (string, []any, *createField, error) {
	if strings.TrimSpace(table) == "" {
		return "", nil, nil, errors.New("create table is required")
	}

	fields, autoPrimary, err := collectCreateFields(value)
	if err != nil {
		return "", nil, nil, err
	}

	query, args := buildInsertSQL(dialect, table, fields, autoPrimary)
	return query, args, autoPrimary, nil
}

func buildUpdateStatement(dialect, table string, value any, where string, whereArgs ...any) (string, []any, error) {
	if strings.TrimSpace(table) == "" {
		return "", nil, errors.New("update table is required")
	}
	if strings.TrimSpace(where) == "" {
		return "", nil, errUpdateWhereRequired
	}

	fields, _, err := collectCreateFields(value)
	if err != nil {
		return "", nil, err
	}

	setParts := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+len(whereArgs))
	for _, field := range fields {
		if isPrimaryLikeField(field) {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s=%s", quoteIdentifier(dialect, field.column), placeholderForDialect(dialect, len(args)+1)))
		args = append(args, field.value)
	}

	if len(setParts) == 0 {
		return "", nil, errors.New("update value has no columns to set")
	}

	where = rewriteWherePlaceholders(dialect, where, len(args))
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quoteIdentifier(dialect, table), strings.Join(setParts, ","), where)
	for _, arg := range whereArgs {
		args = append(args, arg)
	}
	return query, args, nil
}

func buildUpdateColumnsStatement(dialect, table string, columns map[string]any, where string, whereArgs ...any) (string, []any, error) {
	if strings.TrimSpace(table) == "" {
		return "", nil, errors.New("update table is required")
	}
	if strings.TrimSpace(where) == "" {
		return "", nil, errUpdateWhereRequired
	}
	if len(columns) == 0 {
		return "", nil, errors.New("update columns is required")
	}

	columnNames := make([]string, 0, len(columns))
	for name := range columns {
		if strings.TrimSpace(name) == "" {
			continue
		}
		columnNames = append(columnNames, name)
	}
	sort.Strings(columnNames)

	if len(columnNames) == 0 {
		return "", nil, errors.New("update columns is required")
	}

	setParts := make([]string, 0, len(columnNames))
	args := make([]any, 0, len(columnNames)+len(whereArgs))
	for _, name := range columnNames {
		setParts = append(setParts, fmt.Sprintf("%s=%s", quoteIdentifier(dialect, name), placeholderForDialect(dialect, len(args)+1)))
		args = append(args, columns[name])
	}

	where = rewriteWherePlaceholders(dialect, where, len(args))
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quoteIdentifier(dialect, table), strings.Join(setParts, ","), where)
	for _, arg := range whereArgs {
		args = append(args, arg)
	}
	return query, args, nil
}

func rewriteWherePlaceholders(dialect, where string, offset int) string {
	switch strings.ToLower(dialect) {
	case "mssql", "sqlserver":
		var b strings.Builder
		index := offset
		for _, ch := range where {
			if ch == '?' {
				index++
				b.WriteString(placeholderForDialect(dialect, index))
				continue
			}
			b.WriteRune(ch)
		}
		return b.String()
	default:
		return where
	}
}

func executeCreate(
	ctx context.Context,
	dialect string,
	autoPrimary *createField,
	query string,
	args []any,
	execFunc func(context.Context, string, ...any) (sql.Result, error),
	queryRowFunc func(context.Context, string, ...any) *RowResult,
) (sql.Result, error) {
	if autoPrimary != nil && shouldUseReturning(dialect) {
		dest := reflect.New(derefType(autoPrimary.fieldValue.Type())).Interface()
		if err := queryRowFunc(ctx, query, args...).Scan(dest); err != nil {
			return nil, err
		}
		insertedID := reflect.ValueOf(dest).Elem().Interface()
		if err := assignAutoPrimary(autoPrimary.fieldValue, insertedID); err != nil {
			return nil, err
		}
		return createResult{lastInsertID: insertedID, rowsAffected: 1}, nil
	}

	result, err := execFunc(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	if autoPrimary != nil {
		lastInsertID, err := result.LastInsertId()
		if err == nil {
			if err := assignAutoPrimary(autoPrimary.fieldValue, lastInsertID); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func collectCreateFields(value any) ([]createField, *createField, error) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil, nil, errCreateValueMustBeStruct
	}

	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil, errCreateValueMustBeStruct
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, nil, errCreateValueMustBeStruct
	}

	rt := rv.Type()
	fields := make([]createField, 0, rt.NumField())
	var autoPrimary *createField

	for i := 0; i < rt.NumField(); i++ {
		structField := rt.Field(i)
		if structField.PkgPath != "" {
			continue
		}

		tag := structField.Tag.Get("db")
		column, options := parseDBTag(tag)
		if column == "" {
			continue
		}

		fieldValue := rv.Field(i)
		field := createField{
			column:     column,
			value:      fieldValue.Interface(),
			fieldValue: fieldValue,
			primary:    options["primary"] || options["pk"] || options["primarykey"],
			auto:       options["auto"] || options["autoincrement"] || options["auto_increment"],
		}

		if shouldTreatAsAutoPrimary(field) && fieldValue.IsZero() {
			field.omitInsert = true

			if fieldValue.CanSet() {
				current := field
				current.primary = true
				current.auto = true
				autoPrimary = &current
			}
		}

		fields = append(fields, field)
	}

	if len(fields) == 0 {
		return nil, nil, errors.New("create value has no db tagged fields")
	}

	return fields, autoPrimary, nil
}

func shouldTreatAsAutoPrimary(field createField) bool {
	if field.primary && field.auto {
		return true
	}
	return strings.EqualFold(field.column, "id")
}

func isPrimaryLikeField(field createField) bool {
	if field.primary {
		return true
	}
	return strings.EqualFold(field.column, "id")
}

func parseDBTag(tag string) (string, map[string]bool) {
	if tag == "" || tag == "-" {
		return "", nil
	}

	parts := strings.Split(tag, ",")
	column := strings.TrimSpace(parts[0])
	if column == "" || column == "-" {
		return "", nil
	}

	options := make(map[string]bool, len(parts)-1)
	for _, part := range parts[1:] {
		opt := strings.ToLower(strings.TrimSpace(part))
		if opt == "" {
			continue
		}
		options[opt] = true
	}

	return column, options
}

func buildInsertSQL(dialect, table string, fields []createField, autoPrimary *createField) (string, []any) {
	columns := make([]string, 0, len(fields))
	placeholders := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields))

	for _, field := range fields {
		if field.omitInsert {
			continue
		}
		columns = append(columns, quoteIdentifier(dialect, field.column))
		placeholders = append(placeholders, placeholderForDialect(dialect, len(args)+1))
		args = append(args, field.value)
	}

	if len(columns) == 0 {
		if strings.ToLower(dialect) == "mysql" {
			return fmt.Sprintf("INSERT INTO %s () VALUES ()", quoteIdentifier(dialect, table)), args
		} else {
			return fmt.Sprintf("INSERT INTO %s DEFAULT VALUES", quoteIdentifier(dialect, table)), args
		}
	}

	if autoPrimary != nil && shouldUseReturning(dialect) {
		return fmt.Sprintf(
			"INSERT INTO %s (%s) OUTPUT INSERTED.%s VALUES (%s)",
			quoteIdentifier(dialect, table),
			strings.Join(columns, ","),
			quoteIdentifier(dialect, autoPrimary.column),
			strings.Join(placeholders, ","),
		), args
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(dialect, table),
		strings.Join(columns, ","),
		strings.Join(placeholders, ","),
	), args
}

func quoteIdentifier(dialect, name string) string {
	switch strings.ToLower(dialect) {
	case "mssql", "sqlserver":
		return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
	default:
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
}

func placeholderForDialect(dialect string, index int) string {
	switch strings.ToLower(dialect) {
	case "mssql", "sqlserver":
		return "@p" + strconv.Itoa(index)
	default:
		return "?"
	}
}

func shouldUseReturning(dialect string) bool {
	switch strings.ToLower(dialect) {
	case "mssql", "sqlserver":
		return true
	default:
		return false
	}
}

func assignAutoPrimary(field reflect.Value, value any) error {
	if !field.CanSet() {
		return nil
	}

	if value == nil {
		return nil
	}

	if field.Kind() == reflect.Ptr {
		elem := reflect.New(field.Type().Elem())
		if err := assignAutoPrimary(elem.Elem(), value); err != nil {
			return err
		}
		field.Set(elem)
		return nil
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := toInt64(value)
		if err != nil {
			return err
		}
		field.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := toInt64(value)
		if err != nil {
			return err
		}
		field.SetUint(uint64(v))
	case reflect.String:
		field.SetString(fmt.Sprint(value))
	default:
		rv := reflect.ValueOf(value)
		if rv.Type().AssignableTo(field.Type()) {
			field.Set(rv)
			return nil
		}
		if rv.Type().ConvertibleTo(field.Type()) {
			field.Set(rv.Convert(field.Type()))
			return nil
		}
		return fmt.Errorf("unsupported auto primary field type %s", field.Type())
	}

	return nil
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint:
		return int64(v), nil
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

type createResult struct {
	lastInsertID any
	rowsAffected int64
}

func (r createResult) LastInsertId() (int64, error) {
	return toInt64(r.lastInsertID)
}

func (r createResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}
