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
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/perun/util"
	"github.com/SamsungSLAV/slav/logger"
)

// profileWatcher watches a single profile by running ticker and delegates
// tasks to TaskConsumer, every configured period.
type profileWatcher struct {
	// conf is a configuration of the profile watcher containing profile type and name,
	// and a ticker period.
	conf ProfileConfiguration
	// grs allows synchronization of profile watcher's goroutine.
	grs util.GoRoutineSync
	// taskConsumer consumes created tasks.
	taskConsumer TaskConsumer
}

// newProfileWatcher creates a new profileWatcher structure with all fields initialized.
func newProfileWatcher(pc ProfileConfiguration, grs util.GoRoutineSync,
	taskConsumer TaskConsumer) *profileWatcher {
	return &profileWatcher{
		conf:         pc,
		grs:          grs,
		taskConsumer: taskConsumer,
	}
}

// start sets up ticker and creates a goroutine for handling tick events.
func (pw *profileWatcher) start() {
	ticker := time.NewTicker(time.Duration(pw.conf.Period))
	pw.grs.Done.Add(1)
	go pw.loop(ticker)
}

// loop reacts on ticker events, creates tasks and delegates them to tasks consumer.
func (pw *profileWatcher) loop(ticker *time.Ticker) {
	defer pw.grs.Done.Done()
	defer logger.WithProperty("type", pw.conf.Type).WithProperty("profile", pw.conf.Profile).
		Notice("Profile watcher finished.")
	logger.WithProperty("type", pw.conf.Type).WithProperty("profile", pw.conf.Profile).
		Notice("Profile watcher started.")
	defer ticker.Stop()

	task := Task{
		TaskType: ListSnapshots,
		Image: perun.Image{
			ImageType: pw.conf.Type,
			Profile:   pw.conf.Profile,
		},
	}

	for {
		select {
		case <-pw.grs.Stop:
			return
		case <-ticker.C:
			pw.taskConsumer.Run(task)
		}
	}
}
