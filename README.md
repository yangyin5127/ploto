# ploto

A go Library for scan database/sql rows to struct、slice、other types, and support SQL logging

extensions to golang's database/sql

## 功能

- Scan rows, 支持struct,slice,map,其他基本类型
- 数据库配置连接管理
- sql日志
- Query/QueryContext
- QueryRow/QueryRowContext
- `Create` 
- `Update` 
- `UpdateColumns`


## 说明

ploto 是对 `database/sql` 的轻量扩展，不引入完整 ORM 的模型管理、关联关系、自动迁移等能力，重点是补足常用的数据读写辅助。

- 保留 `database/sql` 的原生使用方式，底层仍然直接基于 `*sql.DB` / `*sql.Tx`
- `Query/QueryContext` 结果 `Scan` 支持 `*Slice`、`*Struct`、`*Map`、`*int` 等基本类型
- `QueryRow/QueryRowContext` 结果 `Scan` 支持 `*Struct`、`*Map`、`*int` 等基本类型，结果为空返回 `sql.ErrNoRows`
- `Create` 支持根据 struct 的 `db` tag 生成 insert SQL，并在自增主键场景下回填字段
- `Update` 支持根据 struct 的 `db` tag 生成 update SQL，且要求必须传入 `WHERE` 条件
- `UpdateColumns` 支持通过 `map[string]any` 更新部分字段，且要求必须传入 `WHERE` 条件
- `Tx` 保留和 `DB` 一致的 `Create`、`Update`、`UpdateColumns`、`Query`、`Exec` 能力，并增加 `transaction_id` 日志





## Using

### 配合多数据库管理一起使用

```golang
package main

import (
    "encoding/json"
    "fmt"
    "github.com/yangyin5127/ploto"
     _ "github.com/go-sql-driver/mysql"
)

//Sql日志输出
type MyStdLogger struct {
}

func (m *MyStdLogger) Info(ctx context.Context, format string, v ...interface{}) {
	//...
}
func (m *MyStdLogger) Debug(ctx context.Context, format string, v ...interface{}) {
	//....
}

func (m *MyStdLogger) Warn(ctx context.Context, format string, v ...interface{}) {
	//...
}

func (m *MyStdLogger) Error(ctx context.Context, format string, v ...interface{}) {
	//....
}


func getConfig() (config Configs) {
    testConfig := `{"mysql": {
        "clients": {
            "test":{
                "host": "127.0.0.1",
                "port": 3306,
                "user": "root",
                "password": "root",
                "database": "test"
            }
        },
        "default": {
            "port": 3306,
            "dialect": "mysql",
            "pool": {
                "maxIdleConns": 2,
                "maxLeftTime": 60000, 
                "maxOpenConns": 5
            },
            "dialectOptions": {
                "parseTime":true,
                "multiStatements": true,
                "writeTimeout": "3000ms",
                "readTimeout": "3000ms",
                "timeout":"3000ms",
				"parseTime": true,
				"loc":"Local",
            }   
        }
    }}`

    var conf Configs

    json.Unmarshal([]byte(testConfig), &conf)

    // fmt.Printf("conf %+v", conf)
    return conf

}

type User struct {
    Id          int64  `db:"id"`
    Name        string `db:"name"`
    CreatedTime string `db:"created_time"`
    UpdatedTime string `db:"updated_time"`
}

type Configs struct {
    Mysql ploto.DialectConfig `json:"mysql"`
   // Mssql ploto.DialectConfig `json:"mssql"`
}

func main() {

    configs := getConfig()
	defaultLogger := &MyStdLogger{}

    db, err := ploto.Open(configs.Mysql, defaultLogger)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    
    var users []User
    err = db.Use("test").Query("select * from users where id<100").Scan(&users)
    if err != nil {
        panic(err)
    }
    fmt.Printf("users %+v", users)

	//Exec....
	result, err := db.Use("test").Exec("update users set name=? where  id=?","xxx",1)
    if err != nil {
		//...
        panic(err)
    }
	
	
	//Exec....
	result, err := db.Use("test").Exec("insert uesrs(name,created_time) values(?,now())","xxx")
    if err != nil {
		//...
        panic(err)
    }

	//Create....
	type CreateUser struct {
		UserID      int64  `db:"user_id,primary,auto"` // db:"column_name[,primary][,auto]"
		Name        string `db:"name"`
		CreatedTime string `db:"created_time"`
	}

	newUser := &CreateUser{
		Name:        "xxx",
		CreatedTime: "2024-01-01 00:00:00",
	}

	_, err = db.Use("test").Create("users", newUser)
	if err != nil {
		panic(err)
	}

	fmt.Printf("new user id=%d\n", newUser.UserID)

	//Update....
	newUser.Name = "yyy"
	_, err = db.Use("test").Update("users", newUser, "user_id=?", newUser.UserID)
	if err != nil {
		panic(err)
	}

	//UpdateColumns....
	_, err = db.Use("test").UpdateColumns("users", map[string]any{
		"name": "zzz",
	}, "user_id=?", newUser.UserID)
	if err != nil {
		panic(err)
	}

}

```

### Create 约定

- 仅处理带 `db` tag 的导出字段，`db:"-"` 会忽略
- `db` tag 第一个值是列名，例如 `db:"user_id"`
- 可选项 `primary`/`pk` 标记主键，`auto` 标记自增
- 自增主键字段在零值时会自动从 insert 字段里排除，create 成功后回填到对象
- 如果列名是 `id`，即使不写 `primary,auto` 也会按自增主键兼容处理
- 主键列名不要求是 `id`，例如 `db:"account_no,primary,auto"` 也支持

### Update 约定

- 仅处理带 `db` tag 的导出字段
- `where` 条件必填，空字符串会直接返回错误，防止更新全表
- 主键字段不会进入 `SET`
- `db:"id"` 会按主键兼容处理，不会被更新

```go
type User struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Age  int    `db:"age"`
}

user := &User{
	ID:   1,
	Name: "alice",
	Age:  20,
}

_, err := db.Update("users", user, "id=?", user.ID)
```

生成 SQL 类似：
```sql
UPDATE `users` SET `name`=?,`age`=? WHERE id=?
```

### UpdateColumns 约定

- 适合局部字段更新，参数为 `map[string]any`
- `map` 的 key 直接作为数据库列名使用，不走 `db` tag 映射
- `where` 条件必填，空字符串会直接返回错误，防止更新全表
- `columns` 为空时会直接返回错误
- 为了保证 SQL 稳定，更新列会按 key 排序生成

```go
_, err := db.UpdateColumns("users", map[string]any{
	"name": "alice",
	"age":  20,
}, "id=?", 1)
```

生成 SQL 类似：
```sql
UPDATE `users` SET `age`=?,`name`=? WHERE id=?
```


### 只用Scan功能

> 支持对rows结果转化到struct,slice，int等

struct定义字段tag为db
```go
type User struct {
    Id          int64  `db:"id"`
    Name        string `db:"name"`
    CreatedTime string `db:"created_time"`
    UpdatedTime string `db:"updated_time"`
}
```
```golang

package main

import (
	"database/sql"
	"fmt"
	"github.com/yangyin5127/ploto"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "user:password@/database")
	if err != nil {
		panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
	}
	defer db.Close()

	//scan rows to slices
	var users []User
	rows, err = db.Query("select * from users where id<100")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		err := ploto.Scan(rows, &user)
		users = append(users, user)
	}


	//ScanResult等同上代码
	var users []User
	rows, err = db.Query("select * from users where id<100")
	if err != nil {
		panic(err)
	}

	//No need to Close
	err := ploto.ScanResult(rows, &users)

	//.....
	// select count(1) as cnt from users

	if rows.Next() {
		var a int64
		ploto.Scan(rows,&a)
	}
	//.....

	// select * from users where id=1

	if rows.Next() {
		var user User 
		ploto.Scan(rows,&user)
	}
	//.....
}

```

## 数据库配置

配置支持多数据库连接，格式如下：

### mysql 
```json
{"mysql": {
		"clients": {
			"test":{
				"host": "127.0.0.1",
				"port": 3307,
				"user": "test",
				"password": "asfasdf@#sddfsdf",
				"database": "test"
			}
		},
		"default": {
			"port": 3306,
			"dialect": "mysql",
			"pool": {
				"maxIdleConns": 2,
				"maxLeftTime": 60000, 
				"maxOpenConns": 5
			},
			"dialectOptions": {
				"parseTime":"true",
				"multiStatements": "true",
				"writeTimeout": "3000ms",
				"readTimeout": "3000ms",
				"timeout":"3000ms",
				"parseTime": "true",
				"loc":"Local",

			}	
		}
	}}
```
更多dialectOptions参数见: https://github.com/go-sql-driver/mysql#parameters
### mssql

```
{"mssql": {
		"clients": {
	 
			"test":{
				"host": "127.0.0.1",
				"user": "sa",
				"password": "test123",
				"database": "test",
				"pool": {
					"maxIdleConns": 20,
					"maxLeftTime": 60000,
					"maxOpenConns": 50
				},
				"dialectOptions": {
					"dial timeout": "10"

				}
			}
		},
		"default": {
			"port": 1433,
			"dialect": "sqlserver", //or mssql
			"pool": {
				"maxIdleConns": 2,
				"maxLeftTime": 60000,
				"maxOpenConns": 5
			},
			"dialectOptions": {
				"dial timeout": "3"
			}
		}
	}}
```
更多dialectOptions 参数见:https://github.com/denisenkom/go-mssqldb#connection-parameters-and-dsn
