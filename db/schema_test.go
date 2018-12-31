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
	. "github.com/onsi/gomega"

	// Initialize the sqlite3 driver for database/sql package.
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("schema", func() {
	const (
		testName      = "testName"
		testSchema    = "testSchema"
		invalidString = "!@#$%^^&*("
	)
	var (
		s      *schema
		eTable = schemaElem{
			name:     testName,
			elemType: table,
			schema:   testSchema,
		}
		eTrigger = schemaElem{
			name:     testName,
			elemType: trigger,
			schema:   testSchema,
		}
	)

	BeforeEach(func() {
		s = &schema{}
		Expect(s).NotTo(BeNil())
		Expect(s.elems).To(BeEmpty())
	})

	Describe("add", func() {
		It("should append elems to schema", func() {
			s.add(eTable)
			Expect(len(s.elems)).To(Equal(1))
			Expect(s.elems[0]).To(Equal(eTable))

			s.add(eTrigger)
			Expect(len(s.elems)).To(Equal(2))
			Expect(s.elems[0]).To(Equal(eTable))
			Expect(s.elems[1]).To(Equal(eTrigger))
		})
	})
	Describe("addTable", func() {
		It("should append table to schema", func() {
			s.addTable(testName, testSchema)
			Expect(len(s.elems)).To(Equal(1))
			Expect(s.elems[0]).To(Equal(eTable))
		})
	})
	Describe("addTrigger", func() {
		It("should append trigger to schema", func() {
			s.addTrigger(testName, testSchema)
			Expect(len(s.elems)).To(Equal(1))
			Expect(s.elems[0]).To(Equal(eTrigger))
		})
	})
	Describe("define", func() {
		It("should append all elements to schema", func() {
			s.define()
			Expect(len(s.elems)).To(Equal(4))
			Expect(s.elems[0]).To(Equal(schemaElem{
				name:     "meta",
				elemType: table,
				schema:   metaCreate,
			}))
			Expect(s.elems[1]).To(Equal(schemaElem{
				name:     "images",
				elemType: table,
				schema:   imagesCreate,
			}))
			Expect(s.elems[2]).To(Equal(schemaElem{
				name:     "imagesFilter",
				elemType: trigger,
				schema:   imagesFilterCreate,
			}))
			Expect(s.elems[3]).To(Equal(schemaElem{
				name:     "imagesRevision",
				elemType: trigger,
				schema:   imagesRevisionCreate,
			}))
		})
	})
	Describe("with database", func() {
		var (
			dbConnection  *sql.DB
			dbTransaction *sql.Tx
			lock          = new(sync.Mutex)
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
			It("should create all schema elements in database", func() {
				s.define()

				createFunc := func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := s.create(dbTransaction)
					Expect(err).NotTo(HaveOccurred())
				}

				log, logerr := testutil.WithStderrMocked(createFunc)
				Expect(logerr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "New elem created.")).To(Equal(4))

				// Called for the second time should not have any effect.
				log, logerr = testutil.WithStderrMocked(createFunc)
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
			It("should return error if creation of schema elem fails", func() {
				s.addTable("meta", metaCreate)
				s.addTable(invalidString, invalidString)
				s.addTable("images", imagesCreate)

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := s.create(dbTransaction)
					Expect(err).To(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "New elem created.")).To(Equal(1)) // meta creation is OK.
				Expect(strings.Count(log, "Failed to create an elem.")).To(Equal(1))
			})
		})
		Describe("initialize", func() {
			It("should return error if meta cannot be initialized", func() {
				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := s.initialize(dbTransaction)
					Expect(err).To(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "Failed to initialize Revision.")).To(Equal(1))
			})
			It("should initialize meta revision with value 0", func() {
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					s.define()
					err := s.create(dbTransaction)
					Expect(err).NotTo(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())

				Expect(getRevision(dbTransaction)).To(Equal(-1))

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := s.initialize(dbTransaction)
					Expect(err).NotTo(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())

				Expect(getRevision(dbTransaction)).To(Equal(0))
			})
			It("should not affect meta revision if it has already a value assigned", func() {
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					s.define()
					err := s.create(dbTransaction)
					Expect(err).NotTo(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())

				Expect(getRevision(dbTransaction)).To(Equal(-1))

				testRev := 123456
				err := setRevision(dbTransaction, testRev)
				Expect(err).NotTo(HaveOccurred())
				Expect(getRevision(dbTransaction)).To(Equal(testRev))

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := s.initialize(dbTransaction)
					Expect(err).NotTo(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())

				Expect(getRevision(dbTransaction)).To(Equal(testRev))
			})
		})
	})
})
