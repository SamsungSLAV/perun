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

// Package metrics provides an easy to use wrapper for using prometheus gauges.
// The proper usage requires:
// 1) Creation of Metrics interface instance with NewMetrics function.
//    The created instance will act as prometheus client providing an http service
//    listening on given interface, address and port for prometheus server requests.
// 2) Registration of metrics/gauges with RegisterGauge method of Metrics.
//    Multiple gauges can be registered. Every call will return a Gauge interface instance
//    for later operations.
// 3) Starting prometheus client service with Start method. This call will create HTTP service
//    and will start goroutines listening for incoming requests.
// 4) Using Gauge interface instances for controlling metrics by increasing, decreasing values
//    of gauges or explicit setting values.
// 5) Closing prometheus client with Close method of Metrics instance. The call will close
//    the server, send stop notification to internal goroutines and wait until they are completed.
//
// Internal implementation uses a default logger from github.com/SamsungSLAV/slav/logger package
// for logging errors and notifications.
package metrics

// Gauge represents a single Gauge metric for controlling a single numerical value.
type Gauge interface {
	// Inc increases gauge value by 1.
	Inc()
	// Dec decreases gauge value by 1.
	Dec()
	// Set changes gauge to specified value.
	Set(value float64)
}

// Metrics defines API for controlling prometheus client service.
type Metrics interface {
	// RegisterGauge creates and registers in prometheus new gauge metric.
	// In case of success it returns a Gauge interface instance for controlling
	// created metric.
	RegisterGauge(namespace, subsystem, name, help string) (Gauge, error)
	// Start starts prometheus client HTTP service.
	Start()
	// Close stops and cleans up the service.
	Close()
}

// NewMetrics is the only valid way of creating Metrics interface instance.
// The listenAddress parameter is a listening address and port for HTTP service,
// e.g. ":7378". The prometheus service should be configured to poll metrics from
// this address.
// This is a public wrapper, presented here for keeping whole metrics package API in single file.
func NewMetrics(listenAddress string) (Metrics, error) {
	return newMetrics(listenAddress)
}
