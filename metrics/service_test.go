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
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/SamsungSLAV/slav/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func httpClient(url string) string {
	resp, err := http.Get(url)
	Expect(err).NotTo(HaveOccurred())

	defer resp.Body.Close()

	size := 65536
	data := make([]byte, size)
	resp.Body.Read(data)
	return string(data)
}

type dummyListener struct {
	wg   *sync.WaitGroup
	once *sync.Once
}

func newDummyListener() *dummyListener {
	dl := &dummyListener{
		wg:   new(sync.WaitGroup),
		once: new(sync.Once),
	}
	dl.wg.Add(1)
	return dl
}

const dummyListenerTestError = "dummyListenerTestError"

func (dl *dummyListener) Accept() (net.Conn, error) {
	dl.once.Do(func() {
		dl.wg.Done()
	})
	return nil, &dummyNetError{}
}
func (*dummyListener) Close() error {
	return errors.New(dummyListenerTestError)
}
func (*dummyListener) Addr() net.Addr {
	return nil
}

func (dl *dummyListener) wait() {
	dl.wg.Wait()
}

type dummyNetError struct{}

func (*dummyNetError) Error() string {
	return "net error"
}
func (*dummyNetError) Timeout() bool {
	return false
}
func (*dummyNetError) Temporary() bool {
	return true
}

var _ = Describe("service", func() {
	var (
		oldHTTPMux *http.ServeMux
		newHTTPMux *http.ServeMux
	)

	BeforeEach(func() {
		logger.SetThreshold(logger.DebugLevel)

		newHTTPMux = http.NewServeMux()
		oldHTTPMux = http.DefaultServeMux
		http.DefaultServeMux = newHTTPMux

	})
	AfterEach(func() {
		http.DefaultServeMux = oldHTTPMux
	})
	Describe("newMetrics", func() {
		It("should create new metrics structure with all fields initialized", func() {
			laddr := "10.13.131.250:3333"
			log := withStderrMocked(func() {
				defer GinkgoRecover()
				m, err := newMetrics(laddr)
				Expect(err).NotTo(HaveOccurred())
				Expect(m).NotTo(BeNil())
				Expect(m.(*metrics).listenAddress).To(Equal(laddr))
				Expect(m.(*metrics).server).NotTo(BeNil())
				Expect(m.(*metrics).serverDone).NotTo(BeNil())
				Expect(m.(*metrics).registered).To(BeEmpty())
				Expect(m.(*metrics).mutex).NotTo(BeNil())
			})
			Expect(log).To(ContainSubstring("New metrics created."))
		})
		It("NewMetric wrapper should create new metrics structure with all fields initialized",
			func() {
				laddr := "10.13.131.250:3333"
				log := withStderrMocked(func() {
					defer GinkgoRecover()
					m, err := NewMetrics(laddr)
					Expect(err).NotTo(HaveOccurred())
					Expect(m).NotTo(BeNil())
					Expect(m.(*metrics).listenAddress).To(Equal(laddr))
					Expect(m.(*metrics).server).NotTo(BeNil())
					Expect(m.(*metrics).serverDone).NotTo(BeNil())
					Expect(m.(*metrics).registered).To(BeEmpty())
					Expect(m.(*metrics).mutex).NotTo(BeNil())
				})
				Expect(log).To(ContainSubstring("New metrics created."))
			},
		)
	})
	Describe("Start/Close", func() {
		It("should start and close server properly", func() {
			laddr := ":12345"
			log := withStderrMocked(func() {
				defer GinkgoRecover()
				m, err := NewMetrics(laddr)
				Expect(err).NotTo(HaveOccurred())

				m.Start()
				defer m.Close()
				time.Sleep(time.Millisecond * 500)

				ret := httpClient("http://127.0.0.1" + laddr + "/metrics")
				Expect(ret).To(ContainSubstring("# HELP go_goroutines Number of goroutines" +
					" that currently exist."))
			})
			Expect(log).To(ContainSubstring("New metrics created."))
			Expect(log).To(ContainSubstring("Metrics started."))
			Expect(log).To(ContainSubstring("Metrics closed."))
		})
		It("should log error if server cannot be started", func() {
			laddr := ":nosuchport"
			log := withStderrMocked(func() {
				defer GinkgoRecover()
				m, err := NewMetrics(laddr)
				Expect(err).NotTo(HaveOccurred())

				m.Start()
				defer m.Close()
			})
			Expect(log).To(ContainSubstring("Running metrics HTTP server failed."))
			Expect(log).To(ContainSubstring("listen tcp: address tcp/nosuchport: unknown port"))
		})
		It("should log error on Close if server's listeners failed to close properly", func() {
			laddr := ":12345"
			log := withStderrMocked(func() {
				defer GinkgoRecover()
				m, err := NewMetrics(laddr)
				Expect(err).NotTo(HaveOccurred())

				dl := newDummyListener()
				go m.(*metrics).server.Serve(dl)
				defer m.Close()

				dl.wait()
			})
			Expect(log).To(ContainSubstring("Error closing metrics HTTP server listeners."))
			Expect(log).To(ContainSubstring(dummyListenerTestError))
		})
	})
	Describe("RegisterGauge", func() {
		const (
			namespace = "testNamespace"
			subsystem = "testSubsystem"
			name      = "testName"
			help      = "testHelp"
		)
		It("should register gauge", func() {
			laddr := ":12345"
			log := withStderrMocked(func() {
				defer GinkgoRecover()
				m, err := NewMetrics(laddr)
				Expect(err).NotTo(HaveOccurred())

				G, err := m.RegisterGauge(namespace, subsystem, name, help)
				Expect(err).NotTo(HaveOccurred())
				Expect(G).NotTo(BeNil())

				m.Start()
				defer m.Close()
				time.Sleep(time.Millisecond * 500)

				ret := httpClient("http://127.0.0.1" + laddr + "/metrics")
				Expect(ret).To(ContainSubstring("testNamespace_testSubsystem_testName 0"))
			})
			Expect(log).To(ContainSubstring("New gauge registered."))
		})
		It("should not register same metric twice", func() {
			laddr := ":12345"
			withStderrMocked(func() {
				defer GinkgoRecover()
				m, err := NewMetrics(laddr)
				Expect(err).NotTo(HaveOccurred())

				G, err := m.RegisterGauge(namespace, subsystem, name, help)
				Expect(err).NotTo(HaveOccurred())
				Expect(G).NotTo(BeNil())

				G, err = m.RegisterGauge(namespace, subsystem, name, help)
				Expect(err).To(Equal(ErrMetricAlreadyRegistered))
				Expect(G).To(BeNil())
			})
		})
		It("should fail to register metric not allowed by prometheus", func() {
			laddr := ":12345"
			withStderrMocked(func() {
				defer GinkgoRecover()
				m, err := NewMetrics(laddr)
				Expect(err).NotTo(HaveOccurred())

				G, err := m.RegisterGauge("", "", "!@%^%", "")
				Expect(err).To(HaveOccurred())
				Expect(G).To(BeNil())
			})
		})
	})
})
