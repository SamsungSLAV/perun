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

package watcher

import (
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/SamsungSLAV/perun"

	"github.com/SamsungSLAV/perun/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("configuration", func() {
	Describe("Period.UnmarshalJSON", func() {
		It("should unmarshal period successfully", func() {
			bytes := []byte(`"300ms"`)
			var p Period
			err := p.UnmarshalJSON(bytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(p).To(Equal(Period(time.Millisecond * 300)))
		})
		It("should fail if string is not quoted", func() {
			bytes := []byte(`300ms`)
			var p Period
			err := p.UnmarshalJSON(bytes)
			Expect(err).To(Equal(strconv.ErrSyntax))
			Expect(p).To(Equal(Period(0)))
		})
		It("should fail if string is not a proper time.Duration string", func() {
			bytes := []byte(`"not valid"`)
			var p Period
			err := p.UnmarshalJSON(bytes)
			Expect(err).To(HaveOccurred())
			Expect(p).To(Equal(Period(0)))
		})
	})
	Describe("loadConfiguration", func() {
		var tempFile *os.File
		const validConfiguration = `{"profiles":[` +
			`{"server": "abc", "type": "prerelease", "profile": "common-3.0", "period": "65s"},` +
			`{"server": "def", "type": "snapshot", "profile": "unified", "period": "2m"}` +
			`], "tasksqueue": 3000, "workers": 50}`
		const invalidConfiguration = `{"prof: 50}`
		saveConfiguration := func(content string) {
			_, err := tempFile.Write([]byte(content))
			Expect(err).NotTo(HaveOccurred())
			err = tempFile.Close()
			Expect(err).NotTo(HaveOccurred())
		}
		BeforeEach(func() {
			var err error
			tempFile, err = ioutil.TempFile("", "loadConfiguration_tests_tempFile")
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			os.Remove(tempFile.Name())
		})

		It("should load configuration properly", func() {
			saveConfiguration(validConfiguration)
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				conf, err := loadConfiguration(tempFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(conf).NotTo(BeNil())
				Expect(len(conf.Profiles)).To(Equal(2))
				Expect(conf.Profiles[0]).To(Equal(ProfileConfiguration{
					Server:  "abc",
					Type:    perun.PRERELEASE,
					Profile: "common-3.0",
					Period:  Period(time.Second * 65),
				}))
				Expect(conf.Profiles[1]).To(Equal(ProfileConfiguration{
					Server:  "def",
					Type:    perun.SNAPSHOT,
					Profile: "unified",
					Period:  Period(time.Minute * 2),
				}))
				Expect(conf.TasksQueue).To(Equal(3000))
				Expect(conf.Workers).To(Equal(50))
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Watcher's configuration file loaded."))
		})
		It("should fail to load not exiting file", func() {
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				conf, err := loadConfiguration("/such/file/should/not/exist")
				Expect(err).To(HaveOccurred())
				Expect(conf).To(BeNil())
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Error loading watcher's configuration file."))
		})
		It("should fail to load invalid file", func() {
			saveConfiguration(invalidConfiguration)
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				conf, err := loadConfiguration(tempFile.Name())
				Expect(err).To(HaveOccurred())
				Expect(conf).To(BeNil())
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Error parsing watcher's configuration file."))
		})
	})
	Describe("closeFile", func() {
		It("should fail to close invalid file", func() {
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				closeFile("testPath", nil)
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Error closing watcher's configuration file."))
		})
	})
})
