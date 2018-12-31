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
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/SamsungSLAV/perun/db"
	"github.com/SamsungSLAV/perun/metrics"
	"github.com/SamsungSLAV/perun/watcher"
	"github.com/SamsungSLAV/slav/logger"
)

type server struct {
	conf     *configuration
	logLevel logger.Level
	modules  []module
}

func newServer() *server {
	return &server{
		conf:     &configuration{},
		logLevel: logger.ErrLevel,
		modules:  make([]module, 0),
	}
}

func (s *server) getConfiguration() {
	s.conf.get()
}

func (s *server) setLoggerLevel(change int) error {
	newLevel := logger.Level(change + int(s.logLevel))
	err := logger.SetThreshold(newLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot set new log level %d. Error: %s\n", newLevel, err.Error())
		return err
	}
	s.logLevel = newLevel
	return nil
}

func (s *server) setupLogger() error {
	var err error
	s.logLevel, err = logger.StringToLevel(s.conf.logLevelStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level %s. Error: %s\n", s.conf.logLevelStr,
			err.Error())
		return err
	}
	return s.setLoggerLevel(0)
}

func (s *server) setupModules() error {
	DB, err := db.NewDB(s.conf.dbPath)
	if err != nil {
		return err
	}
	s.modules = append(s.modules, DB)

	M, err := metrics.NewMetrics(s.conf.metricsListenAddress)
	if err != nil {
		return err
	}
	s.modules = append(s.modules, M)

	W, err := watcher.NewWatcher(s.conf.watcherConfFilePath, DB, M)
	if err != nil {
		return err
	}
	s.modules = append(s.modules, W)

	return nil
}

func (s *server) setupRLimit() error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		logger.WithError(err).Error("Error getting current Rlimit.")
		return err
	}
	logger.WithProperty("rlimit", rLimit).Info("Initial Rlimit.")

	rLimit.Cur = rLimit.Max

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		logger.WithError(err).Error("Error setting new Rlimit.")
		return err
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		logger.WithError(err).Error("Error getting new Rlimit.")
		return err
	}
	logger.WithProperty("rlimit", rLimit).Info("Modified Rlimit.")

	return nil
}

func (s *server) handleSignals() error {
	// Buffered channels used no to miss the signal if we're not ready yet to receive
	// when the signal is sent.
	cterm := make(chan os.Signal, 1)
	ccore := make(chan os.Signal, 1)
	cspec := make(chan os.Signal, 1)

	terminate := []os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
	}
	coredump := []os.Signal{
		syscall.SIGQUIT,
	}
	special := []os.Signal{
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	}

	signal.Notify(cterm, terminate...)
	signal.Notify(ccore, coredump...)
	signal.Notify(cspec, special...)

	for {
		select {
		case sig := <-cterm:
			logger.Noticef("Signal %s received. Terminating.", sig)
			return nil
		case sig := <-ccore:
			logger.Noticef("Signal %s received. Dumping core.", sig)
			runtime.Breakpoint()
			return nil
		case sig := <-cspec:
			logger.Noticef("Signal %s received. Changing log level.", sig)
			switch sig {
			case syscall.SIGUSR1:
				_ = s.setLoggerLevel(+1)
			case syscall.SIGUSR2:
				_ = s.setLoggerLevel(-1)
			}
		}
	}
}

func (s *server) run() error {
	err := s.setupRLimit()
	if err != nil {
		return err
	}
	for _, mod := range s.modules {
		err = mod.Start()
		if err != nil {
			return err
		}
		defer func(m module) {
			m.Close()
		}(mod)
	}

	return s.handleSignals()
}
