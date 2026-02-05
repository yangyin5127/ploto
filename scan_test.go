package ploto

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type JSONData struct {
	Data string `json:"data"`
}

func (j *JSONData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)

	default:
		return nil
	}
	if len(bytes) == 0 {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

type UsersWithJSON struct {
	Id   int64     `db:"id"`
	Name string    `db:"name"`
	JSON *JSONData `db:"json_data"`
}

type Users struct {
	Id          int64  `db:"id"`
	CreatedTime string `db:"created_time"`
	UpdatedTime string `db:"updated_time"`
	Name        string `db:"name"`
}

type UsersEx struct {
	Users
}

func TestScanMap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dataRows := sqlmock.NewRows([]string{"id", "name", "created_time", "updated_time"}).
		AddRow(1, "1111", "2021-10-01 00:00:00", "2021-10-01 00:00:00")
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id<?").WithArgs(1).WillReturnRows(dataRows)

	rows, err := db.Query("SELECT id,name,created_time,updated_time FROM users WHERE id = 1", 1)
	if err != nil {
		t.Errorf("error '%s' was not expected while retrieving mock rows", err)
	}
	defer rows.Close()

	if rows.Next() {
		m := make(map[string]interface{})

		err = Scan(rows, &m)
		t.Logf("m result:%+v", m)

		if v, _ := m["id"].(int64); v != 1 {
			t.Fatalf("scan map error")
		}

	}

}

func TestScanStruct(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dataRows := sqlmock.NewRows([]string{"id", "name", "created_time", "updated_time"}).
		AddRow(1, "1111", "2021-10-01 00:00:00", "2021-10-01 00:00:00")
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id<?").WithArgs(1).WillReturnRows(dataRows)

	rows, err := db.Query("SELECT id,name,created_time,updated_time FROM users WHERE id = 1", 1)
	if err != nil {
		t.Errorf("error '%s' was not expected while retrieving mock rows", err)
	}
	defer rows.Close()

	if rows.Next() {

		var user Users

		err = Scan(rows, &user)
		t.Logf("user result:%+v", user)

		if user.Id != 1 {
			t.Fatalf("scan user struct error")
		}

	}

}

func TestScanStructs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dataRows := sqlmock.NewRows([]string{"id", "name", "created_time", "updated_time"}).
		AddRow(1, "1111", "2021-10-01 00:00:00", "2021-10-01 00:00:00")
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id<?").WithArgs(1).WillReturnRows(dataRows)

	rows, err := db.Query("SELECT id,name,created_time,updated_time FROM users WHERE id = 1", 1)
	if err != nil {
		t.Errorf("error '%s' was not expected while retrieving mock rows", err)
	}
	defer rows.Close()
	var user []Users

	err = ScanResult(rows, &user)
	t.Logf("user result:%+v", user)

}

func TestScanScalar(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dataRows := sqlmock.NewRows([]string{"cnt"}).
		AddRow(1)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id<?").WithArgs(1).WillReturnRows(dataRows)

	rows, err := db.Query("SELECT count(1) as cnt FROM users WHERE id = 1", 1)
	if err != nil {
		t.Errorf("error '%s' was not expected while retrieving mock rows", err)
	}
	defer rows.Close()

	defer rows.Close()

	if rows.Next() {
		var count int32

		err = Scan(rows, &count)

		if count != 1 {
			t.Fatalf("scan scalar error")
		}
		// fmt.Printf("cnt = %d err:%+v", count, err)

	}
}

func TestScanSlices(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dataRows := sqlmock.NewRows([]string{"id", "name", "created_time", "updated_time"}).
		AddRow(1, "1111", "2021-10-01 00:00:00", "2021-10-01 00:00:00")
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id<?").WithArgs(1).WillReturnRows(dataRows)

	rows, err := db.Query("SELECT id,name,created_time,updated_time FROM users WHERE id = 1", 1)
	if err != nil {
		t.Errorf("error '%s' was not expected while retrieving mock rows", err)
	}
	defer rows.Close()
	var user []*Users

	err = ScanResult(rows, &user)
	t.Logf("user result:%+v", user[0])

}

func TestScanSlices2(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dataRows := sqlmock.NewRows([]string{"id", "name", "created_time", "updated_time"}).
		AddRow(1, "1111", "2021-10-01 00:00:00", "2021-10-01 00:00:00")
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id<?").WithArgs(1).WillReturnRows(dataRows)

	rows, err := db.Query("SELECT id,name,created_time,updated_time FROM users WHERE id = 1", 1)
	if err != nil {
		t.Errorf("error '%s' was not expected while retrieving mock rows", err)
	}
	defer rows.Close()
	var user []*UsersEx

	err = ScanResult(rows, &user)
	if len(user) == 0 {
		t.Fatalf("scan userex struct error")
	}
	if user[0].CreatedTime != "2021-10-01 00:00:00" {
		t.Fatalf("scan userex struct error")
	}
	t.Logf("userex result:%+v", user[0])

}

func TestScanCustomScanner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	jsonStr := `{"data":"test"}`
	jsonStr2 := `` // to test scan error handling
	dataRows := sqlmock.NewRows([]string{"id", "name", "json_data"}).
		AddRow(1, "testuser", jsonStr).
		AddRow(2, "testuser2", jsonStr2)
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id<?").WithArgs(1).WillReturnRows(dataRows)

	rows, err := db.Query("SELECT id,name,json_data FROM users WHERE id <= 2", 1)
	if err != nil {
		t.Errorf("error '%s' was not expected while retrieving mock rows", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user UsersWithJSON

		err = Scan(rows, &user)
		if err != nil {
			t.Fatalf("scan custom scanner error: %v", err)
		}
		t.Logf("user with json result:%+v, json:%+v", user, user.JSON)

	}
}
