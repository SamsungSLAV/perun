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

package main

import (
	"flag"

	"github.com/SamsungSLAV/slav/logger"
)

type configuration struct {
	logLevelStr          string
	dbPath               string
	metricsListenAddress string
	watcherConfFilePath  string
}

func (c *configuration) get() {
	flag.StringVar(&c.logLevelStr, "log", logger.ErrLevelStr, "level used as logger's threshold")
	flag.StringVar(&c.dbPath, "db", "perun.db3", "filepath of Perun's database")
	flag.StringVar(&c.metricsListenAddress, "metrics", ":7378", "listening address for metrics HTTP server")
	flag.StringVar(&c.watcherConfFilePath, "watcher", "watcher.conf", "filepath of watcher's module configuration file")

	flag.Parse()

}
