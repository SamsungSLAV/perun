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
	"context"
	"database/sql"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/perun/testutil"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// Initialize the sqlite3 driver for database/sql package.
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("Instance", func() {
	const (
		testRevision   = 136897
		testURL        = "testURL"
		testServer     = "testServer"
		testImageType  = perun.SNAPSHOT
		testProfile    = "testProfile"
		testSnapshot   = "testSnapshot"
		testPrerelease = "testPrerelease"
		testRepository = "testRepository"
		testImageName  = "testImageName"
		testFileName   = "testFileName"
		testLength     = 67
	)
	var (
		lock         = new(sync.Mutex)
		dbfile       *os.File
		testError    = errors.New("testError")
		testModified = time.Now().UTC()
		ifi          = perun.ImageFileInfo{
			URL: testURL,
			Image: perun.Image{
				Server:     testServer,
				ImageType:  testImageType,
				Profile:    testProfile,
				Snapshot:   testSnapshot,
				Prerelease: testPrerelease,
				Repository: testRepository,
				ImageName:  testImageName,
				FileName:   testFileName,
			},
			Info: perun.FileInfo{
				Length:   testLength,
				Modified: testModified,
			},
		}
	)
	verifyRecord := func(tx *sql.Tx, key string, expected ImageInfo, expectedRevision int) {
		var r ImageInfo
		var rev int
		var modified int64
		err := tx.QueryRow(`SELECT * FROM images WHERE url = $1`, key).
			Scan(&r.URL, &r.Image.Server, &r.Image.ImageType, &r.Image.Profile,
				&r.Image.Snapshot, &r.Image.Prerelease, &r.Image.Repository,
				&r.Image.ImageName, &r.Image.FileName, &r.Info.Length, &modified,
				&rev)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		r.Info.Modified = time.Unix(modified, 0).UTC()
		expected.Info.Modified = expected.Info.Modified.Truncate(time.Second)
		ExpectWithOffset(1, r).To(Equal(expected))
		ExpectWithOffset(1, rev).To(Equal(expectedRevision))
	}

	BeforeEach(func() {
		lock.Lock()
		defer lock.Unlock()

		var err error
		dbfile, err = ioutil.TempFile("", "perunTestDatabase")
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		lock.Lock()
		defer lock.Unlock()

		if dbfile != nil {
			_ = dbfile.Close()
			_ = os.Remove(dbfile.Name())
		}
	})

	Describe("NewDB", func() {
		It("should create new initialized instance structure and define schema", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				db, err := NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(db).NotTo(BeNil())
				inst := db.(*instance)
				Expect(inst).NotTo(BeNil())
				Expect(inst.path).To(Equal(dbfile.Name()))
				Expect(inst.connection).To(BeNil())
				Expect(len(inst.schema.elems)).To(Equal(4))
				Expect(inst.mutex).NotTo(BeNil())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "New database connection created.")).To(Equal(1))
		})
	})
	Describe("runInTransaction", func() {
		It("should fail if database is not properly initialized", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				db, err := NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				trigger := 0

				err = db.(*instance).runInTransaction(func(*sql.Tx) error {
					trigger++
					return nil
				})
				Expect(err).To(Equal(ErrDatabaseConnectionNotInitialized))
				Expect(trigger).To(Equal(0))
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Connection to database not established.")).To(Equal(1))
		})
		It("should fail if transaction cannot be open", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				db, err := NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
				err = db.(*instance).connection.Close()
				Expect(err).NotTo(HaveOccurred())
				trigger := 0

				err = db.(*instance).runInTransaction(func(*sql.Tx) error {
					trigger++
					return nil
				})
				Expect(err).To(HaveOccurred())
				Expect(trigger).To(Equal(0))
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Cannot open new transaction.")).To(Equal(1))
		})
		It("should run operation in transaction and commit changes to database", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())

				_, err = db.(*instance).connection.Exec(metaCreate)
				Expect(err).NotTo(HaveOccurred())
				_, err = db.(*instance).connection.Exec(metaRevisionInit)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(0))
					terr := setRevision(tx, testRevision)
					Expect(terr).NotTo(HaveOccurred())
					rev = getRevision(tx)
					Expect(rev).To(Equal(testRevision))
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(testRevision))
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())

			lock.Lock()
			defer lock.Unlock()
			err := db.(*instance).connection.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		It("should run operation in transaction but rollback after error", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())

				_, err = db.(*instance).connection.Exec(metaCreate)
				Expect(err).NotTo(HaveOccurred())
				_, err = db.(*instance).connection.Exec(metaRevisionInit)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(0))
					terr := setRevision(tx, testRevision)
					Expect(terr).NotTo(HaveOccurred())
					rev = getRevision(tx)
					Expect(rev).To(Equal(testRevision))
					return testError
				})
				Expect(err).To(Equal(testError))

				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(0))
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())

			lock.Lock()
			defer lock.Unlock()
			err := db.(*instance).connection.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		It("should log error if transaction cannot be commited", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					_ = tx.Rollback()
					return nil
				})
				Expect(err).To(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Cannot commit the transaction.")).To(Equal(1))

			lock.Lock()
			defer lock.Unlock()
			err := db.(*instance).connection.Close()
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("prepare", func() {
		It("should prepare schema and initilialize meta data", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.(*instance).runInTransaction(db.(*instance).prepare)
				Expect(err).NotTo(HaveOccurred())

				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(0))
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "New elem created.")).To(Equal(4))

			lock.Lock()
			defer lock.Unlock()
			err := db.(*instance).connection.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		It("should log error if schema cannot be applied", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				db.(*instance).schema.addTable("invalid", "Completely invalid SQL querry.")
				err := db.(*instance).runInTransaction(db.(*instance).prepare)
				Expect(err).To(HaveOccurred())

				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(-1))
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "New elem created.")).To(Equal(4))

			lock.Lock()
			defer lock.Unlock()
			err := db.(*instance).connection.Close()
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("UpdateImage", func() {
		It("should insert image into database", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
				err = db.(*instance).runInTransaction(db.(*instance).prepare)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.UpdateImage(&ifi)
				Expect(err).NotTo(HaveOccurred())
				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(1))
					verifyRecord(tx, testURL, ImageInfo(ifi), 1)
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())

			lock.Lock()
			defer lock.Unlock()
			err := db.(*instance).connection.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		It("verify that changes are permanently sored in database", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()

				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
				err = db.(*instance).runInTransaction(db.(*instance).prepare)
				Expect(err).NotTo(HaveOccurred())
				err = db.UpdateImage(&ifi)
				Expect(err).NotTo(HaveOccurred())
				err = db.(*instance).connection.Close()
				Expect(err).NotTo(HaveOccurred())

				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(1))
					verifyRecord(tx, testURL, ImageInfo(ifi), 1)
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
				err = db.(*instance).connection.Close()
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
		})
	})
	Describe("Close", func() {
		It("should ignore nil connection", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.Close()
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Database connection closed.")).To(Equal(1))
		})
		It("should close connection", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.Close()
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Database connection closed.")).To(Equal(1))
		})
		It("should log error if database connection closing fails", func() {
			const mockDriverName = "mockDriver"
			ctrl := gomock.NewController(GinkgoT())
			driver := NewMockDriver(ctrl)
			conn := NewMockConn(ctrl)
			ctx := context.Background()
			defer ctrl.Finish()

			sql.Register(mockDriverName, driver)

			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				db.(*instance).connection, err = sql.Open(mockDriverName, "")
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			func() {
				lock.Lock()
				defer lock.Unlock()

				driver.EXPECT().Open("").Return(conn, nil)
				c, err := db.(*instance).connection.Conn(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())

				c.Close()
			}()

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				conn.EXPECT().Close().Return(testError)
				err := db.Close()
				Expect(err).To(Equal(testError))
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Problem closing database.")).To(Equal(1))
		})
	})
	Describe("Start", func() {
		It("should open valid connection to database", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := db.Start()
				Expect(err).NotTo(HaveOccurred())

				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					rev := getRevision(tx)
					Expect(rev).To(Equal(0))
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Database connection functional.")).To(Equal(1))
		})
		It("should fail to open database using nonexisting driver", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				db.(*instance).driver = "noSuchDriver"
				err := db.Start()
				Expect(err).To(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Cannot open database.")).To(Equal(1))
		})
		It("should fail to setup invalid schema", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				db.(*instance).schema.addTable("testTable", "SELECT DROP INSERT DELETE")
				err := db.Start()
				Expect(err).To(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Database schema cannot be set.")).To(Equal(1))
		})
	})
	Describe("GetRevision", func() {
		It("should get proper revision", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				err = db.Start()
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				rev, err := db.GetRevision()
				Expect(err).NotTo(HaveOccurred())
				Expect(rev).To(Equal(0))

				err = db.(*instance).runInTransaction(func(tx *sql.Tx) error {
					return setRevision(tx, testRevision)
				})
				Expect(err).NotTo(HaveOccurred())

				rev, err = db.GetRevision()
				Expect(err).NotTo(HaveOccurred())
				Expect(rev).To(Equal(testRevision))
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
		It("should return error if revision cannot be got", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				rev, err := db.GetRevision()
				Expect(err).To(HaveOccurred())
				Expect(rev).To(Equal(-1))
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
	})
	Describe("GetImages", func() {
		It("should get proper records", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				err = db.Start()
				Expect(err).NotTo(HaveOccurred())

				ins := ifi
				ins.URL = "record_1"
				ins.Image.FileName = "AHE"
				err = db.UpdateImage(&ins)
				Expect(err).NotTo(HaveOccurred())

				ins = ifi
				ins.URL = "record_2"
				ins.Image.FileName = "AGC"
				err = db.UpdateImage(&ins)
				Expect(err).NotTo(HaveOccurred())

				ins = ifi
				ins.URL = "record_3"
				ins.Image.FileName = "ABC"
				err = db.UpdateImage(&ins)
				Expect(err).NotTo(HaveOccurred())

				ins = ifi
				ins.URL = "record_4"
				ins.Image.FileName = "BCD"
				err = db.UpdateImage(&ins)
				Expect(err).NotTo(HaveOccurred())

				ins = ifi
				ins.URL = "record_5"
				ins.Image.FileName = "ADE"
				err = db.UpdateImage(&ins)
				Expect(err).NotTo(HaveOccurred())

				ins = ifi
				ins.URL = "record_6"
				ins.Image.FileName = "AEF"
				err = db.UpdateImage(&ins)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				filter := perun.Image{
					Server:     "*",
					ImageType:  "*",
					Profile:    "*",
					Snapshot:   "*",
					Prerelease: "*",
					Repository: "*",
					ImageName:  "*",
					FileName:   "A?[C-E]",
				}
				images, rev, err := db.GetImages(&filter, 2)
				Expect(err).NotTo(HaveOccurred())
				Expect(rev).To(Equal(6))
				Expect(len(images)).To(Equal(2))
				urls := []string{images[0].URL, images[1].URL}
				Expect(urls).To(ContainElement("record_3"))
				Expect(urls).To(ContainElement("record_5"))
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
		It("should return error if revision cannot be got", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				images, rev, err := db.GetImages(nil, 0)
				Expect(err).To(Equal(ErrNilPointer))
				Expect(rev).To(Equal(-1))
				Expect(images).To(BeEmpty())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Nil filter passed to newImageFilter function.")).To(Equal(1))
		})
		It("should return error if revision cannot be got", func() {
			var db DB
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				db, err = NewDB(dbfile.Name())
				Expect(err).NotTo(HaveOccurred())
				dataSource := "file:" + dbfile.Name()
				db.(*instance).connection, err = sql.Open(defaultSQLDriver, dataSource)
				Expect(err).NotTo(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())

			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				filter := perun.Image{}
				images, rev, err := db.GetImages(&filter, 0)
				Expect(err).To(HaveOccurred())
				Expect(rev).To(Equal(-1))
				Expect(images).To(BeEmpty())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Failed to select images.")).To(Equal(1))
		})
	})
})
