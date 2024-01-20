// Copyright 2023 Kesuaheli
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"database/sql"
	"fmt"
	logger "log"
	"time"

	// mysql driver used for database
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

var log = logger.New(logger.Writer(), "[Database] ", logger.LstdFlags|logger.Lmsgprefix)
var db *sql.DB

type connectionConfig struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
}

func (c connectionConfig) String() string {
	return fmt.Sprintf("user='%s', host:port='%s:%d', database='%s'", c.User, c.Host, c.Port, c.Database)
}

// Connect sets the connection to the configured database
func Connect() {
	log.Println("Connecting to Database...")

	// setting default values
	config := connectionConfig{
		Host: "localhost",
		Port: 3306,
	}
	err := viper.UnmarshalKey("mysql", &config)
	if err != nil {
		log.Fatalf("Could not read msql connection data from config: %v", err)
	}

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", config.User, config.Password, config.Host, config.Port, config.Database)

	db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatalf("Could not open database connection: %v", err)
	}

	db.SetConnMaxLifetime(3 * time.Minute)

	err = db.Ping()
	if err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	log.Printf("Connected to database %s@%s:%d/%s\n", config.User, config.Host, config.Port, config.Database)
}

// Close closes the database and prevents new queries from starting.
// Close then waits for all queries that have started processing on the server
// to finish.
//
// It is rare to Close a DB, as the DB handle is meant to be
// long-lived and shared between many goroutines.
func Close() {
	db.Close()
	log.Println("Closed connection to database")
}

// Ping verifies a connection to the database is still alive,
// establishing a connection if necessary.
func Ping() (err error) {
	return db.Ping()
}

// Exec executes a query without returning any rows.
// The args are for any placeholder parameters in the query.
func Exec(query string, args ...any) (result sql.Result, err error) {
	return db.Exec(query, args...)
}

// Query executes a query that returns rows, typically a SELECT.
// The args are for any placeholder parameters in the query.
func Query(query string, args ...any) (rows *sql.Rows, err error) {
	return db.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value. Errors are deferred until
// Row's Scan method is called.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
func QueryRow(query string, args ...any) (row *sql.Row) {
	return db.QueryRow(query, args...)
}
