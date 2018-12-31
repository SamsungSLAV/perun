/*
 *  Copyright (c) 2018 Samsung Electronics Co., Ltd All Rights Reserved
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License
 */

package db

import (
	"database/sql"
	"sync"

	// Initialize the sqlite3 driver for database/sql package.
	_ "github.com/mattn/go-sqlite3"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/slav/logger"
)

// instance represents a perun's DB module instance.
type instance struct {
	// path points to database file.
	path string
	// connection is a database connections.
	connection *sql.DB
	// schema defines schema of the database.
	schema schema
	// mutex protects concurent access to the database.
	mutex sync.Locker
}

// NewDB creates a new instance of perun's database structure.
func NewDB(databaseFile string) (DB, error) {
	db := &instance{
		path:  databaseFile,
		mutex: new(sync.Mutex),
	}
	db.schema.define()

	logger.WithProperty("db", db.path).Notice("New database connection created.")
	return db, nil
}

// prepare prepares database to perun's usage by creating schema and initializing meta data.
func (db *instance) prepare(tx *sql.Tx) error {
	// Create schema in database.
	err := db.schema.create(tx)
	if err != nil {
		logger.WithError(err).Error("Failed to create schema.")
		return err
	}

	// Initialize meta data.
	err = db.schema.initialize(tx)
	return err
}

// Start connects Perun to the database and prepares schema in database.
func (db *instance) Start() error {
	// Open connection to database.
	dataSource := "file:" + db.path
	dbConnection, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		logger.WithError(err).WithProperty("db", db.path).Critical("Cannot open database.")
		return err
	}
	db.connection = dbConnection

	// Prepare schema.
	err = db.runInTransaction(db.prepare)
	if err != nil {
		logger.WithError(err).WithProperty("db", db.path).Critical("Database schema cannot be set.")
		// Closing database connection. Error will be logged by Close. There is more to be done.
		_ = db.Close() //nolint:gosec
		return err
	}
	logger.WithProperty("db", db.path).Notice("Database connection functional.")
	return nil
}

// Close disconnects from database.
func (db *instance) Close() error {
	if db.connection != nil {
		err := db.connection.Close()
		if err != nil {
			logger.WithError(err).WithProperty("db", db.path).Error("Problem closing database.")
			return err
		}
	}
	logger.WithProperty("db", db.path).Notice("Database connection closed.")
	return nil
}

// UpdateImage updates database with image information.
func (db *instance) UpdateImage(info *perun.ImageFileInfo) error {
	i := ImageInfo(*info)
	err := db.runInTransaction(i.insert)
	return err
}

// runInTransaction runs an operation
func (db *instance) runInTransaction(f func(*sql.Tx) error) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	tx, err := db.connection.Begin()
	if err != nil {
		logger.WithError(err).WithProperty("db", db.path).Error("Cannot open new transaction.")
		return err
	}
	defer func() {
		// There are 2 cases when Rollback doesn't succeed:
		// 1) when former tx.Commit has completed transaction. Sych error must be ignored by design.
		// 2) when transaction should be rolled back, but this operation fails. This situation
		//    happen only when there was an error during processing operation. That error will be
		//    returned by the runInTransaction method.
		_ = tx.Rollback() //nolint:gosec
	}()

	err = f(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		logger.WithError(err).WithProperty("db", db.path).Error("Cannot commit the transaction.")
		return err
	}
	return nil
}
