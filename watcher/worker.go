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
	"strings"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/perun/db"
	"github.com/SamsungSLAV/perun/util"
	"github.com/SamsungSLAV/slav/logger"
)

// worker is responsible for running time consuming tasks in goroutine.
type worker struct {
	// id is a worker identifier used for logging purposes.
	id int
	// tasks is a channel providing tasks to be executed by worker.
	tasks <-chan Task
	// grs allows synchronization of worker's goroutine.
	grs util.GoRoutineSync
	// taskConsumer consumes created tasks.
	taskConsumer TaskConsumer
	// db provides access to database for updating image files' status.
	db db.DB

	gauges gauges
}

// newWorker creates a new worker structure with all fields initialized.
func newWorker(id int, tasks <-chan Task, grs util.GoRoutineSync, taskConsumer TaskConsumer,
	db db.DB, gauges gauges) *worker {
	return &worker{
		id:           id,
		tasks:        tasks,
		grs:          grs,
		taskConsumer: taskConsumer,
		db:           db,
		gauges:       gauges,
	}
}

// Start runs worker's goroutine.
func (w *worker) start() {
	w.grs.Done.Add(1)
	go w.loop()
}

// loop reacts on incoming tasks and processes them.
func (w *worker) loop() {
	defer w.grs.Done.Done()
	defer logger.Noticef("Worker [%d] finished.", w.id)
	logger.Noticef("Worker [%d] started.", w.id)

	defer w.gauges.workerIdle.Dec()
	w.gauges.workerIdle.Inc()

	for {
		select {
		case <-w.grs.Stop:
			return
		case t := <-w.tasks:
			w.gauges.queueTasks.Dec()
			w.gauges.workerBusy.Inc()
			w.gauges.workerIdle.Dec()
			w.process(t)
			w.gauges.workerIdle.Inc()
			w.gauges.workerBusy.Dec()
		}
	}
}

// process a single task.
func (w *worker) process(task Task) {
	defer logger.WithProperty("task", task).Infof("Worker [%d] completed processing task.", w.id)
	logger.WithProperty("task", task).Infof("Worker [%d] starts processing task.", w.id)

	switch task.TaskType {
	case ListSnapshots:
		w.processListSnapshots(task)
	case ListPrereleases:
		w.processListPrereleases(task)
	case ListRepositories:
		w.processListRepositories(task)
	case ListImages:
		w.processListImages(task)
	case ListFiles:
		w.processListFiles(task)
	case GetFileInfo:
		w.processGetFileInfo(task)
	default:
		logger.WithProperty("task", task).Error("Unknown task type.")
		return
	}
}

// buildURL creates an URL based on image and task types.
func (w *worker) buildURL(task Task) string {
	const urlPrefix = "http://download.tizen.org"
	var url string
	switch task.ImageType {
	case perun.SNAPSHOT:
		url = urlPrefix + "/snapshots/tizen/" + task.Profile + "/"
		if len(task.Snapshot) > 0 {
			url += "tizen-" + task.Profile + "_" + task.Snapshot + "/images/"
		}
	case perun.PRERELEASE:
		url = urlPrefix + "/prerelease/tizen/" + task.Profile + "/"
		if len(task.Snapshot) > 0 {
			url += "tizen-" + task.Profile + "_" + task.Snapshot + "/"
		}
	default:
		logger.WithProperty("imageType", task.ImageType).Warning("Invalid image type in task")
		return ""
	}
	if len(task.Prerelease) > 0 {
		url += "tizen-" + task.Profile + "_" + task.Snapshot + "." + task.Prerelease + "/images/"
	}
	if len(task.Repository) > 0 {
		url += task.Repository + "/"
	}
	if len(task.ImageName) > 0 {
		url += task.ImageName + "/"
	}
	if len(task.FileName) > 0 {
		url += task.FileName
	}
	return url
}

// processListSnapshots acquires list of snapshot images and for each of it delegates a new task:
// * listRepositories (in case of snapshot images)
// * listPrereleases (in case of prerelease images)
func (w *worker) processListSnapshots(task Task) {
	defer w.gauges.listSnapshots.Dec()
	w.gauges.listSnapshots.Inc()

	url := w.buildURL(task)
	if len(url) == 0 {
		return
	}
	links := newCrawler(url).getLinks()
	for _, l := range links {
		switch task.ImageType {
		case perun.SNAPSHOT:
			task.TaskType = ListRepositories
		case perun.PRERELEASE:
			task.TaskType = ListPrereleases
		}
		task.Snapshot = strings.TrimPrefix(l, "tizen-"+task.Profile+"_")
		w.taskConsumer.Run(task)
	}
}

// processListPrereleases acquires list of published prerelease projects for a reference snapshot.
// For each prerelease project found a new task for listing repositories is created.
// This type of task is run only for prereleases image type.
func (w *worker) processListPrereleases(task Task) {
	defer w.gauges.listPrereleases.Dec()
	w.gauges.listPrereleases.Inc()

	url := w.buildURL(task)
	if len(url) == 0 {
		return
	}
	links := newCrawler(url).getLinks()
	for _, l := range links {
		task.TaskType = ListRepositories
		task.Prerelease = strings.TrimPrefix(l, "tizen-"+task.Profile+"_"+task.Snapshot+".")
		w.taskConsumer.Run(task)
	}
}

// processListRepositories acquires list of published repositories and for each of them creates
// a new task for listing published images.
func (w *worker) processListRepositories(task Task) {
	defer w.gauges.listRepositories.Dec()
	w.gauges.listRepositories.Inc()

	url := w.buildURL(task)
	if len(url) == 0 {
		return
	}
	links := newCrawler(url).getLinks()
	for _, l := range links {
		task.TaskType = ListImages
		task.Repository = l
		w.taskConsumer.Run(task)
	}
}

// processListImages acquires list of published images and for each of them creates a new task
// for listing published files.
func (w *worker) processListImages(task Task) {
	defer w.gauges.listImages.Dec()
	w.gauges.listImages.Inc()

	url := w.buildURL(task)
	if len(url) == 0 {
		return
	}
	links := newCrawler(url).getLinks()
	for _, l := range links {
		task.TaskType = ListFiles
		task.ImageName = l
		w.taskConsumer.Run(task)
	}
}

// processListFiles acquires list of published image files using filter in crawler.
// For every found image a getFileInfo task is created.
func (w *worker) processListFiles(task Task) {
	defer w.gauges.listFiles.Dec()
	w.gauges.listFiles.Inc()

	url := w.buildURL(task)
	if len(url) == 0 {
		return
	}
	links := newCrawler(url).requireSuffixes(defaultImageSuffixes).getLinks()
	for _, l := range links {
		task.TaskType = GetFileInfo
		task.FileName = l
		w.taskConsumer.Run(task)
	}
}

// processGetFileInfo gets acquires information about a single image file
// and updates images database with this information.
func (w *worker) processGetFileInfo(task Task) {
	defer w.gauges.getFileInfo.Dec()
	w.gauges.getFileInfo.Inc()

	url := w.buildURL(task)
	if len(url) == 0 {
		return
	}
	info := newCrawler(url).getFileInfo()
	if info == nil {
		return
	}

	ifi := &perun.ImageFileInfo{
		URL:   url,
		Image: task.Image,
		Info:  *info,
	}
	err := w.db.UpdateImage(ifi)
	if err != nil {
		logger.WithProperty("imageFileInfo", *ifi).Error("Error updating image in database.")
	}
}
