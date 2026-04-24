package ploto

import (
	"context"
	"database/sql"

	// "reflect"
	// "strings"
	"errors"
	"time"
)

type Dialect struct {
	Clients map[string]*DB
	Configs DialectConfig
	logger  LoggerInterface
}

type DialectConfig struct {
	Clients map[string]*DialectClientOption `json:"clients"`
	Default *DialectClientOption            `json:"default"`
}

type DialectClientOptionPool struct {
	MaxIdleConns int `json:"maxIdleConns"`
	MaxLeftTime  int `json:"maxLeftTime"`
	MaxOpenConns int `json:"maxOpenConns"`
}

type DialectClientOption struct {
	Host           string                   `json:"host"`
	Port           int                      `json:"port"`
	User           string                   `json:"user"`
	Password       string                   `json:"password"`
	Database       string                   `json:"database"`
	Dialect        string                   `json:"dialect"`
	Logging        *bool                    `json:"logging"`
	Pool           *DialectClientOptionPool `json:"pool"`
	Charset        string                   `json:"charset"`
	DialectOptions map[string]string        `json:"dialectOptions"`
}

type DialectDSN interface {
	GetDialectDSN(database string, config *DialectClientOption) string
}

// CreateClient  create the db pool for  the database
func (dialect *Dialect) CreateClient(database string) (db *DB, err error) {

	config := dialect.getClientConfig(database)
	ctx := context.Background()
	var dsn DialectDSN = nil
	var dialector = config.Dialect
	var dbName = config.Database

	if len(dbName) == 0 {
		return nil, errors.New("invalid database config")
	}

	switch dialector {
	case "mssql", "sqlserver":
		dsn = Mssql{}
	case "mysql":
		dsn = Mysql{}
	default:
		dialect.logger.Error(ctx, "connect to mysql database %s with invalid dialect", dbName)
		return nil, err
	}

	dnsPath := dsn.GetDialectDSN(database, config)

	driverDB, err := sql.Open(dialector, dnsPath)
	if err != nil {
		dialect.logger.Error(ctx, "connect to mysql database %s error", dbName)
		return nil, err
	}

	maxIdleConns := 10
	maxLeftTime := 7200
	maxOpenConns := 50

	if config.Pool != nil && config.Pool.MaxIdleConns > 0 {
		maxIdleConns = config.Pool.MaxIdleConns
	}

	if config.Pool != nil && config.Pool.MaxLeftTime > 0 {
		maxLeftTime = config.Pool.MaxLeftTime
	}

	if config.Pool != nil && config.Pool.MaxOpenConns > 0 {
		maxOpenConns = config.Pool.MaxOpenConns
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	driverDB.SetMaxIdleConns(maxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	driverDB.SetMaxOpenConns(maxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	driverDB.SetConnMaxLifetime(time.Duration(maxLeftTime) * time.Millisecond)
	if err := driverDB.Ping(); err != nil {
		return nil, err
	}

	//set db to the clients
	db = &DB{DB: driverDB, dialector: dialector}

	if config.Logging != nil {
		db.LogSql = *config.Logging
	}

	// logger.Info("create mysql db %s client success", database)

	return db, err

}

// Use get the db's conn
func (dialect *Dialect) Use(database string) (db *DB) {

	client, _ := dialect.Clients[database]
	return client
}

// GetClientConfig get the client config
func (dialect *Dialect) getClientConfig(clientName string) (config *DialectClientOption) {

	// var config *DialectClientOption{}
	// for k, v := range dialect.Configs.Default {
	// 	config[k] = v
	// }

	clients := dialect.Configs.Clients
	if _, ok := clients[clientName]; !ok {
		return nil
	}

	//存在
	config = clients[clientName]

	if dialect.Configs.Default == nil {
		return config
	}

	if len(config.Dialect) == 0 {
		config.Dialect = dialect.Configs.Default.Dialect
	}

	// use default pool if client not set
	if config.Pool == nil {
		config.Pool = dialect.Configs.Default.Pool
	}

	if config.Logging == nil {
		config.Logging = dialect.Configs.Default.Logging
	}

	if config.DialectOptions == nil {
		config.DialectOptions = dialect.Configs.Default.DialectOptions
	}

	if config.Port == 0 {
		config.Port = dialect.Configs.Default.Port
	}

	if len(config.User) == 0 {
		config.User = dialect.Configs.Default.User
	}

	return config
}

// Close  Close the database
func (dialect *Dialect) Close() error {

	ctx := context.Background()
	for k, v := range dialect.Clients {
		err := v.Close()
		if err != nil {
			dialect.logger.Info(ctx, "close db %s error %+v", k, err)
			return err
		}
		dialect.logger.Info(ctx, "close db %s success", k)

	}
	return nil
}
