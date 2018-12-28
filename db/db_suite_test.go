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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}

func getRevision(tx *sql.Tx) int {
	var rev int
	err := tx.QueryRow(`SELECT value FROM meta WHERE key = 'revision'`).Scan(&rev)
	if err != nil {
		return -1
	}
	return rev
}

func setRevision(tx *sql.Tx, rev int) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO meta VALUES ('revision', $1)`, rev)
	return err
}
