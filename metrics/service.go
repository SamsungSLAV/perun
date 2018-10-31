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
	"net/http"
	"sync"

	"github.com/SamsungSLAV/slav/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics is a structure for running prometheus client service.
// It should always be created using public NewMetrics function.
// It implements Metrics interface.
type metrics struct {
	listenAddress string
	server        *http.Server
	serverDone    *sync.WaitGroup
	registered    map[string]prometheus.Collector
	mutex         sync.Locker
}

// newMetrics creates a new metrics structure with all fields initialized.
// It prepares HTTP server to be run.
// It is used by the public NewMetrics function.
func newMetrics(listenAddress string) (Metrics, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	metrics := &metrics{
		listenAddress: listenAddress,
		server: &http.Server{
			Addr:    listenAddress,
			Handler: mux,
		},
		serverDone: new(sync.WaitGroup),
		registered: make(map[string]prometheus.Collector),
		mutex:      new(sync.Mutex),
	}

	logger.WithProperty("listenAddress", metrics.listenAddress).Notice("New metrics created.")
	return metrics, nil
}

// Start starts a new goroutine running prometheus client service.
// It is the part of implementation of Metrics interface on metrics structure.
func (metrics *metrics) Start() {
	metrics.serverDone.Add(1)
	go func() {
		defer metrics.serverDone.Done()
		err := metrics.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.WithError(err).WithProperty("listenAddress", metrics.listenAddress).
				Critical("Running metrics HTTP server failed.")
		}
	}()
	logger.WithProperty("listenAddress", metrics.listenAddress).Notice("Metrics started.")
}

// Close stops HTTP service and waits until internal goroutine is completed.
// It is the part of implementation of Metrics interface on metrics structure.
func (metrics *metrics) Close() {
	// Stop HTTP server.
	err := metrics.server.Close()
	if err != nil {
		logger.WithError(err).WithProperty("listenAddress", metrics.listenAddress).
			Error("Error closing metrics HTTP server listeners.")
	}
	metrics.serverDone.Wait()

	// Unregister metrics.
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	for _, v := range metrics.registered {
		_ = prometheus.Unregister(v)
	}
	logger.WithProperty("listenAddress", metrics.listenAddress).Notice("Metrics closed.")
}

// RegisterGauge creates and registers new gauge metric.
// If a metric with the same name in the same subsystem has already been registered,
// ErrMetricAlreadyRegistered is returned as an error.
// If the name is unique a new gauge is created, registered in prometheus client
// and returned as a Gauge interface instance.
// An error could be returned also when registration in prometheus fails.
func (metrics *metrics) RegisterGauge(namespace, subsystem, name, help string) (Gauge, error) {
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()

	// Verify if name has already been registered
	combined := namespace + "/" + subsystem + "/" + name
	_, ok := metrics.registered[combined]
	if ok {
		return nil, ErrMetricAlreadyRegistered
	}

	promgauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		})
	err := prometheus.Register(promgauge)
	if err != nil {
		return nil, err
	}
	metrics.registered[combined] = promgauge

	logger.WithProperties(logger.Properties{
		"namespace":     namespace,
		"subsystem":     subsystem,
		"name":          name,
		"listenAddress": metrics.listenAddress,
	}).Info("New gauge registered.")
	return &gauge{
		gauge: promgauge,
		mutex: metrics.mutex,
	}, nil
}
