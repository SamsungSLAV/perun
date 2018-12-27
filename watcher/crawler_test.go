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
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/SamsungSLAV/perun/testutil"
	"github.com/SamsungSLAV/slav/logger"
	. "github.com/onsi/ginkgo"
	T "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"golang.org/x/net/html"
)

const (
	testURL     = "testURL"
	testPort    = 7380
	thisPackage = "github.com/SamsungSLAV/perun"
	testFileDir = "watcher/testfiles"
	invalidURL  = "protocol://not.existing.address!@#$%^^&"
)

var (
	testFilePath = testutil.GetGOPATH() + "/src/" + thisPackage + "/" + testFileDir
	lock         = new(sync.Mutex)
	options      = HTTPOptions{ResponseTimeout: time.Second * 123}
)

type dummyIoCloser struct{}

func (*dummyIoCloser) Close() error {
	return errors.New("dummyIoCloserError")
}

var _ = Describe("crawler", func() {
	BeforeEach(func() {
		logger.SetThreshold(logger.DebugLevel)
	})
	Describe("newCrawler", func() {
		It("should create new initialized crawler structure", func() {
			c := newCrawler(testURL, options)
			Expect(c).NotTo(BeNil())
			Expect(c.url).To(Equal(testURL))
			Expect(c.ignoredPrefixes).To(Equal(defaultIgnoredPrefixes))
			Expect(c.httpOptions).To(Equal(options))
		})
	})
	Describe("with crawler", func() {
		var c *crawler
		BeforeEach(func() {
			lock.Lock()
			defer lock.Unlock()
			c = newCrawler(testURL, options)
		})
		Describe("requireSuffixes", func() {
			T.DescribeTable("should set proper suffixes",
				func(suffixes []string) {
					lock.Lock()
					defer lock.Unlock()
					ret := c.requireSuffixes(suffixes)
					Expect(ret).To(Equal(c))
					Expect(c.requiredSuffixes).To(Equal(suffixes))
				},
				T.Entry("valid", []string{"suf1", "suf2"}),
				T.Entry("empty", []string{}),
			)
		})
		Describe("createHTTPTransport", func() {
			It("should create new HTTP transport", func() {
				rt := c.createHTTPTransport()
				Expect(rt).NotTo(BeNil())
				trans := rt.(*http.Transport)
				Expect(trans).NotTo(BeNil())

				// Default tarnsport fields.
				Expect(trans.Proxy).NotTo(BeNil())
				Expect(trans.DialContext).NotTo(BeNil())
				Expect(trans.MaxIdleConns).To(Equal(100))
				Expect(trans.IdleConnTimeout).To(Equal(time.Second * 90))
				Expect(trans.TLSHandshakeTimeout).To(Equal(time.Second * 10))
				Expect(trans.ExpectContinueTimeout).To(Equal(time.Second * 1))

				// Customized fields
				Expect(trans.ResponseHeaderTimeout).To(Equal(options.ResponseTimeout))

				// Not initialized fields.
				// Disable linter check for deprecated Dial field to supress warning.
				Expect(trans.Dial).To(BeZero()) // nolint: megacheck
				Expect(trans.DialTLS).To(BeZero())
				Expect(trans.DisableCompression).To(BeZero())
				Expect(trans.DisableKeepAlives).To(BeZero())
				Expect(trans.MaxConnsPerHost).To(BeZero())
				Expect(trans.MaxIdleConnsPerHost).To(BeZero())
				Expect(trans.MaxResponseHeaderBytes).To(BeZero())
				Expect(trans.ProxyConnectHeader).To(BeZero())
				Expect(trans.TLSClientConfig).To(BeZero())
				Expect(trans.TLSNextProto).To(BeZero())
			})
		})
		Describe("getLinks", func() {
			var server *testutil.HTTPFileServer

			BeforeEach(func() {
				server = testutil.NewHTTPFileServer(testPort, testFilePath)
				server.Start()
				Eventually(server.IsStarted).Should(BeTrue())
				Consistently(server.IsFailed).Should(BeFalse())
			})
			AfterEach(func() {
				server.Stop()
				Consistently(server.IsFailed).Should(BeFalse())
			})
			T.DescribeTable("should get and parse valid html filescontaining links",
				func(filename string, expectedLinks []string) {
					lock.Lock()
					c.url = fmt.Sprintf("http://127.0.0.1:%d/%s", testPort, filename)
					lock.Unlock()
					log, err := testutil.WithStderrMocked(func() {
						defer GinkgoRecover()
						lock.Lock()
						defer lock.Unlock()
						ret := c.getLinks()
						Expect(ret).To(Equal(expectedLinks))
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(log).To(BeEmpty())
				},
				T.Entry("empty", "empty.html", []string{}),
				T.Entry("single", "single_prerelease.html", []string{
					"tizen-3.0-common_20181024.1.20181030.084905",
				}),
				T.Entry("files_list", "files_list.html", []string{
					"MD5SUMS",
					"SHA1SUMS",
					"SHA256SUMS",
					"manifest.json",
					"tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.ks",
					"tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.log",
					"tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.packages",
					"tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.tar.gz",
					"tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.xml",
				}),
				T.Entry("images_list", "images_list.html", []string{
					"common-boot-armv7l-odroidu3",
					"common-boot-armv7l-odroidxu3",
					"common-wayland-3parts-arm64",
					"common-wayland-3parts-armv7l-odroidu3",
					"common-wayland-3parts-armv7l-rpi3",
					"iot-boot-arm64-artik710",
					"iot-boot-arm64-rpi3",
					"iot-boot-armv7l-artik530",
					"iot-boot-armv7l-artik533s",
					"iot-headed-3parts-armv7l-artik530_710",
					"iot-headless-2parts-armv7l-artik530_710",
					"iot-headless-2parts-armv7l-rpi3",
					"mobile-boot-arm64-tm2",
					"mobile-boot-armv7l-tm1",
					"mobile-wayland-arm64-tm2",
					"mobile-wayland-armv7l-tm1",
					"mobile-wayland-armv7l-tm2",
					"tv-boot-armv7l-odroidu3",
					"tv-boot-armv7l-odroidxu3",
					"tv-wayland-armv7l-odroidxu3",
					"tv-wayland-armv7l-rpi3",
					"wearable-wayland-armv7l-tw2",
				}),
				T.Entry("snapshot_list", "snapshot_list.html", []string{
					"tizen-unified_20181018.1",
					"tizen-unified_20181019.1",
					"tizen-unified_20181023.1",
					"tizen-unified_20181023.2",
					"tizen-unified_20181023.3",
					"tizen-unified_20181024.1",
					"tizen-unified_20181031.1",
					"tizen-unified_20181103.1",
					"tizen-unified_20181105.1",
					"tizen-unified_20181105.2",
					"tizen-unified_20181106.1",
				}),
				T.Entry("repositories_list", "snapshot_repositories_list.html", []string{
					"emulator",
					"standard",
				}),
			)
			It("should fail to get from unexisting url", func() {
				lock.Lock()
				c.url = invalidURL
				lock.Unlock()
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()
					ret := c.getLinks()
					Expect(ret).To(BeNil())
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot GET requested url."))
			})
		})
		Describe("closeBody", func() {
			It("should log error when closing fails", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					closer := &dummyIoCloser{}
					c.closeBody(closer)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot close response body."))
			})
		})
		Describe("parsing links", func() {
			var file *os.File

			openFile := func(filename string) {
				var err error
				file, err = os.Open(testFilePath + "/" + filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(file).NotTo(BeNil())
			}
			AfterEach(func() {
				err := file.Close()
				Expect(err).NotTo(HaveOccurred())
			})
			It("parseLinks should parse valid file", func() {
				openFile("snapshot_repositories_list.html")
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.parseLinks(file)
					Expect(ret).To(Equal([]string{
						"emulator",
						"standard",
					}))
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
			It("parseATags should log on html token parsing failure", func() {
				openFile("invalid_tag.html")
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					tok := html.NewTokenizer(file)
					tok.SetMaxBuf(1)

					ret := c.parseATags(tok)
					Expect(ret).To(BeNil())
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Error during parsing HTML token."))
			})
			It("isMatching should use required Suffixes", func() {
				openFile("files_list.html")
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					c.requireSuffixes([]string{".json", ".ks", ".log"})

					ret := c.parseLinks(file)
					Expect(ret).To(Equal([]string{
						"manifest.json",
						"tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.ks",
						"tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.log",
					}))
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
		})
		Describe("getFileInfo", func() {
			var server *testutil.HTTPFileServer

			BeforeEach(func() {
				server = testutil.NewHTTPFileServer(testPort, testFilePath)
				server.Start()
				Eventually(server.IsStarted).Should(BeTrue())
				Consistently(server.IsFailed).Should(BeFalse())
			})
			AfterEach(func() {
				server.Stop()
				Consistently(server.IsFailed).Should(BeFalse())
			})
			T.DescribeTable("should get and parse valid html header",
				func(filename string) {
					lock.Lock()
					c.url = fmt.Sprintf("http://127.0.0.1:%d/%s", testPort, filename)
					lock.Unlock()
					path := testFilePath + "/" + filename

					modifiedTime := time.Now()
					err := os.Chtimes(path, modifiedTime, modifiedTime)
					Expect(err).NotTo(HaveOccurred())

					info, err := os.Stat(path)
					Expect(err).NotTo(HaveOccurred())

					log, err := testutil.WithStderrMocked(func() {
						defer GinkgoRecover()
						lock.Lock()
						defer lock.Unlock()

						ret := c.getFileInfo()
						Expect(ret).NotTo(BeNil())
						Expect(ret.Length).To(Equal(info.Size()))
						Expect(ret.Modified).To(BeTemporally("~", modifiedTime, time.Second))
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(log).To(BeEmpty())
				},
				T.Entry("empty", "empty.html"),
				T.Entry("single", "single_prerelease.html"),
				T.Entry("files_list", "files_list.html"),
				T.Entry("images_list", "images_list.html"),
				T.Entry("snapshot_list", "snapshot_list.html"),
				T.Entry("repositories_list", "snapshot_repositories_list.html"),
			)
			It("should fail to head from unexisting url", func() {
				lock.Lock()
				c.url = invalidURL
				lock.Unlock()
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.getFileInfo()
					Expect(ret).To(BeNil())
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot HEAD requested url."))
			})
		})
		Describe("parseHeader", func() {
			const (
				testLenStr = "123456789"
				testLen    = int64(123456789)
				lenKey     = "Content-Length"
				dateKey    = "Last-Modified"
				badString  = "badString"
			)
			var (
				testDate    = time.Date(2013, 12, 11, 10, 9, 8, 0, time.UTC)
				testDateStr = testDate.Format(time.RFC1123)
			)
			It("should parse header successfully", func() {
				header := http.Header{
					lenKey:  []string{testLenStr},
					dateKey: []string{testDateStr},
				}
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.parseHeader(header)
					Expect(ret).NotTo(BeNil())
					Expect(ret.Length).To(Equal(testLen))
					Expect(ret.Modified).To(BeTemporally("==", testDate))
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
			It("should fail if length key-value pair is missing", func() {
				header := http.Header{
					dateKey: []string{testDateStr},
				}
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.parseHeader(header)
					Expect(ret).To(BeNil())
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("No valid Content-Length value."))
			})
			It("should fail if length is invalid", func() {
				header := http.Header{
					lenKey:  []string{badString},
					dateKey: []string{testDateStr},
				}
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.parseHeader(header)
					Expect(ret).To(BeNil())
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Parsing Content-Length value failed."))
			})
			It("should fail if modification date key-value pair is missing", func() {
				header := http.Header{
					lenKey: []string{testLenStr},
				}
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.parseHeader(header)
					Expect(ret).To(BeNil())
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("No valid Last-Modified value."))
			})
			It("should fail if modification date is invalid", func() {
				header := http.Header{
					lenKey:  []string{testLenStr},
					dateKey: []string{badString},
				}
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.parseHeader(header)
					Expect(ret).To(BeNil())
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Parsing Last-Modified value failed."))
			})
			It("should use only 1st string in value slice ignoring rest", func() {
				header := http.Header{
					lenKey:  []string{testLenStr, badString, badString, badString},
					dateKey: []string{testDateStr, badString, badString},
				}
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					ret := c.parseHeader(header)
					Expect(ret).NotTo(BeNil())
					Expect(ret.Length).To(Equal(testLen))
					Expect(ret.Modified).To(BeTemporally("==", testDate))
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(BeEmpty())
			})
		})
	})
})
