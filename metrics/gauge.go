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
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// gauge is an internal metrics package wrap for prometheus.Gauge.
// It uses metrics mutex for securing access to gauges.
// It implements Gauge interface.
type gauge struct {
	gauge prometheus.Gauge
	mutex sync.Locker
}

// Inc increases gauge value by 1.
func (gauge *gauge) Inc() {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()

	gauge.gauge.Inc()
}

// Dec decreases gauge value by 1.
func (gauge *gauge) Dec() {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()

	gauge.gauge.Dec()
}

// Set sets the gauge value.
func (gauge *gauge) Set(value float64) {
	gauge.mutex.Lock()
	defer gauge.mutex.Unlock()

	gauge.gauge.Set(value)
}
