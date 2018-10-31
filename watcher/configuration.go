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
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/slav/logger"
)

// Period covers time.Duration type allowing method definition.
type Period time.Duration

// UnmarshalJSON parses Period from byte array.
func (period *Period) UnmarshalJSON(b []byte) error {
	str, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*period = Period(d)
	return nil
}

// ProfileConfiguration defines configuration entity for a single profile.
type ProfileConfiguration struct {
	Type    perun.ImageType `json:"type"`
	Profile perun.Profile   `json:"profile"`
	Period  Period          `json:"period"`
}

// Configuration defines configuration of watcher.
type Configuration struct {
	Profiles   []ProfileConfiguration `json:"profiles"`
	TasksQueue int                    `json:"tasksqueue"`
	Workers    int                    `json:"workers"`
}

// loadConfiguration loads watcher's configuration from json file.
func loadConfiguration(path string) (*Configuration, error) {
	file, err := os.Open(path) // nolint: gosec
	if err != nil {
		logger.WithError(err).WithProperty("file", path).
			Critical("Error loading watcher's configuration file.")
		return nil, err
	}
	defer closeFile(path, file)

	decoder := json.NewDecoder(file)
	conf := new(Configuration)
	err = decoder.Decode(conf)
	if err != nil {
		logger.WithError(err).WithProperty("file", path).
			Critical("Error parsing watcher's configuration file.")
		return nil, err
	}

	logger.WithProperty("file", path).Notice("Watcher's configuration file loaded.")
	return conf, nil
}

// closeFile closes configuration file and writes log in case of error.
func closeFile(path string, file *os.File) {
	err := file.Close()
	if err != nil {
		logger.WithError(err).WithProperty("file", path).
			Critical("Error closing watcher's configuration file.")
	}
}
