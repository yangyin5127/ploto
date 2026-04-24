package ploto

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type createUser struct {
	UserID int64  `db:"user_id,primary,auto"`
	Name   string `db:"name"`
	Age    int    `db:"age"`
}

type createUserWithCustomPK struct {
	AccountNo int64  `db:"account_no,primary,auto"`
	Name      string `db:"name"`
}

type createUserWithImplicitID struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type createOnlyID struct {
	ID int64 `db:"id"`
}

func TestDBCreateAssignsAutoPrimary(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO `users` \\(`name`,`age`\\) VALUES \\(\\?,\\?\\)").
		WithArgs("alice", 18).
		WillReturnResult(sqlmock.NewResult(12, 1))

	db := &DB{DB: mockDB, dialector: "mysql"}
	user := &createUser{Name: "alice", Age: 18}

	result, err := db.Create("users", user)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId error: %v", err)
	}

	if lastInsertID != 12 {
		t.Fatalf("LastInsertId should be 12")
	}

	if user.UserID != 12 {
		t.Fatalf("UserID should be assigned after create")
	}
}

func TestDBCreateSupportsCustomPrimaryColumnName(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO `accounts` \\(`name`\\) VALUES \\(\\?\\)").
		WithArgs("bob").
		WillReturnResult(sqlmock.NewResult(101, 1))

	db := &DB{DB: mockDB, dialector: "mysql"}
	account := &createUserWithCustomPK{Name: "bob"}

	if _, err := db.Create("accounts", account); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	if account.AccountNo != 101 {
		t.Fatalf("custom primary key should be assigned after create")
	}
}

func TestDBCreateUsesMSSQLOutputForAutoPrimary(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	rows := sqlmock.NewRows([]string{"user_id"}).AddRow(33)
	mock.ExpectQuery("INSERT INTO \\[users\\] \\(\\[name\\],\\[age\\]\\) OUTPUT INSERTED.\\[user_id\\] VALUES \\(@p1,@p2\\)").
		WithArgs("carol", 20).
		WillReturnRows(rows)

	db := &DB{DB: mockDB, dialector: "sqlserver"}
	user := &createUser{Name: "carol", Age: 20}

	result, err := db.Create("users", user)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId error: %v", err)
	}

	if lastInsertID != 33 {
		t.Fatalf("LastInsertId should be 33")
	}

	if user.UserID != 33 {
		t.Fatalf("UserID should be assigned after create")
	}
}

func TestDBCreateTreatsIDColumnAsImplicitAutoPrimary(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO `users` \\(`name`\\) VALUES \\(\\?\\)").
		WithArgs("dave").
		WillReturnResult(sqlmock.NewResult(56, 1))

	db := &DB{DB: mockDB, dialector: "mysql"}
	user := &createUserWithImplicitID{Name: "dave"}

	if _, err := db.Create("users", user); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	if user.ID != 56 {
		t.Fatalf("implicit id column should be assigned after create")
	}
}

func TestDBCreateUsesMySQLEmptyValuesSyntaxWhenOnlyImplicitIDExists(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO `users` \\(\\) VALUES \\(\\)").
		WillReturnResult(sqlmock.NewResult(78, 1))

	db := &DB{DB: mockDB, dialector: "mysql"}
	user := &createOnlyID{}

	if _, err := db.Create("users", user); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	if user.ID != 78 {
		t.Fatalf("implicit id should be assigned after create")
	}
}

func TestDBUpdateRequiresWhereClause(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	db := &DB{DB: mockDB, dialector: "mysql"}
	user := &createUserWithImplicitID{ID: 1, Name: "alice"}

	if _, err := db.Update("users", user, "   "); err != errUpdateWhereRequired {
		t.Fatalf("Update should require non-empty where clause")
	}
}

func TestDBUpdateSkipsImplicitIDColumnInSet(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectExec("UPDATE `users` SET `name`=\\? WHERE id=\\?").
		WithArgs("alice", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	db := &DB{DB: mockDB, dialector: "mysql"}
	user := &createUserWithImplicitID{ID: 1, Name: "alice"}

	result, err := db.Update("users", user, "id=?", 1)
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected error: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("RowsAffected should be 1")
	}
}

func TestDBUpdateSkipsExplicitPrimaryColumnInSet(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectExec("UPDATE `users` SET `name`=\\?,`age`=\\? WHERE user_id=\\?").
		WithArgs("alice", 18, 12).
		WillReturnResult(sqlmock.NewResult(0, 1))

	db := &DB{DB: mockDB, dialector: "mysql"}
	user := &createUser{UserID: 12, Name: "alice", Age: 18}

	if _, err := db.Update("users", user, "user_id=?", 12); err != nil {
		t.Fatalf("Update error: %v", err)
	}
}

func TestDBUpdateColumns(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectExec("UPDATE `users` SET `age`=\\?,`name`=\\? WHERE id=\\?").
		WithArgs(20, "alice", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	db := &DB{DB: mockDB, dialector: "mysql"}
	result, err := db.UpdateColumns("users", map[string]any{
		"name": "alice",
		"age":  20,
	}, "id=?", 1)
	if err != nil {
		t.Fatalf("UpdateColumns error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected error: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("RowsAffected should be 1")
	}
}

func TestDBUpdateColumnsRequiresWhereClause(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	db := &DB{DB: mockDB, dialector: "mysql"}
	if _, err := db.UpdateColumns("users", map[string]any{"name": "alice"}, ""); err != errUpdateWhereRequired {
		t.Fatalf("UpdateColumns should require non-empty where clause")
	}
}

func TestDBUpdateColumnsRequiresColumns(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer mockDB.Close()

	db := &DB{DB: mockDB, dialector: "mysql"}
	if _, err := db.UpdateColumns("users", map[string]any{}, "id=?", 1); err == nil {
		t.Fatalf("UpdateColumns should require non-empty columns")
	}
}
