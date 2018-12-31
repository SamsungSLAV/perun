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
	"strings"
	"sync"

	"github.com/SamsungSLAV/perun/testutil"

	. "github.com/onsi/ginkgo"
	T "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	// Initialize the sqlite3 driver for database/sql package.
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("schemaElem", func() {
	const (
		createExistingTable   = "CREATE TABLE existingTable (key TEXT)"
		createExistingTrigger = "CREATE TRIGGER existingTrigger AFTER UPDATE ON existingTable" +
			" BEGIN VALUES (1); END"
		createTestTable   = "CREATE TABLE testTable (key INTEGER PRIMARY KEY, value INTEGER)"
		createTestTrigger = "CREATE TRIGGER testTrigger AFTER INSERT ON existingTable BEGIN" +
			" VALUES (2); END"
	)
	var (
		dbConnection  *sql.DB
		dbTransaction *sql.Tx
		lock          = new(sync.Mutex)
		invalidElem   = schemaElem{
			name:     "??*&*$^^",
			elemType: "(@($^))",
			schema:   "@$^#@$#",
		}
	)
	BeforeEach(func() {
		lock.Lock()
		defer lock.Unlock()

		var err error
		dbConnection, err = sql.Open(defaultSQLDriver, ":memory:")
		Expect(err).NotTo(HaveOccurred())
		Expect(dbConnection).NotTo(BeNil())

		dbTransaction, err = dbConnection.Begin()
		Expect(err).NotTo(HaveOccurred())

		_, err = dbTransaction.Exec(createExistingTable)
		Expect(err).NotTo(HaveOccurred())

		_, err = dbTransaction.Exec(createExistingTrigger)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		lock.Lock()
		defer lock.Unlock()

		err := dbTransaction.Rollback()
		Expect(err).NotTo(HaveOccurred())

		err = dbConnection.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("create", func() {
		T.DescribeTable("should create a new entity",
			func(elem schemaElem) {
				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := elem.create(dbTransaction)
					Expect(err).NotTo(HaveOccurred())

					var schema string
					err = dbTransaction.QueryRow(masterSelect, elem.elemType, elem.name).
						Scan(&schema)
					Expect(err).NotTo(HaveOccurred())
					Expect(schema).To(Equal(elem.schema))
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "New elem created.")).To(Equal(1))
			},
			T.Entry("table", schemaElem{
				name:     "testTable",
				elemType: table,
				schema:   createTestTable,
			}),
			T.Entry("trigger", schemaElem{
				name:     "testTrigger",
				elemType: trigger,
				schema:   createTestTrigger,
			}),
		)
		It("should return error if cannot ask master table", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					var err error
					dbTransaction, err = dbConnection.Begin()
					Expect(err).NotTo(HaveOccurred())
				}()

				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := dbTransaction.Rollback()
				Expect(err).NotTo(HaveOccurred())

				elem := invalidElem
				err = elem.create(dbTransaction)
				Expect(err).To(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Cannot ask master table for schema.")).To(Equal(1))
		})
		It("should return error if cannot elem's schema is invalid", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				elem := invalidElem
				err := elem.create(dbTransaction)
				Expect(err).To(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Failed to create an elem.")).To(Equal(1))
		})
		T.DescribeTable("should do nothing if entity exists and matches schema",
			func(elem schemaElem) {
				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := elem.create(dbTransaction)
					Expect(err).NotTo(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			},
			T.Entry("table", schemaElem{
				name:     "existingTable",
				elemType: table,
				schema:   createExistingTable,
			}),
			T.Entry("trigger", schemaElem{
				name:     "existingTrigger",
				elemType: trigger,
				schema:   createExistingTrigger,
			}),
		)
		T.DescribeTable("should return ErrSchemaMismatch if elem exists but schema doesn't match",
			func(elem schemaElem) {
				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := elem.create(dbTransaction)
					Expect(err).To(Equal(ErrSchemaMismatch))
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "Schema not matching currently existing one.")).
					To(Equal(1))
			},
			T.Entry("table", schemaElem{
				name:     "existingTable",
				elemType: table,
				schema:   createTestTable,
			}),
			T.Entry("trigger", schemaElem{
				name:     "existingTrigger",
				elemType: trigger,
				schema:   createTestTrigger,
			}),
		)
	})
})
