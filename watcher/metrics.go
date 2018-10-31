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
	"github.com/SamsungSLAV/perun/metrics"
	"github.com/SamsungSLAV/slav/logger"
)

const (
	namespace = "perun"
	submodule = "watcher"
	// metric names:
	workerBusy  = "worker_busy"
	workerIdle  = "worker_idle"
	queuedTasks = "queued_tasks"
	queueSize   = "queue_size"
	// metric helps:
	workerBusyHelp  = "Number of busy workers."
	workerIdleHelp  = "Number of idle workers."
	queuedTasksHelp = "Number of tasks pending in queue."
	queueSizeHelp   = "Size of the task queue."
)

// gauges structure aggregates all gauges metrics used in watcher module for easy access in code.
type gauges struct {
	// metrics related to running tasks.
	listSnapshots    metrics.Gauge
	listPrereleases  metrics.Gauge
	listRepositories metrics.Gauge
	listImages       metrics.Gauge
	listFiles        metrics.Gauge
	getFileInfo      metrics.Gauge

	// metrics related to workers' state.
	workerBusy metrics.Gauge
	workerIdle metrics.Gauge

	// metrics related to pending tasks queue.
	queueTasks metrics.Gauge
	queueSize  metrics.Gauge
}

// registerMetric registers a single gauge metric with given name and help.
// It also sets initial value of gauge to 0.
func registerMetric(metrics metrics.Metrics, name, help string) (metrics.Gauge, error) {
	g, err := metrics.RegisterGauge(namespace, submodule, name, help)
	if err != nil {
		logger.WithError(err).Error("Error registering Gauge.")
		return nil, err
	}
	g.Set(0.0)
	return g, nil
}

// name creates a gauge name for running tasks metric based on task type.
func name(t TaskType) string {
	return "current_" + t
}

// help creates a help description for running tasks metric based on task type.
func help(t TaskType) string {
	return "Number of " + t + " running tasks."
}

// registerMetrics registers all perun watcher's metrics.
func (gauges *gauges) registerMetrics(metrics metrics.Metrics) error {
	var err error

	gauges.listSnapshots, err = registerMetric(metrics, name(ListSnapshots), help(ListSnapshots))
	if err != nil {
		return err
	}
	gauges.listPrereleases, err = registerMetric(metrics, name(ListPrereleases),
		help(ListPrereleases))
	if err != nil {
		return err
	}
	gauges.listRepositories, err = registerMetric(metrics, name(ListRepositories),
		help(ListRepositories))
	if err != nil {
		return err
	}
	gauges.listImages, err = registerMetric(metrics, name(ListImages), help(ListImages))
	if err != nil {
		return err
	}
	gauges.listFiles, err = registerMetric(metrics, name(ListFiles), help(ListFiles))
	if err != nil {
		return err
	}
	gauges.getFileInfo, err = registerMetric(metrics, name(GetFileInfo), help(GetFileInfo))
	if err != nil {
		return err
	}

	gauges.workerBusy, err = registerMetric(metrics, workerBusy, workerBusyHelp)
	if err != nil {
		return err
	}
	gauges.workerIdle, err = registerMetric(metrics, workerIdle, workerIdleHelp)
	if err != nil {
		return err
	}

	gauges.queueTasks, err = registerMetric(metrics, queuedTasks, queuedTasksHelp)
	if err != nil {
		return err
	}
	gauges.queueSize, err = registerMetric(metrics, queueSize, queueSizeHelp)
	return err
}
