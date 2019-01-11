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
	"errors"
	"fmt"
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

var _ = Describe("ImageFilter", func() {
	const (
		testServer        = "testServer"
		testImageName     = "testImageName"
		testLength        = 67
		testRevision      = 1342
		testURL           = "testURL"
		testGenericFilter = "*"
	)
	var (
		dbConnection     *sql.DB
		dbTransaction    *sql.Tx
		lock             = new(sync.Mutex)
		testError        = errors.New("testError")
		testValues       = []string{"AAA", "AAB", "ABC", "BCD"}
		genericTestValue = "ZZZ"
		testPatterns     = []string{"*", "A*", "?A*", "A?[AC]", "A?[A-C]", "*[B-Z]"}
		match            = [][]bool{
			// AAA
			{true, true, true, true, true, false},
			// AAB
			{true, true, true, false, true, true},
			// ABC
			{true, true, false, true, true, true},
			// BCD
			{true, false, false, false, false, true},
		}
		matchesForPattern = []int{4, 3, 2, 2, 3, 3}
		nV                = len(testValues)
		nP                = len(testPatterns)
		testModified      = time.Now().Round(time.Second).UTC()
		testFilter        = perun.Image{
			Server:     testGenericFilter,
			ImageType:  testGenericFilter,
			Profile:    testGenericFilter,
			Snapshot:   testGenericFilter,
			Prerelease: testGenericFilter,
			Repository: testGenericFilter,
			ImageName:  testGenericFilter,
			FileName:   testGenericFilter,
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
	setupSchema := func() {
		s := &schema{}
		s.define()
		err := s.create(dbTransaction)
		Expect(err).NotTo(HaveOccurred())
		err = s.initialize(dbTransaction)
		Expect(err).NotTo(HaveOccurred())
	}
	verifyResult := func(result, expected []perun.ImageFileInfo) {
		ExpectWithOffset(1, len(result)).To(Equal(len(expected)))
		for i, r := range result {
			ExpectWithOffset(1, r).To(Equal(expected[i]), "for element with index %d", i)
		}
	}
	Describe("newImageFilter", func() {
		It("should fail to create new structure if given filter is nil", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				req, err := newImageFilter(nil, testRevision)
				Expect(err).To(Equal(ErrNilPointer))
				Expect(req).To(BeNil())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Nil filter passed to newImageFilter function.")).
				To(Equal(1))
		})
		It("should create and initialize new structure", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				pi := &perun.Image{ImageName: testImageName}

				req, err := newImageFilter(pi, testRevision)
				Expect(err).NotTo(HaveOccurred())
				Expect(req).NotTo(BeNil())
				Expect(req.image).To(Equal(pi))
				Expect(req.since).To(Equal(testRevision))
				Expect(req.result).To(BeNil())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
	})
	Describe("get", func() {
		It("should fail to get images, when ImageFilter is invalid", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				req := &ImageFilter{}

				err := req.get(dbTransaction)
				Expect(err).To(Equal(ErrNilPointer))
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "ImageFilter not properly initialized.")).To(Equal(1))
		})
		It("should fail to get images, when schema is not created", func() {
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				pi := &perun.Image{ImageName: testImageName}
				req, err := newImageFilter(pi, testRevision)
				Expect(err).NotTo(HaveOccurred())

				err = req.get(dbTransaction)
				Expect(err).To(HaveOccurred())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Failed to select images.")).To(Equal(1))
		})
		It("should fail to get images, when scanAll fails", func() {
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				setupSchema()
			})
			Expect(logerr).NotTo(HaveOccurred())
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				pi := &perun.Image{ImageName: testImageName}
				req, err := newImageFilter(pi, testRevision)
				Expect(err).NotTo(HaveOccurred())
				req.scanAll = func(*sql.Rows, scanOneFunc) ([]perun.ImageFileInfo, error) {
					return nil, testError
				}

				err = req.get(dbTransaction)
				Expect(err).To(Equal(testError))
				Expect(req.result).To(BeNil())
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
		It("should get all images returned by scanAll function", func() {
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				setupSchema()
			})
			Expect(logerr).NotTo(HaveOccurred())
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				pi := &perun.Image{ImageName: testImageName}
				req, err := newImageFilter(pi, testRevision)
				Expect(err).NotTo(HaveOccurred())
				testCollection := []perun.ImageFileInfo{
					{Info: perun.FileInfo{Length: testLength}},
					{Image: perun.Image{Server: testServer}},
				}
				req.scanAll = func(*sql.Rows, scanOneFunc) ([]perun.ImageFileInfo, error) {
					return testCollection, nil
				}

				err = req.get(dbTransaction)
				Expect(err).NotTo(HaveOccurred())
				verifyResult(req.result, testCollection)
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
		T.DescribeTable("should return proper subset of collection filtered by different globs",
			func(iField int) {
				var collection []perun.ImageFileInfo
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					setupSchema()

					for v := 0; v < nV; v++ {
						ifi := perun.ImageFileInfo{
							URL: fmt.Sprintf("%s_%d", testURL, v),
							Image: perun.Image{
								Server:     genericTestValue,
								ImageType:  genericTestValue,
								Profile:    genericTestValue,
								Snapshot:   genericTestValue,
								Prerelease: genericTestValue,
								Repository: genericTestValue,
								ImageName:  genericTestValue,
								FileName:   genericTestValue,
							},
							Info: perun.FileInfo{
								Length:   testLength,
								Modified: testModified,
							},
						}
						switch iField {
						case 0:
							ifi.Image.Server = testValues[v]
						case 1:
							ifi.Image.ImageType = testValues[v]
						case 2:
							ifi.Image.Profile = testValues[v]
						case 3:
							ifi.Image.Snapshot = testValues[v]
						case 4:
							ifi.Image.Prerelease = testValues[v]
						case 5:
							ifi.Image.Repository = testValues[v]
						case 6:
							ifi.Image.ImageName = testValues[v]
						case 7:
							ifi.Image.FileName = testValues[v]
						}
						collection = append(collection, ifi)
						ii := ImageInfo(ifi)
						ii.insert(dbTransaction)
					}
				})
				Expect(logerr).NotTo(HaveOccurred())

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					for p := 0; p < nP; p++ {
						pi := testFilter
						switch iField {
						case 0:
							pi.Server = testPatterns[p]
						case 1:
							pi.ImageType = testPatterns[p]
						case 2:
							pi.Profile = testPatterns[p]
						case 3:
							pi.Snapshot = testPatterns[p]
						case 4:
							pi.Prerelease = testPatterns[p]
						case 5:
							pi.Repository = testPatterns[p]
						case 6:
							pi.ImageName = testPatterns[p]
						case 7:
							pi.FileName = testPatterns[p]
						}
						req, err := newImageFilter(&pi, 0)
						Expect(err).NotTo(HaveOccurred())

						err = req.get(dbTransaction)
						Expect(err).NotTo(HaveOccurred())

						Expect(len(req.result)).To(Equal(matchesForPattern[p]))
						for i := 0; i < nV; i++ {
							if match[i][p] {
								Expect(req.result).To(ContainElement(collection[i]))
							}
						}
					}
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			},
			T.Entry("Server", 0),
			T.Entry("ImageType", 1),
			T.Entry("Profile", 2),
			T.Entry("Snapshot", 3),
			T.Entry("Prerelease", 4),
			T.Entry("Repository", 5),
			T.Entry("ImageName", 6),
			T.Entry("FileName", 7),
		)
		It("should return only record with revision newer than requested", func() {
			var collection []perun.ImageFileInfo
			_, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				setupSchema()
				for v := 0; v < nV; v++ {
					ifi := perun.ImageFileInfo{
						URL: fmt.Sprintf("%s_%d", testURL, v),
						Image: perun.Image{
							Server:     testValues[v],
							ImageType:  testValues[v],
							Profile:    testValues[v],
							Snapshot:   testValues[v],
							Prerelease: testValues[v],
							Repository: testValues[v],
							ImageName:  testValues[v],
							FileName:   testValues[v],
						},
						Info: perun.FileInfo{
							Length:   testLength,
							Modified: testModified,
						},
					}
					collection = append(collection, ifi)
					ii := ImageInfo(ifi)
					ii.insert(dbTransaction)
				}
			})
			Expect(logerr).NotTo(HaveOccurred())
			log, logerr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				pi := testFilter

				for rev := 0; rev < nV+1; rev++ {
					req, err := newImageFilter(&pi, rev)
					Expect(err).NotTo(HaveOccurred())

					err = req.get(dbTransaction)
					Expect(err).NotTo(HaveOccurred())

					Expect(len(req.result)).To(Equal(nV - rev))
					for i := rev; i < nV; i++ {
						Expect(req.result).To(ContainElement(collection[i]))
					}
				}
			})
			Expect(logerr).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
	})
	Describe("scan functions", func() {
		ifi := perun.ImageFileInfo{
			URL: testURL,
			Image: perun.Image{
				Server:     genericTestValue,
				ImageType:  genericTestValue,
				Profile:    genericTestValue,
				Snapshot:   genericTestValue,
				Prerelease: genericTestValue,
				Repository: genericTestValue,
				ImageName:  genericTestValue,
				FileName:   genericTestValue,
			},
			Info: perun.FileInfo{
				Length:   testLength,
				Modified: testModified,
			},
		}
		Describe("scanOne", func() {
			It("should scan proper record", func() {
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					setupSchema()
					ii := ImageInfo(ifi)
					ii.insert(dbTransaction)
				})
				Expect(logerr).NotTo(HaveOccurred())
				rows, err := dbTransaction.Query(imageSelect, 0,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter)
				Expect(err).NotTo(HaveOccurred())
				rows.Next()
				defer rows.Close()

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					res, err := scanOne(rows)
					Expect(err).NotTo(HaveOccurred())
					Expect(*res).To(Equal(ifi))
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
			It("should log and return error if record cannnot be scanned", func() {
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					setupSchema()
				})
				Expect(logerr).NotTo(HaveOccurred())
				rows, err := dbTransaction.Query(imageSelect, 0,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter)
				Expect(err).NotTo(HaveOccurred())
				rows.Next()
				rows.Close()

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					res, err := scanOne(rows)
					Expect(err).To(HaveOccurred())
					Expect(res).To(BeNil())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "Failed to scan row while selecting images.")).
					To(Equal(1))
			})
		})
		Describe("scanAll", func() {
			It("should scan proper record", func() {
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					setupSchema()
					ii := ImageInfo(ifi)
					ii.insert(dbTransaction)
				})
				Expect(logerr).NotTo(HaveOccurred())
				rows, err := dbTransaction.Query(imageSelect, 0,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter)
				Expect(err).NotTo(HaveOccurred())
				defer rows.Close()

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					res, err := scanAll(rows, scanOne)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(res)).To(Equal(1))
					Expect(res[0]).To(Equal(ifi))
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
			It("should return error if row cannnot be scanned", func() {
				mockScan := func(*sql.Rows) (*perun.ImageFileInfo, error) {
					return nil, testError
				}
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					setupSchema()
					ii := ImageInfo(ifi)
					ii.insert(dbTransaction)
				})
				Expect(logerr).NotTo(HaveOccurred())
				rows, err := dbTransaction.Query(imageSelect, 0,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter)
				Expect(err).NotTo(HaveOccurred())
				defer rows.Close()

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					res, err := scanAll(rows, mockScan)
					Expect(err).To(Equal(testError))
					Expect(res).To(BeNil())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
			It("should log and return error if there is a row error", func() {
				_, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					setupSchema()
				})
				Expect(logerr).NotTo(HaveOccurred())
				rows, err := dbTransaction.Query(imageSelect, 0,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter, testGenericFilter,
					testGenericFilter, testGenericFilter)
				Expect(err).NotTo(HaveOccurred())
				dbTransaction.Exec("DROP TABLE images;")
				defer rows.Close()

				log, logerr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					res, err := scanAll(rows, scanOne)
					Expect(err).To(HaveOccurred())
					Expect(res).To(BeNil())
				})
				Expect(logerr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "Failed to iterate rows while selecting images.")).
					To(Equal(1))
			})
		})
	})
})
