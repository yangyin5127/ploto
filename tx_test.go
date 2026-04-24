package ploto

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type MyStdLogger struct {
}

func (m *MyStdLogger) Info(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(fmt.Sprintf(format, v...))

}
func (m *MyStdLogger) Debug(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(fmt.Sprintf(format, v...))

}

func (m *MyStdLogger) Warn(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(fmt.Sprintf(format, v...))

}

func (m *MyStdLogger) Error(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(fmt.Sprintf(format, v...))
}

func TestTransactionCommit(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	mock.ExpectBegin()

	mock.ExpectExec("update users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	db := &DB{DB: mockDB}

	db.LogSql = true
	db.logger = &MyStdLogger{}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin with error %+v", err)
	}
	result, err := tx.Exec("update users set name='xxxx' WHERE id=1")

	if err != nil {
		t.Fatalf("update with error %+v", err)
	}

	affected, err := result.RowsAffected()

	t.Logf("update:%d", affected)
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Commit with error %+v", err)

	}

}

func TestTransactionRollback(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	mock.ExpectBegin()
	// mock.ExpectExec("update users").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("update users").WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()
	// mock.ExpectCommit()

	db := &DB{DB: mockDB}
	db.LogSql = true
	db.logger = &MyStdLogger{}

	tx, err := db.Begin()
	defer func() {

		err := tx.Rollback()
		t.Logf("rollback with %+v", err)

	}()

	if err != nil {
		t.Fatalf("Begin with error %+v", err)
	}

	result, err := tx.Exec("update users set name='xxxx' WHERE id=1")
	if err != nil {
		t.Logf("update with error %+v", err)
		return
	}

	affected, err := result.RowsAffected()

	if err != nil {
		t.Logf("Commit with error %+v", err)
		return

	}

	t.Logf("update:%d", affected)
	err = tx.Commit()
	t.Logf("commit with %+v", err)

}
