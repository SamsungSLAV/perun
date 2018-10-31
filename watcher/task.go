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

import "github.com/SamsungSLAV/perun"

// TaskType defines type of task, which defines way of creating url and parsing it.
type TaskType = string

const (
	// ListSnapshots gets list of snapshots.
	ListSnapshots TaskType = "ListSnapshots"
	// ListPrereleases gets list of prerelease images. It is used only for prerelease images.
	ListPrereleases TaskType = "ListPrereleases"
	// ListRepositories gets list of OBS repositories for the project.
	ListRepositories TaskType = "ListRepositories"
	// ListImages gets list of folders with published images.
	ListImages TaskType = "ListImages"
	// ListFiles gets the list of files in image publication folder.
	ListFiles TaskType = "ListFiles"
	// GetFileInfo acquires information about image file.
	GetFileInfo TaskType = "GetFileInfo"
)

// Task describes task for searching or getting information about image. Tasks are run by workers.
// The TaskType contains information about type of task to be done.
// The Image contains all the information allowing to create proper URL and acquire information
// about image.
type Task struct {
	TaskType
	perun.Image
}

// TaskConsumer defines API for passing tasks ready to be executed. It is singled out fo future use
// in cli programs or HTTP services allowing to delegate tasks to perun's watcher on external
// request.
type TaskConsumer interface {
	// Run delegates a task to be run by perun watcher's workers.
	Run(Task)
}
