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

package metrics

import (
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("gauge", func() {
	var (
		ctrl      *gomock.Controller
		locker    *MockLocker
		promGauge *MockGauge

		g *gauge
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		locker = NewMockLocker(ctrl)
		promGauge = NewMockGauge(ctrl)

		g = &gauge{
			gauge: promGauge,
			mutex: locker,
		}
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	It("Inc", func() {
		gomock.InOrder(
			locker.EXPECT().Lock(),
			promGauge.EXPECT().Inc(),
			locker.EXPECT().Unlock(),
		)
		g.Inc()
	})
	It("Dec", func() {
		gomock.InOrder(
			locker.EXPECT().Lock(),
			promGauge.EXPECT().Dec(),
			locker.EXPECT().Unlock(),
		)
		g.Dec()
	})
	It("Set", func() {
		const value = 67.54321
		gomock.InOrder(
			locker.EXPECT().Lock(),
			promGauge.EXPECT().Set(value),
			locker.EXPECT().Unlock(),
		)
		g.Set(value)
	})
})
