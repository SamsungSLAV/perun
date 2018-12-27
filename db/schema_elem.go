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

	"github.com/SamsungSLAV/slav/logger"
)

// schemaElemType defines type of database schema entity.
type schemaElemType = string

const (
	table   schemaElemType = "table"
	trigger schemaElemType = "trigger"
)

// schemaElem describes a single database schema entity.
type schemaElem struct {
	name     string         // name of the entity, e.g. name of the table.
	elemType schemaElemType // type of the entity (defined by one of above constants).
	schema   string         // schema is a complete schema creating command of the entity.
}

// create ensures that schema entity is created in the database provided by active transaction.
// If the entity already exists and it's schema creating command matches elem, no action is taken.
// If the entitty exists, but differs an error is returned.
// If the entitty does not exist in the database schema, it is created.
func (elem *schemaElem) create(tx *sql.Tx) error {
	var existingSchema string
	err := tx.QueryRow(masterSelect, elem.elemType, elem.name).Scan(&existingSchema)
	if err != nil {
		if err != sql.ErrNoRows {
			logger.WithError(err).WithProperty("schemaElem", *elem).
				Error("Cannot ask master table for schema.")
			return err
		}
	} else {
		// Elem with that type and name already exists in DB. Verify if schema matches.
		if existingSchema == elem.schema {
			// Existing schema is up-to-date.
			return nil
		}
		err = ErrSchemaMismatch
		logger.WithError(err).WithProperty("schemaElem", *elem).
			WithProperty("old_schema", existingSchema).
			Error("Schema not matching currently existing one.")
		return err
	}

	_, err = tx.Exec(elem.schema)
	if err != nil {
		logger.WithError(err).WithProperty("schemaElem", *elem).Error("Failed to create an elem.")
		return err
	}
	logger.WithProperty("schemaElem", *elem).Info("New elem created.")
	return nil
}
