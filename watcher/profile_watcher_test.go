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
	"sync"
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/perun/testutil"
	"github.com/SamsungSLAV/perun/util"
	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("profileWatcher", func() {
	var (
		lock     = sync.Locker(new(sync.Mutex))
		ctrl     *gomock.Controller
		consumer *MockTaskConsumer

		testPeriod  = Period(time.Second)
		testProfile = "testProfile"
		testType    = "testType"
		testServer  = "testServer"

		pc = ProfileConfiguration{
			Server:  testServer,
			Type:    testType,
			Profile: testProfile,
			Period:  testPeriod,
		}
		testTask = Task{
			TaskType: ListSnapshots,
			Image: perun.Image{
				Server:    testServer,
				ImageType: testType,
				Profile:   testProfile,
			},
		}

		grs  util.GoRoutineSync
		stop chan interface{}
	)

	BeforeEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl = gomock.NewController(GinkgoT())
		consumer = NewMockTaskConsumer(ctrl)

		grs.Done = new(sync.WaitGroup)
		stop = make(chan interface{})
		grs.Stop = stop
	})
	AfterEach(func() {
		lock.Lock()
		defer lock.Unlock()

		ctrl.Finish()
	})
	Describe("newProfileWatcher", func() {
		It("should create new, properly initilialized profileWatcher structure", func() {
			lock.Lock()
			defer lock.Unlock()
			pw := newProfileWatcher(pc, grs, consumer)
			Expect(pw).NotTo(BeNil())
			Expect(pw.conf).To(Equal(pc))
			Expect(pw.grs).To(Equal(grs))
			Expect(pw.taskConsumer).To(Equal(consumer))
		})
	})
	Describe("start / loop", func() {
		It("should create ticker, run loop in separate goroutine and handle ticks", func() {
			lock.Lock()
			pw := newProfileWatcher(pc, grs, consumer)
			lock.Unlock()

			log, err := testutil.WithStderrMocked(func() {
				defer GinkgoRecover()
				lock.Lock()
				defer lock.Unlock()

				consumer.EXPECT().Run(testTask).Times(3)
				pw.start()
				time.Sleep(time.Millisecond * 3500)
				close(stop)
				grs.Done.Wait()
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Profile watcher started."))
			Expect(log).To(ContainSubstring("Profile watcher finished."))
		})
	})
})
