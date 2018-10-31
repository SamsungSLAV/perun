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
	"strings"
	"sync"

	"github.com/SamsungSLAV/perun/testutil"
	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("metrics", func() {
	var (
		lock      = sync.Locker(new(sync.Mutex))
		ctrl      *gomock.Controller
		metrics   *MockMetrics
		testError error
	)

	BeforeEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl = gomock.NewController(GinkgoT())
		metrics = NewMockMetrics(ctrl)

		testError = errors.New("testError")
	})
	AfterEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl.Finish()
	})
	Describe("registerMetric", func() {
		const (
			testName = "testName"
			testHelp = "testHelp"
		)
		It("should register new metric and set initial value to 0", func() {
			lock.Lock()
			gauge := NewMockGauge(ctrl)
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, testName, testHelp).
					Return(gauge, nil),
				gauge.EXPECT().Set(0.0),
			)
			lock.Unlock()
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				g, err := registerMetric(metrics, testName, testHelp)
				Expect(err).NotTo(HaveOccurred())
				Expect(g).To(Equal(gauge))
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
		It("should log and return error if registration of gauge fails", func() {
			lock.Lock()
			metrics.EXPECT().RegisterGauge(namespace, submodule, testName, testHelp).
				Return(nil, testError)
			lock.Unlock()
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				g, err := registerMetric(metrics, testName, testHelp)
				Expect(err).To(Equal(testError))
				Expect(g).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Error registering Gauge."))
			Expect(log).To(ContainSubstring(testError.Error()))
		})
	})
	Describe("running tasks helpers", func() {
		const taskType = TaskType("testTaskType")

		It("should add 'current_' prefix to task type", func() {
			str := name(taskType)
			Expect(str).To(Equal("current_" + taskType))
		})
		It("should create description of metric", func() {
			str := help(taskType)
			Expect(str).To(Equal("Number of " + taskType + " running tasks."))
		})
	})
	Describe("registerMetrics", func() {
		var (
			gaugeListSnapshots    *MockGauge
			gaugeListPrereleases  *MockGauge
			gaugeListRepositories *MockGauge
			gaugeListImages       *MockGauge
			gaugeListFiles        *MockGauge
			gaugeGetFileInfo      *MockGauge
			gaugeWorkerBusy       *MockGauge
			gaugeWorkerIdle       *MockGauge
			gaugeQueueTasks       *MockGauge
			gaugeQueueSize        *MockGauge
		)

		BeforeEach(func() {
			lock.Lock()
			defer lock.Unlock()

			gaugeListSnapshots = NewMockGauge(ctrl)
			gaugeListPrereleases = NewMockGauge(ctrl)
			gaugeListRepositories = NewMockGauge(ctrl)
			gaugeListImages = NewMockGauge(ctrl)
			gaugeListFiles = NewMockGauge(ctrl)
			gaugeGetFileInfo = NewMockGauge(ctrl)
			gaugeWorkerBusy = NewMockGauge(ctrl)
			gaugeWorkerIdle = NewMockGauge(ctrl)
			gaugeQueueTasks = NewMockGauge(ctrl)
			gaugeQueueSize = NewMockGauge(ctrl)
		})
		It("should successfully register all gauges", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(gaugeListImages, nil),
				gaugeListImages.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
					help(ListFiles)).Return(gaugeListFiles, nil),
				gaugeListFiles.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
					help(GetFileInfo)).Return(gaugeGetFileInfo, nil),
				gaugeGetFileInfo.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, workerBusy, workerBusyHelp).
					Return(gaugeWorkerBusy, nil),
				gaugeWorkerBusy.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, workerIdle, workerIdleHelp).
					Return(gaugeWorkerIdle, nil),
				gaugeWorkerIdle.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, queuedTasks, queuedTasksHelp).
					Return(gaugeQueueTasks, nil),
				gaugeQueueTasks.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, queueSize, queueSizeHelp).
					Return(gaugeQueueSize, nil),
				gaugeQueueSize.EXPECT().Set(0.0),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).NotTo(HaveOccurred())
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(Equal(gaugeListImages))
				Expect(gs.listFiles).To(Equal(gaugeListFiles))
				Expect(gs.getFileInfo).To(Equal(gaugeGetFileInfo))
				Expect(gs.workerBusy).To(Equal(gaugeWorkerBusy))
				Expect(gs.workerIdle).To(Equal(gaugeWorkerIdle))
				Expect(gs.queueTasks).To(Equal(gaugeQueueTasks))
				Expect(gs.queueSize).To(Equal(gaugeQueueSize))
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(BeEmpty())
		})
		It("should fail if queueSize metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(gaugeListImages, nil),
				gaugeListImages.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
					help(ListFiles)).Return(gaugeListFiles, nil),
				gaugeListFiles.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
					help(GetFileInfo)).Return(gaugeGetFileInfo, nil),
				gaugeGetFileInfo.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, workerBusy, workerBusyHelp).
					Return(gaugeWorkerBusy, nil),
				gaugeWorkerBusy.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, workerIdle, workerIdleHelp).
					Return(gaugeWorkerIdle, nil),
				gaugeWorkerIdle.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, queuedTasks, queuedTasksHelp).
					Return(gaugeQueueTasks, nil),
				gaugeQueueTasks.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, queueSize, queueSizeHelp).
					Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(Equal(gaugeListImages))
				Expect(gs.listFiles).To(Equal(gaugeListFiles))
				Expect(gs.getFileInfo).To(Equal(gaugeGetFileInfo))
				Expect(gs.workerBusy).To(Equal(gaugeWorkerBusy))
				Expect(gs.workerIdle).To(Equal(gaugeWorkerIdle))
				Expect(gs.queueTasks).To(Equal(gaugeQueueTasks))
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if queueTasks metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(gaugeListImages, nil),
				gaugeListImages.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
					help(ListFiles)).Return(gaugeListFiles, nil),
				gaugeListFiles.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
					help(GetFileInfo)).Return(gaugeGetFileInfo, nil),
				gaugeGetFileInfo.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, workerBusy, workerBusyHelp).
					Return(gaugeWorkerBusy, nil),
				gaugeWorkerBusy.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, workerIdle, workerIdleHelp).
					Return(gaugeWorkerIdle, nil),
				gaugeWorkerIdle.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, queuedTasks, queuedTasksHelp).
					Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(Equal(gaugeListImages))
				Expect(gs.listFiles).To(Equal(gaugeListFiles))
				Expect(gs.getFileInfo).To(Equal(gaugeGetFileInfo))
				Expect(gs.workerBusy).To(Equal(gaugeWorkerBusy))
				Expect(gs.workerIdle).To(Equal(gaugeWorkerIdle))
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if workerIdle metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(gaugeListImages, nil),
				gaugeListImages.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
					help(ListFiles)).Return(gaugeListFiles, nil),
				gaugeListFiles.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
					help(GetFileInfo)).Return(gaugeGetFileInfo, nil),
				gaugeGetFileInfo.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, workerBusy, workerBusyHelp).
					Return(gaugeWorkerBusy, nil),
				gaugeWorkerBusy.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, workerIdle, workerIdleHelp).
					Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(Equal(gaugeListImages))
				Expect(gs.listFiles).To(Equal(gaugeListFiles))
				Expect(gs.getFileInfo).To(Equal(gaugeGetFileInfo))
				Expect(gs.workerBusy).To(Equal(gaugeWorkerBusy))
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if workerBusy metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(gaugeListImages, nil),
				gaugeListImages.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
					help(ListFiles)).Return(gaugeListFiles, nil),
				gaugeListFiles.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
					help(GetFileInfo)).Return(gaugeGetFileInfo, nil),
				gaugeGetFileInfo.EXPECT().Set(0.0),

				metrics.EXPECT().RegisterGauge(namespace, submodule, workerBusy, workerBusyHelp).
					Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(Equal(gaugeListImages))
				Expect(gs.listFiles).To(Equal(gaugeListFiles))
				Expect(gs.getFileInfo).To(Equal(gaugeGetFileInfo))
				Expect(gs.workerBusy).To(BeNil())
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if getFileInfo metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(gaugeListImages, nil),
				gaugeListImages.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
					help(ListFiles)).Return(gaugeListFiles, nil),
				gaugeListFiles.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(GetFileInfo),
					help(GetFileInfo)).Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(Equal(gaugeListImages))
				Expect(gs.listFiles).To(Equal(gaugeListFiles))
				Expect(gs.getFileInfo).To(BeNil())
				Expect(gs.workerBusy).To(BeNil())
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if listFiles metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(gaugeListImages, nil),
				gaugeListImages.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListFiles),
					help(ListFiles)).Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(Equal(gaugeListImages))
				Expect(gs.listFiles).To(BeNil())
				Expect(gs.getFileInfo).To(BeNil())
				Expect(gs.workerBusy).To(BeNil())
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if listImages metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(gaugeListRepositories, nil),
				gaugeListRepositories.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListImages),
					help(ListImages)).Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(Equal(gaugeListRepositories))
				Expect(gs.listImages).To(BeNil())
				Expect(gs.listFiles).To(BeNil())
				Expect(gs.getFileInfo).To(BeNil())
				Expect(gs.workerBusy).To(BeNil())
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if listRepositories metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(gaugeListPrereleases, nil),
				gaugeListPrereleases.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListRepositories),
					help(ListRepositories)).Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(Equal(gaugeListPrereleases))
				Expect(gs.listRepositories).To(BeNil())
				Expect(gs.listImages).To(BeNil())
				Expect(gs.listFiles).To(BeNil())
				Expect(gs.getFileInfo).To(BeNil())
				Expect(gs.workerBusy).To(BeNil())
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if listPrereleases metric fails", func() {
			gomock.InOrder(
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
					help(ListSnapshots)).Return(gaugeListSnapshots, nil),
				gaugeListSnapshots.EXPECT().Set(0.0),
				metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListPrereleases),
					help(ListPrereleases)).Return(nil, testError),
			)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(Equal(gaugeListSnapshots))
				Expect(gs.listPrereleases).To(BeNil())
				Expect(gs.listRepositories).To(BeNil())
				Expect(gs.listImages).To(BeNil())
				Expect(gs.listFiles).To(BeNil())
				Expect(gs.getFileInfo).To(BeNil())
				Expect(gs.workerBusy).To(BeNil())
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
		It("should fail if listSnapshots metric fails", func() {
			metrics.EXPECT().RegisterGauge(namespace, submodule, name(ListSnapshots),
				help(ListSnapshots)).Return(nil, testError)
			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				gs := &gauges{}
				err := gs.registerMetrics(metrics)
				Expect(err).To(Equal(testError))
				Expect(gs.listSnapshots).To(BeNil())
				Expect(gs.listPrereleases).To(BeNil())
				Expect(gs.listRepositories).To(BeNil())
				Expect(gs.listImages).To(BeNil())
				Expect(gs.listFiles).To(BeNil())
				Expect(gs.getFileInfo).To(BeNil())
				Expect(gs.workerBusy).To(BeNil())
				Expect(gs.workerIdle).To(BeNil())
				Expect(gs.queueTasks).To(BeNil())
				Expect(gs.queueSize).To(BeNil())
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Count(log, "Error registering Gauge.")).To(Equal(1))
		})
	})
})
