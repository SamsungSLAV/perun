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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/perun/testutil"
	"github.com/SamsungSLAV/perun/util"
	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	T "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("worker", func() {
	const (
		testID      = 67
		server      = "download.tizen.org"
		profile     = "unified"
		snapshot    = "20181103.1"
		repository  = "standard"
		imageName   = "iot-headless-2parts-armv7l-artik530_710"
		fileName    = "tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.tar.gz"
		prerelease2 = "20181107.094441"
		profile2    = "4.0-unified"
		snapshot2   = "20181031.1"
		imageName2  = "mobile-wayland-arm64-tm2"
		fileName2   = "tizen-4.0-unified_20181031.1.20181107.094441_mobile-wayland-arm64-tm2.tar.gz"
	)

	var (
		lock             = sync.Locker(new(sync.Mutex))
		ctrl             *gomock.Controller
		consumer         *MockTaskConsumer
		db               *MockDB
		listSnapshots    *MockGauge
		listPrereleases  *MockGauge
		listRepositories *MockGauge
		listImages       *MockGauge
		listFiles        *MockGauge
		getFileInfo      *MockGauge
		workerBusy       *MockGauge
		workerIdle       *MockGauge
		queueTasks       *MockGauge
		gauges           gauges

		tasks chan Task
		grs   util.GoRoutineSync
		stop  chan interface{}

		SnapshotListSnapshots = Task{
			TaskType: ListSnapshots,
			Image: perun.Image{
				Server:    server,
				ImageType: perun.SNAPSHOT,
				Profile:   profile,
			},
		}
		SnapshotListRepositoriesTask = Task{
			TaskType: ListRepositories,
			Image: perun.Image{
				Server:    server,
				ImageType: perun.SNAPSHOT,
				Profile:   profile,
				Snapshot:  snapshot,
			},
		}
		SnapshotListImagesTask = Task{
			TaskType: ListImages,
			Image: perun.Image{
				Server:     server,
				ImageType:  perun.SNAPSHOT,
				Profile:    profile,
				Snapshot:   snapshot,
				Repository: repository,
			},
		}
		SnapshotListFilesTask = Task{
			TaskType: ListFiles,
			Image: perun.Image{
				Server:     server,
				ImageType:  perun.SNAPSHOT,
				Profile:    profile,
				Snapshot:   snapshot,
				Repository: repository,
				ImageName:  imageName,
			},
		}
		SnapshotGetFileInfoTask = Task{
			TaskType: GetFileInfo,
			Image: perun.Image{
				Server:     server,
				ImageType:  perun.SNAPSHOT,
				Profile:    profile,
				Snapshot:   snapshot,
				Repository: repository,
				ImageName:  imageName,
				FileName:   fileName,
			},
		}

		PrereleaseListSnapshots = Task{
			TaskType: ListSnapshots,
			Image: perun.Image{
				Server:    server,
				ImageType: perun.PRERELEASE,
				Profile:   profile2,
			},
		}
		PrereleaseListPrereleasesTask = Task{
			TaskType: ListPrereleases,
			Image: perun.Image{
				Server:    server,
				ImageType: perun.PRERELEASE,
				Profile:   profile2,
				Snapshot:  snapshot2,
			},
		}
		PrereleaseListRepositoriesTask = Task{
			TaskType: ListRepositories,
			Image: perun.Image{
				Server:     server,
				ImageType:  perun.PRERELEASE,
				Profile:    profile2,
				Snapshot:   snapshot2,
				Prerelease: prerelease2,
			},
		}
		PrereleaseListImagesTask = Task{
			TaskType: ListImages,
			Image: perun.Image{
				Server:     server,
				ImageType:  perun.PRERELEASE,
				Profile:    profile2,
				Snapshot:   snapshot2,
				Prerelease: prerelease2,
				Repository: repository,
			},
		}
		PrereleaseListFilesTask = Task{
			TaskType: ListFiles,
			Image: perun.Image{
				Server:     server,
				ImageType:  perun.PRERELEASE,
				Profile:    profile2,
				Snapshot:   snapshot2,
				Prerelease: prerelease2,
				Repository: repository,
				ImageName:  imageName2,
			},
		}
		PrereleaseGetFileInfoTask = Task{
			TaskType: GetFileInfo,
			Image: perun.Image{
				Server:     server,
				ImageType:  perun.PRERELEASE,
				Profile:    profile2,
				Snapshot:   snapshot2,
				Prerelease: prerelease2,
				Repository: repository,
				ImageName:  imageName2,
				FileName:   fileName2,
			},
		}
		testError error
	)

	BeforeEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl = gomock.NewController(GinkgoT())
		consumer = NewMockTaskConsumer(ctrl)
		db = NewMockDB(ctrl)
		listSnapshots = NewMockGauge(ctrl)
		listPrereleases = NewMockGauge(ctrl)
		listRepositories = NewMockGauge(ctrl)
		listImages = NewMockGauge(ctrl)
		listFiles = NewMockGauge(ctrl)
		getFileInfo = NewMockGauge(ctrl)
		workerBusy = NewMockGauge(ctrl)
		workerIdle = NewMockGauge(ctrl)
		queueTasks = NewMockGauge(ctrl)
		gauges.listSnapshots = listSnapshots
		gauges.listPrereleases = listPrereleases
		gauges.listRepositories = listRepositories
		gauges.listImages = listImages
		gauges.listFiles = listFiles
		gauges.getFileInfo = getFileInfo
		gauges.workerBusy = workerBusy
		gauges.workerIdle = workerIdle
		gauges.queueTasks = queueTasks

		tasks = make(chan Task)
		grs.Done = new(sync.WaitGroup)
		stop = make(chan interface{})
		grs.Stop = stop

		testError = errors.New("testError")
	})
	AfterEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl.Finish()
	})
	Describe("newWorker", func() {
		It("should create new, properly initilialized worker", func() {
			lock.Lock()
			defer lock.Unlock()
			w := newWorker(testID, tasks, grs, consumer, db, gauges)
			Expect(w).NotTo(BeNil())
			Expect(w.id).To(Equal(testID))
			Expect(w.tasks).To(Equal((<-chan Task)(tasks)))
			Expect(w.grs).To(Equal(grs))
			Expect(w.taskConsumer).To(Equal(consumer))
			Expect(w.db).To(Equal(db))
			Expect(w.gauges).To(Equal(gauges))
		})
	})
	Describe("buildURL", func() {
		It("should return empty URL and log error if task type is not valid", func() {
			lock.Lock()
			w := newWorker(testID, tasks, grs, consumer, db, gauges)
			lock.Unlock()

			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				url := w.buildURL(Task{
					Image: perun.Image{
						ImageType: "unknown",
					},
				})
				Expect(url).To(BeEmpty())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Invalid image type in task."))
		})
		T.DescribeTable("should build proper URL",
			func(task Task, expectedURL string) {
				lock.Lock()
				defer lock.Unlock()
				w := newWorker(testID, tasks, grs, consumer, db, gauges)

				url := w.buildURL(task)
				Expect(url).To(Equal(expectedURL))
			},
			T.Entry("snapshot list", SnapshotListSnapshots,
				`http://download.tizen.org/snapshots/tizen/unified/`),
			T.Entry("snapshot repository list", SnapshotListRepositoriesTask,
				`http://download.tizen.org/snapshots/tizen/unified/tizen-unified_20181103.1/`+
					`images/`),
			T.Entry("snapshot images list", SnapshotListImagesTask,
				`http://download.tizen.org/snapshots/tizen/unified/tizen-unified_20181103.1/`+
					`images/standard/`),
			T.Entry("snapshot files list", SnapshotListFilesTask,
				`http://download.tizen.org/snapshots/tizen/unified/tizen-unified_20181103.1/`+
					`images/standard/iot-headless-2parts-armv7l-artik530_710/`),
			T.Entry("snapshot file name", SnapshotGetFileInfoTask,
				`http://download.tizen.org/snapshots/tizen/unified/tizen-unified_20181103.1/`+
					`images/standard/iot-headless-2parts-armv7l-artik530_710/`+
					`tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.tar.gz`),
			T.Entry("prerelease snapshot list", PrereleaseListSnapshots,
				`http://download.tizen.org/prerelease/tizen/4.0-unified/`),
			T.Entry("prerelease list", PrereleaseListPrereleasesTask,
				`http://download.tizen.org/prerelease/tizen/4.0-unified/`+
					`tizen-4.0-unified_20181031.1/`),
			T.Entry("prerelease repository list", PrereleaseListRepositoriesTask,
				`http://download.tizen.org/prerelease/tizen/4.0-unified/`+
					`tizen-4.0-unified_20181031.1/`+
					`tizen-4.0-unified_20181031.1.20181107.094441/images/`),
			T.Entry("prerelease images list", PrereleaseListImagesTask,
				`http://download.tizen.org/prerelease/tizen/4.0-unified/`+
					`tizen-4.0-unified_20181031.1/`+
					`tizen-4.0-unified_20181031.1.20181107.094441/images/standard/`),
			T.Entry("prerelease files list", PrereleaseListFilesTask,
				`http://download.tizen.org/prerelease/tizen/4.0-unified/`+
					`tizen-4.0-unified_20181031.1/`+
					`tizen-4.0-unified_20181031.1.20181107.094441/images/standard/`+
					`mobile-wayland-arm64-tm2/`),
			T.Entry("prerelease file name", PrereleaseGetFileInfoTask,
				`http://download.tizen.org/prerelease/tizen/4.0-unified/`+
					`tizen-4.0-unified_20181031.1/`+
					`tizen-4.0-unified_20181031.1.20181107.094441/images/standard/`+
					`mobile-wayland-arm64-tm2/`+
					`tizen-4.0-unified_20181031.1.20181107.094441_`+
					`mobile-wayland-arm64-tm2.tar.gz`),
		)
	})
	Describe("with local server", func() {
		const (
			localServer   = "127.0.0.1:7380"
			testPort      = 7380
			thisPackage   = "github.com/SamsungSLAV/perun"
			testFileDir   = "watcher/testfiles/root"
			invalidServer = "1000.500.100.900"
		)
		var (
			testFilePath = testutil.GetGOPATH() + "/src/" + thisPackage + "/" + testFileDir
			server       *testutil.HTTPFileServer
			w            *worker
		)

		BeforeEach(func() {
			lock.Lock()
			defer lock.Unlock()
			server = testutil.NewHTTPFileServer(testPort, testFilePath)
			server.Start()
			Eventually(server.IsStarted).Should(BeTrue())
			Consistently(server.IsFailed).Should(BeFalse())
			w = newWorker(testID, tasks, grs, consumer, db, gauges)
		})
		AfterEach(func() {
			lock.Lock()
			defer lock.Unlock()
			server.Stop()
			Consistently(server.IsFailed).Should(BeFalse())
		})
		Describe("processListSnapshots", func() {
			It("should delegate sub tasks for snapshot image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := SnapshotListSnapshots
				task.Server = localServer

				call := listSnapshots.EXPECT().Inc()
				expectedSnapshots := []perun.Snapshot{
					"20181018.1",
					"20181019.1",
					"20181023.1",
					"20181023.2",
					"20181023.3",
					"20181024.1",
					"20181031.1",
					"20181103.1",
					"20181105.1",
					"20181105.2",
					"20181106.1",
				}
				for _, snapshot := range expectedSnapshots {
					newtask := SnapshotListRepositoriesTask
					newtask.Server = localServer
					newtask.Snapshot = snapshot
					call = consumer.EXPECT().Run(newtask).After(call)
				}
				listSnapshots.EXPECT().Dec().After(call)
				w.processListSnapshots(task)
			})
			It("should delegate sub tasks for prerelease image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := PrereleaseListSnapshots
				task.Server = localServer

				call := listSnapshots.EXPECT().Inc()
				expectedSnapshots := []perun.Snapshot{
					"20181011.1",
					"20181031.1",
					"20181106.1",
				}
				for _, snapshot := range expectedSnapshots {
					newtask := PrereleaseListPrereleasesTask
					newtask.Server = localServer
					newtask.Snapshot = snapshot
					call = consumer.EXPECT().Run(newtask).After(call)
				}
				listSnapshots.EXPECT().Dec().After(call)
				w.processListSnapshots(task)
			})
			It("should log error if task is invalid", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListSnapshots
					task.Server = localServer
					task.ImageType = "unknown"

					gomock.InOrder(
						listSnapshots.EXPECT().Inc(),
						listSnapshots.EXPECT().Dec(),
					)
					w.processListSnapshots(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Invalid image type in task."))
			})
			It("should log error if URL is not found", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListSnapshots
					task.Server = invalidServer

					gomock.InOrder(
						listSnapshots.EXPECT().Inc(),
						listSnapshots.EXPECT().Dec(),
					)
					w.processListSnapshots(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot GET requested url."))
			})
		})
		Describe("processListPrereleases", func() {
			It("should delegate sub tasks for prerelease image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := PrereleaseListPrereleasesTask
				task.Server = localServer

				call := listPrereleases.EXPECT().Inc()
				expectedPrereleases := []perun.Prerelease{
					"20181106.191541",
					"20181107.094441",
					"20181108.004736",
				}
				for _, prerelease := range expectedPrereleases {
					newtask := PrereleaseListRepositoriesTask
					newtask.Server = localServer
					newtask.Prerelease = prerelease
					call = consumer.EXPECT().Run(newtask).After(call)
				}
				listPrereleases.EXPECT().Dec().After(call)
				w.processListPrereleases(task)
			})
			It("should log error if task is invalid", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListPrereleasesTask
					task.Server = localServer
					task.ImageType = "unknown"

					gomock.InOrder(
						listPrereleases.EXPECT().Inc(),
						listPrereleases.EXPECT().Dec(),
					)
					w.processListPrereleases(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Invalid image type in task."))
			})
			It("should log error if URL is not found", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListPrereleasesTask
					task.Server = invalidServer

					gomock.InOrder(
						listPrereleases.EXPECT().Inc(),
						listPrereleases.EXPECT().Dec(),
					)
					w.processListPrereleases(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot GET requested url."))
			})
		})
		Describe("processListRepositories", func() {
			It("should delegate sub tasks for snapshot image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := SnapshotListRepositoriesTask
				task.Server = localServer

				call := listRepositories.EXPECT().Inc()
				expectedRepositories := []perun.Repository{
					"emulator",
					"standard",
				}
				for _, repository := range expectedRepositories {
					newtask := SnapshotListImagesTask
					newtask.Server = localServer
					newtask.Repository = repository
					call = consumer.EXPECT().Run(newtask).After(call)
				}
				listRepositories.EXPECT().Dec().After(call)
				w.processListRepositories(task)
			})
			It("should delegate sub tasks for prerelease image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := PrereleaseListRepositoriesTask
				task.Server = localServer

				call := listRepositories.EXPECT().Inc()
				expectedRepositories := []perun.Repository{
					"emulator",
					"standard",
				}
				for _, repository := range expectedRepositories {
					newtask := PrereleaseListImagesTask
					newtask.Server = localServer
					newtask.Repository = repository
					call = consumer.EXPECT().Run(newtask).After(call)
				}
				listRepositories.EXPECT().Dec().After(call)
				w.processListRepositories(task)
			})
			It("should log error if task is invalid", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListRepositoriesTask
					task.Server = localServer
					task.ImageType = "unknown"

					gomock.InOrder(
						listRepositories.EXPECT().Inc(),
						listRepositories.EXPECT().Dec(),
					)
					w.processListRepositories(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Invalid image type in task."))
			})
			It("should log error if URL is not found", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListRepositoriesTask
					task.Server = invalidServer

					gomock.InOrder(
						listRepositories.EXPECT().Inc(),
						listRepositories.EXPECT().Dec(),
					)
					w.processListRepositories(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot GET requested url."))
			})
		})
		Describe("processListImages", func() {
			It("should delegate sub tasks for snapshot image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := SnapshotListImagesTask
				task.Server = localServer

				call := listImages.EXPECT().Inc()
				expectedImages := []perun.ImageName{
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
				}
				for _, image := range expectedImages {
					newtask := SnapshotListFilesTask
					newtask.Server = localServer
					newtask.ImageName = image
					call = consumer.EXPECT().Run(newtask).After(call)
				}
				listImages.EXPECT().Dec().After(call)
				w.processListImages(task)
			})
			It("should delegate sub tasks for prerelease image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := PrereleaseListImagesTask
				task.Server = localServer

				call := listImages.EXPECT().Inc()
				expectedImages := []perun.ImageName{
					"common-boot-armv7l-artik10",
					"common-boot-armv7l-odroidu3",
					"common-boot-armv7l-odroidxu3",
					"common-wayland-3parts-arm64",
					"common-wayland-3parts-armv7l-artik",
					"common-wayland-3parts-armv7l-odroidu3",
					"common-wayland-3parts-armv7l-rpi3",
					"iot-boot-arm64-artik710",
					"iot-boot-arm64-rpi3",
					"iot-boot-armv7l-artik530",
					"iot-boot-armv7l-artik533s",
					"iot-gps_tracker-2parts-armv7l-rpi3",
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
					"tv-wayland-armv7l-odroidu3",
					"tv-wayland-armv7l-rpi3",
					"wearable-wayland-armv7l-tw1",
					"wearable-wayland-armv7l-tw2",
				}
				for _, image := range expectedImages {
					newtask := PrereleaseListFilesTask
					newtask.Server = localServer
					newtask.ImageName = image
					call = consumer.EXPECT().Run(newtask).After(call)
				}
				listImages.EXPECT().Dec().After(call)
				w.processListImages(task)
			})
			It("should log error if task is invalid", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListImagesTask
					task.Server = localServer
					task.ImageType = "unknown"

					gomock.InOrder(
						listImages.EXPECT().Inc(),
						listImages.EXPECT().Dec(),
					)
					w.processListImages(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Invalid image type in task."))
			})
			It("should log error if URL is not found", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListImagesTask
					task.Server = invalidServer

					gomock.InOrder(
						listImages.EXPECT().Inc(),
						listImages.EXPECT().Dec(),
					)
					w.processListImages(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot GET requested url."))
			})
		})
		Describe("processListFiles", func() {
			It("should delegate sub tasks for snapshot image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := SnapshotListFilesTask
				task.Server = localServer

				file := "tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.tar.gz"
				newtask := SnapshotGetFileInfoTask
				newtask.Server = localServer
				newtask.FileName = file
				gomock.InOrder(
					listFiles.EXPECT().Inc(),
					consumer.EXPECT().Run(newtask),
					listFiles.EXPECT().Dec(),
				)
				w.processListFiles(task)
			})
			It("should delegate sub tasks for prerelease image type", func() {
				lock.Lock()
				defer lock.Unlock()

				task := PrereleaseListFilesTask
				task.Server = localServer

				file := `tizen-4.0-unified_20181031.1.20181107.094441_` +
					`mobile-wayland-arm64-tm2.tar.gz`
				newtask := PrereleaseGetFileInfoTask
				newtask.Server = localServer
				newtask.FileName = file
				gomock.InOrder(
					listFiles.EXPECT().Inc(),
					consumer.EXPECT().Run(newtask),
					listFiles.EXPECT().Dec(),
				)
				w.processListFiles(task)
			})
			It("should log error if task is invalid", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListFilesTask
					task.Server = localServer
					task.ImageType = "unknown"

					gomock.InOrder(
						listFiles.EXPECT().Inc(),
						listFiles.EXPECT().Dec(),
					)
					w.processListFiles(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Invalid image type in task."))
			})
			It("should log error if URL is not found", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListFilesTask
					task.Server = invalidServer

					gomock.InOrder(
						listFiles.EXPECT().Inc(),
						listFiles.EXPECT().Dec(),
					)
					w.processListFiles(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot GET requested url."))
			})
		})
		Describe("processGetFileInfo", func() {
			It("should update db for snapshot image type", func() {
				lock.Lock()
				defer lock.Unlock()

				file := `snapshots/tizen/unified/tizen-unified_20181103.1/` +
					`images/standard/iot-headless-2parts-armv7l-artik530_710/` +
					`tizen-unified_20181103.1_iot-headless-2parts-armv7l-artik530_710.tar.gz`
				info, err := os.Stat(testFilePath + "/" + file)
				Expect(err).NotTo(HaveOccurred())

				task := SnapshotGetFileInfoTask
				task.Server = localServer

				gomock.InOrder(
					getFileInfo.EXPECT().Inc(),
					db.EXPECT().UpdateImage(gomock.Any()).Do(func(ifi *perun.ImageFileInfo) {
						Expect(ifi.URL).To(Equal("http://" + localServer + "/" + file))
						Expect(ifi.Image).To(Equal(task.Image))
						Expect(ifi.Info).NotTo(BeNil())
						Expect(ifi.Info.Length).To(Equal(info.Size()))
						Expect(ifi.Info.Modified).To(
							BeTemporally("~", info.ModTime(), time.Second))
					}),
					getFileInfo.EXPECT().Dec(),
				)
				w.processGetFileInfo(task)
			})
			It("should update db for prerelease image type", func() {
				lock.Lock()
				defer lock.Unlock()

				file := `prerelease/tizen/4.0-unified/tizen-4.0-unified_20181031.1/` +
					`tizen-4.0-unified_20181031.1.20181107.094441/images/standard/` +
					`mobile-wayland-arm64-tm2/` +
					`tizen-4.0-unified_20181031.1.20181107.094441_mobile-wayland-arm64-tm2.tar.gz`
				info, err := os.Stat(testFilePath + "/" + file)
				Expect(err).NotTo(HaveOccurred())

				task := PrereleaseGetFileInfoTask
				task.Server = localServer

				gomock.InOrder(
					getFileInfo.EXPECT().Inc(),
					db.EXPECT().UpdateImage(gomock.Any()).Do(func(ifi *perun.ImageFileInfo) {
						Expect(ifi.URL).To(Equal("http://" + localServer + "/" + file))
						Expect(ifi.Image).To(Equal(task.Image))
						Expect(ifi.Info).NotTo(BeNil())
						Expect(ifi.Info.Length).To(Equal(info.Size()))
						Expect(ifi.Info.Modified).To(
							BeTemporally("~", info.ModTime(), time.Second))
					}),
					getFileInfo.EXPECT().Dec(),
				)
				w.processGetFileInfo(task)
			})
			It("should log error if task is invalid", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseGetFileInfoTask
					task.Server = localServer
					task.ImageType = "unknown"

					gomock.InOrder(
						getFileInfo.EXPECT().Inc(),
						getFileInfo.EXPECT().Dec(),
					)
					w.processGetFileInfo(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Invalid image type in task."))
			})
			It("should log error if URL is not found", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseGetFileInfoTask
					task.Server = invalidServer

					gomock.InOrder(
						getFileInfo.EXPECT().Inc(),
						getFileInfo.EXPECT().Dec(),
					)
					w.processGetFileInfo(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Cannot HEAD requested url."))
			})
			It("should log error if database cannot be updated", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()

					lock.Lock()
					defer lock.Unlock()

					file := `prerelease/tizen/4.0-unified/tizen-4.0-unified_20181031.1/` +
						`tizen-4.0-unified_20181031.1.20181107.094441/` +
						`images/standard/mobile-wayland-arm64-tm2/` +
						`tizen-4.0-unified_20181031.1.20181107.094441_` +
						`mobile-wayland-arm64-tm2.tar.gz`
					info, err := os.Stat(testFilePath + "/" + file)
					Expect(err).NotTo(HaveOccurred())

					task := PrereleaseGetFileInfoTask
					task.Server = localServer

					gomock.InOrder(
						getFileInfo.EXPECT().Inc(),
						db.EXPECT().UpdateImage(gomock.Any()).DoAndReturn(
							func(ifi *perun.ImageFileInfo) error {
								Expect(ifi.URL).To(Equal("http://" + localServer + "/" + file))
								Expect(ifi.Image).To(Equal(task.Image))
								Expect(ifi.Info).NotTo(BeNil())
								Expect(ifi.Info.Length).To(Equal(info.Size()))
								Expect(ifi.Info.Modified).To(
									BeTemporally("~", info.ModTime(), time.Second))
								return testError
							}),
						getFileInfo.EXPECT().Dec(),
					)
					w.processGetFileInfo(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Error updating image in database."))
			})
		})
		Describe("process", func() {
			It("should call processListSnapshots", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListSnapshots
					task.Server = localServer

					call := listSnapshots.EXPECT().Inc()
					expectedSnapshots := []perun.Snapshot{
						"20181011.1",
						"20181031.1",
						"20181106.1",
					}
					for _, snapshot := range expectedSnapshots {
						newtask := PrereleaseListPrereleasesTask
						newtask.Server = localServer
						newtask.Snapshot = snapshot
						call = consumer.EXPECT().Run(newtask).After(call)
					}
					listSnapshots.EXPECT().Dec().After(call)
					w.process(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] starts processing task."))
				Expect(log).To(ContainSubstring("Worker [67] completed processing task."))
			})
			It("should call processListPrereleases", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListPrereleasesTask
					task.Server = localServer

					call := listPrereleases.EXPECT().Inc()
					expectedPrereleases := []perun.Prerelease{
						"20181106.191541",
						"20181107.094441",
						"20181108.004736",
					}
					for _, prerelease := range expectedPrereleases {
						newtask := PrereleaseListRepositoriesTask
						newtask.Server = localServer
						newtask.Prerelease = prerelease
						call = consumer.EXPECT().Run(newtask).After(call)
					}
					listPrereleases.EXPECT().Dec().After(call)
					w.process(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] starts processing task."))
				Expect(log).To(ContainSubstring("Worker [67] completed processing task."))
			})
			It("should call processListRepositories", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListRepositoriesTask
					task.Server = localServer

					call := listRepositories.EXPECT().Inc()
					expectedRepositories := []perun.Repository{
						"emulator",
						"standard",
					}
					for _, repository := range expectedRepositories {
						newtask := PrereleaseListImagesTask
						newtask.Server = localServer
						newtask.Repository = repository
						call = consumer.EXPECT().Run(newtask).After(call)
					}
					listRepositories.EXPECT().Dec().After(call)
					w.process(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] starts processing task."))
				Expect(log).To(ContainSubstring("Worker [67] completed processing task."))
			})
			It("should call processListImages", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListImagesTask
					task.Server = localServer

					call := listImages.EXPECT().Inc()
					expectedImages := []perun.ImageName{
						"common-boot-armv7l-artik10",
						"common-boot-armv7l-odroidu3",
						"common-boot-armv7l-odroidxu3",
						"common-wayland-3parts-arm64",
						"common-wayland-3parts-armv7l-artik",
						"common-wayland-3parts-armv7l-odroidu3",
						"common-wayland-3parts-armv7l-rpi3",
						"iot-boot-arm64-artik710",
						"iot-boot-arm64-rpi3",
						"iot-boot-armv7l-artik530",
						"iot-boot-armv7l-artik533s",
						"iot-gps_tracker-2parts-armv7l-rpi3",
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
						"tv-wayland-armv7l-odroidu3",
						"tv-wayland-armv7l-rpi3",
						"wearable-wayland-armv7l-tw1",
						"wearable-wayland-armv7l-tw2",
					}
					for _, image := range expectedImages {
						newtask := PrereleaseListFilesTask
						newtask.Server = localServer
						newtask.ImageName = image
						call = consumer.EXPECT().Run(newtask).After(call)
					}
					listImages.EXPECT().Dec().After(call)
					w.process(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] starts processing task."))
				Expect(log).To(ContainSubstring("Worker [67] completed processing task."))
			})
			It("should call processListFiles", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := PrereleaseListFilesTask
					task.Server = localServer

					file := `tizen-4.0-unified_20181031.1.20181107.094441_` +
						`mobile-wayland-arm64-tm2.tar.gz`
					newtask := PrereleaseGetFileInfoTask
					newtask.Server = localServer
					newtask.FileName = file
					gomock.InOrder(
						listFiles.EXPECT().Inc(),
						consumer.EXPECT().Run(newtask),
						listFiles.EXPECT().Dec(),
					)
					w.process(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] starts processing task."))
				Expect(log).To(ContainSubstring("Worker [67] completed processing task."))
			})
			It("should call processGetFileInfo", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					file := `prerelease/tizen/4.0-unified/tizen-4.0-unified_20181031.1/` +
						`tizen-4.0-unified_20181031.1.20181107.094441/` +
						`images/standard/mobile-wayland-arm64-tm2/` +
						`tizen-4.0-unified_20181031.1.20181107.094441_` +
						`mobile-wayland-arm64-tm2.tar.gz`
					info, err := os.Stat(testFilePath + "/" + file)
					Expect(err).NotTo(HaveOccurred())

					task := PrereleaseGetFileInfoTask
					task.Server = localServer

					gomock.InOrder(
						getFileInfo.EXPECT().Inc(),
						db.EXPECT().UpdateImage(gomock.Any()).Do(func(ifi *perun.ImageFileInfo) {
							Expect(ifi.URL).To(Equal("http://" + localServer + "/" + file))
							Expect(ifi.Image).To(Equal(task.Image))
							Expect(ifi.Info).NotTo(BeNil())
							Expect(ifi.Info.Length).To(Equal(info.Size()))
							Expect(ifi.Info.Modified).To(
								BeTemporally("~", info.ModTime(), time.Second))
						}),
						getFileInfo.EXPECT().Dec(),
					)
					w.process(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] starts processing task."))
				Expect(log).To(ContainSubstring("Worker [67] completed processing task."))
			})
			It("should log error for unknown task type", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := Task{TaskType: "unknown"}
					w.process(task)
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] starts processing task."))
				Expect(log).To(ContainSubstring("Unknown task type."))
				Expect(log).To(ContainSubstring("Worker [67] completed processing task."))
			})
		})
		Describe("start / loop", func() {
			It("should start and stop goroutine waiting for tasks", func() {
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					gomock.InOrder(
						workerIdle.EXPECT().Inc(),
						workerIdle.EXPECT().Dec(),
					)

					w.start()
					close(stop)
					grs.Done.Wait()
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] started."))
				Expect(log).To(ContainSubstring("Worker [67] finished."))
			})
			It("should process tasks", func() {
				const times = 5
				log, err := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					task := Task{TaskType: "unknown"}

					call := workerIdle.EXPECT().Inc()
					for i := 0; i < times; i++ {
						call = queueTasks.EXPECT().Dec().After(call)
						call = workerBusy.EXPECT().Inc().After(call)
						call = workerIdle.EXPECT().Dec().After(call)
						call = workerIdle.EXPECT().Inc().After(call)
						call = workerBusy.EXPECT().Dec().After(call)
					}
					workerIdle.EXPECT().Dec().After(call)

					w.start()
					for i := 0; i < times; i++ {
						tasks <- task
					}
					//TODO synchronized wait for tasks to be consumed
					close(stop)
					grs.Done.Wait()
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("Worker [67] started."))
				Expect(strings.Count(log, "Unknown task type.")).To(Equal(times))
				Expect(log).To(ContainSubstring("Worker [67] finished."))
			})
		})
	})
})
