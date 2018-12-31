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
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/perun/testutil"
	. "github.com/onsi/ginkgo"
	T "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	// Initialize the sqlite3 driver for database/sql package.
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("ImageInfo", func() {
	const (
		testURL             = "testURL"
		testURL2            = "testURL2"
		testServer          = "testServer"
		testImageType       = perun.SNAPSHOT
		testProfile         = "testProfile"
		testSnapshot        = "testSnapshot"
		testPrerelease      = "testPrerelease"
		testRepository      = "testRepository"
		testImageName       = "testImageName"
		testFileName        = "testFileName"
		testLength          = 67
		testLength2         = 672
		testInitialRevision = 90210
	)
	var (
		s             *schema
		dbConnection  *sql.DB
		dbTransaction *sql.Tx
		lock          = new(sync.Mutex)
		testModified  = time.Now().UTC()
		ifi           = perun.ImageFileInfo{
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
	Describe("insert", func() {
		It("should fail to insert image info into database, when schema is not created", func() {
			ii := ImageInfo(ifi)
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				err := ii.insert(dbTransaction)
				Expect(err).To(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Failed to insert image.")).To(Equal(1))
		})

		Describe("withSchema", func() {
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
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					s = &schema{}
					Expect(s).NotTo(BeNil())
					Expect(s.elems).To(BeEmpty())

					s.define()
					err := s.create(dbTransaction)
					Expect(err).NotTo(HaveOccurred())

					err = setRevision(dbTransaction, testInitialRevision)
					Expect(err).NotTo(HaveOccurred())
				})
				Expect(logerr).NotTo(HaveOccurred())
			})

			It("should insert record and update revision", func() {
				ii := ImageInfo(ifi)
				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					err := ii.insert(dbTransaction)
					Expect(err).NotTo(HaveOccurred())

					Expect(getRevision(dbTransaction)).To(Equal(testInitialRevision + 1))
					verifyRecord(dbTransaction, ii.URL, ii, testInitialRevision+1)
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})

			prime := ImageInfo(ifi)
			different := ImageInfo(ifi)
			different.URL = testURL2
			changed := ImageInfo(ifi)
			changed.Info.Length = testLength2
			newer := ImageInfo(ifi)
			newer.Info.Length = testLength2
			newer.Info.Modified = newer.Info.Modified.Add(time.Minute)
			older := ImageInfo(ifi)
			older.Info.Length = testLength2
			older.Info.Modified = older.Info.Modified.Add(-time.Minute)
			T.DescribeTable("should properly handle following inserts",
				func(elem, expected ImageInfo, expectedRevision int) {
					ii := ImageInfo(ifi)
					log, logerr := testutil.WithStderrMocked(func() {
						defer GinkgoRecover()
						lock.Lock()
						defer lock.Unlock()

						// Insert 1st record.
						err := ii.insert(dbTransaction)
						Expect(err).NotTo(HaveOccurred())
						Expect(getRevision(dbTransaction)).To(Equal(testInitialRevision + 1))
						verifyRecord(dbTransaction, ii.URL, ii, testInitialRevision+1)

						// Insert 2nd record.
						err = elem.insert(dbTransaction)
						Expect(err).NotTo(HaveOccurred())
						Expect(getRevision(dbTransaction)).To(Equal(expectedRevision))
						verifyRecord(dbTransaction, elem.URL, expected, expectedRevision)
					})
					Expect(logerr).NotTo(HaveOccurred())
					Expect(log).To(BeEmpty())
				},
				T.Entry("insert another record", different, different, testInitialRevision+2),
				T.Entry("ignore changed record", changed, prime, testInitialRevision+1),
				T.Entry("insert newer record", newer, newer, testInitialRevision+2),
				T.Entry("ignore older record", older, prime, testInitialRevision+1),
			)
		})
	})
})
