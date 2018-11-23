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

/*
Package watcher provides service responsible for monitoring changes in binary images publication.
It crawls for configured servers and profiles and searches for images, than it notifies database
module about found files, their sizes and timestamps. It provides also several metrics for
monitoring status of the watcher service.

It is the part of Perun's solution for automated test lab supporting work of release engineers.

Usage

First you need to create a new Watcher structure with NewWatcher function passing:
path to configuration file and objects implementing DB and Metrics interfaces.

Then all you need to do is start watcher's service with Start function. It will do all the job
for you. It will create profile watchers, workers, tasks. It will run sub goroutines that will
periodically monitor configured network paths and supply database with information about found
images.

If you would like to stop service in a gentle way, please use Close function, that will take care
of all launched goroutines and allocated resources.

For future use, e.g. retriggering gathering data about some image or images branch, there is
Run function. It allows delegating tasks directly to watcher. They will be handled by workers
exactly in the same way as internally created tasks.

	// Create and start watcher's service.
	w, err := watcher.NewWatcher("/etc/perun/watcher.json", db, metrics)
	if err != nil {
		... // Handle the error.
	}
	w.Start()

	...

	// Add some additional tasks to be processed.
	select {
	case task := <-extraTasks:
		w.Run(task)

	...

	// Finish watcher's service.
	w.Close()

Sample configuration

Here is a sample configuration for few profiles monitoring both snapshot and prerelease images.
Please notice that every profile and type requires separate ProfileConfiguration

	{"profiles": [
			{"server": "download.tizen.org", "type": "snapshot", "profile": "unified",
				"period": "20s"},
			{"server": "download.tizen.org", "type": "snapshot", "profile": "unified-4.0",
				"period": "91s"},
			{"server": "download.tizen.org", "type": "snapshot", "profile": "common-3.0",
				"period": "92s"},
			{"server": "download.tizen.org", "type": "prerelease", "profile": "unified",
				"period": "15s"},
			{"server": "download.tizen.org", "type": "prerelease", "profile": "unified-4.0",
				"period": "64s"},
			{"server": "download.tizen.org", "type": "prerelease", "profile": "common-3.0",
				"period": "65s"}
		],
		"tasksqueue": 5000,
		"workers": 70
	}

Architecture

The heart of the watcher's package is Watcher structure, which creates and controls other parts
of watcher, and provides public API for starting and stopping whole watcher service.

The detailed number of monitored profiles, workers, etc. are written in the configuration file.
It is read into Configuration structure, which consist of general information such as number of
workers to be created or tasks queue size; and slice of ProfileConfiguration structures.
Every of ProfileConfiguration provides configuration for monitoring single profile and image type.

When a service is started a pool of workers is created. Every worker has its own goroutine that
waits for a task on a channel. If worker receives a task to be done, it uses crawler for making
and parsing proper http request. After analyzing results, worker creates new tasks and delegate
them to watcher's tasks queue with watcher's TaskConsumer interface (Run method).
Worker is aware of folder structure that images are published in. So this is the part that
implements logic of tasks interpreting and creating and for building valid URL paths for crawler.

Crawler does not contain any secret knowledge. It is a generic tool that can download a HTML with
HTTP GET method and return all links found on that page. It can also ask for file details with
HTTP HEAD method.

During service start, after workers are ready to process tasks, ProfileWatcher structures are
created and run in separate goroutines. They are sleepers that periodically wake up to delegate
a new task for scaning profile from the top to the bottom. Single ProfileWatcher takes care
of single ProfileConfiguration.

Metrics

Watcher registers set of prometheus gauges in metrics service (see
github.com/SamsungSLAV/perun/metrics and
github.com/prometheus/client_golang/prometheus packages).

All gauges are registered in "perun" namespace for submodule "watcher".
The gauges provide information about:

* currently running tasks - one gauge for each task type:
	current_listSnapshots
	current_listPrereleases
	current_listRepositories
	current_listImages
	current_listFiles
	current_getFileInfo

* number of workers by current status:
	worker_busy
	worker_idle

* queue parameters - current usage and total size:
	queued_tasks
	queue_size

*/
package watcher
