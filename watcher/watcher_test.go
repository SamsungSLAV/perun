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
)

var _ = Describe("worker", func() {

	const (
		validConfiguration = `{"profiles":[` +
			`{"server": "abc", "type": "prerelease", "profile": "common-3.0", "period": "65s"},` +
			`{"server": "def", "type": "snapshot", "profile": "unified", "period": "2m"}` +
			`], "tasksqueue": 3000, "workers": 50}`
		tasksQueueSize = float64(3000.0)
		workers        = 50
	)
	var (
		lock              = sync.Locker(new(sync.Mutex))
		ctrl              *gomock.Controller
		db                *MockDB
		metrics           *MockMetrics
		listSnapshotsG    *MockGauge
		listPrereleasesG  *MockGauge
		listRepositoriesG *MockGauge
		listImagesG       *MockGauge
		listFilesG        *MockGauge
		getFileInfoG      *MockGauge
		workerBusyG       *MockGauge
		workerIdleG       *MockGauge
		queueTasksG       *MockGauge
		queueSizeG        *MockGauge
		testError         error
	)

	BeforeEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl = gomock.NewController(GinkgoT())
		db = NewMockDB(ctrl)
		metrics = NewMockMetrics(ctrl)
		listSnapshotsG = NewMockGauge(ctrl)
		listPrereleasesG = NewMockGauge(ctrl)
		listRepositoriesG = NewMockGauge(ctrl)
		listImagesG = NewMockGauge(ctrl)
		listFilesG = NewMockGauge(ctrl)
		getFileInfoG = NewMockGauge(ctrl)
		workerBusyG = NewMockGauge(ctrl)
		workerIdleG = NewMockGauge(ctrl)
		queueTasksG = NewMockGauge(ctrl)
		queueSizeG = NewMockGauge(ctrl)
		testError = errors.New("testError")
	})
	AfterEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl.Finish()
	})

	Describe("NewWatcher", func() {
		var (
			tempFile *os.File
		)
		BeforeEach(func() {
			lock.Lock()
			defer lock.Unlock()
			var err error
			tempFile, err = ioutil.TempFile("", "Watcher_tests_tempFile")
			Expect(err).NotTo(HaveOccurred())
			_, err = tempFile.Write([]byte(validConfiguration))
			Expect(err).NotTo(HaveOccurred())
			err = tempFile.Close()
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			os.Remove(tempFile.Name())
		})

		It("should fail if configuration file path is invalid", func() {
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				watcher, err := NewWatcher("/such/file/should/not/exist", db, metrics)
				Expect(err).To(HaveOccurred())
				Expect(watcher).To(BeNil())
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Error loading watcher's configuration file."))
		})
		It("should fail if metrics cannot be registered", func() {
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(nil, testError)
				watcher, err := NewWatcher(tempFile.Name(), db, metrics)
				Expect(err).To(Equal(testError))
				Expect(watcher).To(BeNil())
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Error registering Gauge."))
		})
		It("should create valid watcher structure with initialized metrics", func() {
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gomock.InOrder(
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
						help(ListSnapshots)).Return(listSnapshotsG, nil),
					listSnapshotsG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
						help(ListPrereleases)).Return(listPrereleasesG, nil),
					listPrereleasesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
						help(ListRepositories)).Return(listRepositoriesG, nil),
					listRepositoriesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
						help(ListImages)).Return(listImagesG, nil),
					listImagesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
						help(ListFiles)).Return(listFilesG, nil),
					listFilesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
						help(GetFileInfo)).Return(getFileInfoG, nil),
					getFileInfoG.EXPECT().Set(0.0),

					metrics.EXPECT().RegisterGauge(namespace, submodule, workerBusy,
						workerBusyHelp).Return(workerBusyG, nil),
					workerBusyG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, workerIdle,
						workerIdleHelp).Return(workerIdleG, nil),
					workerIdleG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, queuedTasks,
						queuedTasksHelp).Return(queueTasksG, nil),
					queueTasksG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, queueSize,
						queueSizeHelp).Return(queueSizeG, nil),
					queueSizeG.EXPECT().Set(0.0),

					queueSizeG.EXPECT().Set(tasksQueueSize),
				)
				watcher, err := NewWatcher(tempFile.Name(), db, metrics)
				Expect(err).NotTo(HaveOccurred())
				Expect(watcher).NotTo(BeNil())

				Expect(watcher.conf).NotTo(BeNil())
				Expect(len(watcher.conf.Profiles)).To(Equal(2))
				Expect(watcher.conf.Profiles[0]).To(Equal(ProfileConfiguration{
					Server:  "abc",
					Type:    perun.PRERELEASE,
					Profile: "common-3.0",
					Period:  Period(time.Second * 65),
				}))
				Expect(watcher.conf.Profiles[1]).To(Equal(ProfileConfiguration{
					Server:  "def",
					Type:    perun.SNAPSHOT,
					Profile: "unified",
					Period:  Period(time.Minute * 2),
				}))
				Expect(watcher.conf.TasksQueue).To(Equal(3000))
				Expect(watcher.conf.Workers).To(Equal(50))

				Expect(watcher.tickersStop).NotTo(BeNil())
				Expect(watcher.tickersDone).NotTo(BeNil())
				Expect(watcher.workersStop).NotTo(BeNil())
				Expect(watcher.workersDone).NotTo(BeNil())
				Expect(watcher.tasks).NotTo(BeNil())
				Expect(watcher.db).To(Equal(db))
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("New watcher created."))
		})
	})
	Describe("with Watcher initialized", func() {
		var (
			tempFile *os.File
			watcher  *Watcher
		)
		BeforeEach(func() {
			log, logErr := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()
				var err error
				tempFile, err = ioutil.TempFile("", "Watcher_tests_tempFile")
				Expect(err).NotTo(HaveOccurred())
				_, err = tempFile.Write([]byte(validConfiguration))
				Expect(err).NotTo(HaveOccurred())
				err = tempFile.Close()
				Expect(err).NotTo(HaveOccurred())
				gomock.InOrder(
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
						help(ListSnapshots)).Return(listSnapshotsG, nil),
					listSnapshotsG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
						help(ListPrereleases)).Return(listPrereleasesG, nil),
					listPrereleasesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
						help(ListRepositories)).Return(listRepositoriesG, nil),
					listRepositoriesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
						help(ListImages)).Return(listImagesG, nil),
					listImagesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
						help(ListFiles)).Return(listFilesG, nil),
					listFilesG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
						help(GetFileInfo)).Return(getFileInfoG, nil),
					getFileInfoG.EXPECT().Set(0.0),

					metrics.EXPECT().RegisterGauge(namespace, submodule, workerBusy,
						workerBusyHelp).Return(workerBusyG, nil),
					workerBusyG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, workerIdle,
						workerIdleHelp).Return(workerIdleG, nil),
					workerIdleG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, queuedTasks,
						queuedTasksHelp).Return(queueTasksG, nil),
					queueTasksG.EXPECT().Set(0.0),
					metrics.EXPECT().RegisterGauge(namespace, submodule, queueSize,
						queueSizeHelp).Return(queueSizeG, nil),
					queueSizeG.EXPECT().Set(0.0),

					queueSizeG.EXPECT().Set(tasksQueueSize),
				)
				watcher, err = NewWatcher(tempFile.Name(), db, metrics)
				Expect(err).NotTo(HaveOccurred())
				Expect(watcher).NotTo(BeNil())
			})
			Expect(logErr).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Watcher's configuration file loaded."))
			Expect(log).To(ContainSubstring("New watcher created."))
		})
		AfterEach(func() {
			os.Remove(tempFile.Name())
		})
		Describe("Run", func() {
			It("should log, increase metric and queue task", func() {
				log, logErr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					testTask := Task{TaskType: "testTask"}
					queueTasksG.EXPECT().Inc()
					watcher.Run(testTask)
					Eventually(watcher.tasks).Should(Receive(Equal(testTask)))
				})
				Expect(logErr).NotTo(HaveOccurred())
				Expect(log).To(ContainSubstring("New task scheduled."))
			})
			It("should drop tasks if queue is full", func() {
				lock.Lock()
				testTask := Task{TaskType: "testTask"}
				q := watcher.conf.TasksQueue
				d := q / 2
				lock.Unlock()

				log, logErr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					queueTasksG.EXPECT().Inc().Times(q + d)
					queueTasksG.EXPECT().Dec().Times(d)

					for i := 0; i < q+d; i++ {
						watcher.Run(testTask)
					}
					for i := 0; i < q; i++ {
						Eventually(watcher.tasks).Should(Receive(Equal(testTask)))
					}
				})
				lock.Lock()
				defer lock.Unlock()
				Expect(logErr).NotTo(HaveOccurred())
				Expect(strings.Count(log, "New task scheduled.")).To(Equal(q))
				Expect(strings.Count(log, "Task dropped. Tasks queue overflow.")).To(Equal(d))
			})
		})
		Describe("Start / Close", func() {
			It("should start and stop workers, profile watchers", func() {
				log, logErr := testutil.WithStderrMocked(func() {
					defer GinkgoRecover()
					lock.Lock()
					defer lock.Unlock()

					workerIdleG.EXPECT().Inc().Times(workers)
					watcher.Start()

					workerIdleG.EXPECT().Dec().Times(workers)
					watcher.Close()
				})
				Expect(logErr).NotTo(HaveOccurred())
				for id := 0; id < workers; id++ {
					str := fmt.Sprintf("Worker [%d] started.", id)
					Expect(log).To(ContainSubstring(str))
				}
				Expect(strings.Count(log, "Profile watcher started.")).To(Equal(2))
				Expect(log).To(ContainSubstring("Watcher started."))
				Expect(strings.Count(log, "Profile watcher finished.")).To(Equal(2))
				for id := 0; id < workers; id++ {
					str := fmt.Sprintf("Worker [%d] finished.", id)
					Expect(log).To(ContainSubstring(str))
				}
				Expect(log).To(ContainSubstring("Watcher closed."))
			})
		})
	})
})
