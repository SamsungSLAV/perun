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

// schema contains schema of whole perun's database.
type schema struct {
	elems []schemaElem
}

// add appends a single element to schema.
func (s *schema) add(e schemaElem) {
	s.elems = append(s.elems, e)
}

// addTable appends a table element to schema.
func (s *schema) addTable(name string, schema string) {
	s.add(schemaElem{elemType: table, name: name, schema: schema})
}

// addTrigger appends a table element to schema.
func (s *schema) addTrigger(name string, schema string) {
	s.add(schemaElem{elemType: trigger, name: name, schema: schema})
}

// define adds all database schema elements used by perun.
func (s *schema) define() {
	s.addTable("meta", metaCreate)
	s.addTable("images", imagesCreate)
	s.addTrigger("imagesFilter", imagesFilterCreate)
	s.addTrigger("imagesRevision", imagesRevisionCreate)
}

// create applies schema to the database provided by open transaction.
func (s *schema) create(tx *sql.Tx) error {
	var err error
	for _, e := range s.elems {
		err = e.create(tx)
		if err != nil {
			return err
		}
	}
	return nil
}

// initialize sets required meta data in database.
func (s *schema) initialize(tx *sql.Tx) error {
	// Initialize revision.
	_, err := tx.Exec(metaRevisionInit)
	if err != nil {
		logger.WithError(err).WithProperty("sql", metaRevisionInit).
			Error("Failed to initialize Revision.")
	}
	return err
}
