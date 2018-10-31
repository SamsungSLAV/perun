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

	"github.com/SamsungSLAV/perun/db"
	"github.com/SamsungSLAV/perun/metrics"
	"github.com/SamsungSLAV/perun/util"
	"github.com/SamsungSLAV/slav/logger"
)

// Watcher crawls given profile urls and stores information about found images in database.
type Watcher struct {
	// conf contains watcher's configuration.
	conf *Configuration

	// tickersStop is a channel used to notify profile watchers goroutines to break their job.
	tickersStop chan interface{}
	// tickersDone is waiting point for profile watchers goroutines to end.
	tickersDone *sync.WaitGroup

	// workersStop is a channel used to notify workers to break their job.
	workersStop chan interface{}
	// workersDone is waiting point for workers goroutines to end.
	workersDone *sync.WaitGroup

	// tasks define channel for passing new tasks to workers.
	tasks chan Task

	// db provides access to database for updating image files' status.
	db db.DB

	// gauges stores all watcher's gauge metrics.
	gauges gauges
}

// NewWatcher returns new Watcher with fields set and loaded configuration.
func NewWatcher(configurationPath string, db db.DB, metrics metrics.Metrics) (*Watcher, error) {
	conf, err := loadConfiguration(configurationPath)
	if err != nil {
		return nil, err
	}
	watcher := &Watcher{
		conf: conf,

		tickersStop: make(chan interface{}),
		tickersDone: new(sync.WaitGroup),

		workersStop: make(chan interface{}),
		workersDone: new(sync.WaitGroup),

		tasks: make(chan Task, conf.TasksQueue),

		db: db,
	}

	err = watcher.gauges.registerMetrics(metrics)
	if err != nil {
		return nil, err
	}
	watcher.gauges.queueSize.Set(float64(conf.TasksQueue))

	logger.Notice("New watcher created.")

	return watcher, nil
}

// Start runs watcher service.
func (watcher *Watcher) Start() {
	// Start worker goroutines.
	wgrs := util.GoRoutineSync{
		Stop: watcher.workersStop,
		Done: watcher.workersDone,
	}
	for id := 0; id < watcher.conf.Workers; id++ {
		newWorker(id, watcher.tasks, wgrs, watcher, watcher.db, watcher.gauges).start()
	}

	// Start profile watchers and goroutines.
	pwgrs := util.GoRoutineSync{
		Stop: watcher.tickersStop,
		Done: watcher.tickersDone,
	}
	for _, p := range watcher.conf.Profiles {
		newProfileWatcher(p, pwgrs, watcher).start()
	}

	logger.Notice("Watcher started.")
}

// Close frees resources associated with a watcher.
func (watcher *Watcher) Close() {
	// Stop profile watchers and wait for all goroutines to finish.
	close(watcher.tickersStop)
	watcher.tickersDone.Wait()

	// Stop workers and wait for all goroutines to finish.
	close(watcher.workersStop)
	watcher.workersDone.Wait()

	// Close tasks channel.
	close(watcher.tasks)

	logger.Notice("Watcher closed.")
}

// Run delegates a task to workers. It is the implementation of TaskConsumer interface.
func (watcher *Watcher) Run(task Task) {
	logger.WithProperty("task", task).Info("New task scheduled.")
	watcher.gauges.queueTasks.Inc()
	watcher.tasks <- task
}
