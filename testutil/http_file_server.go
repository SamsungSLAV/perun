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

package testutil

import (
	"fmt"
	"net/http"
	"sync"
)

// HTTPFileServer describes a simple http file server.
type HTTPFileServer struct {
	port       int
	dir        string
	serveMux   *http.ServeMux
	server     *http.Server
	serverDone *sync.WaitGroup
	mutex      sync.Locker
	started    bool
	failed     bool
}

// NewHTTPFileServer creates a new instance of initialized server.
func NewHTTPFileServer(port int, dir string) *HTTPFileServer {
	mux := http.NewServeMux()
	return &HTTPFileServer{
		port:     port,
		dir:      dir,
		serveMux: mux,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		serverDone: new(sync.WaitGroup),
		mutex:      new(sync.Mutex),
		started:    false,
		failed:     false,
	}
}

// Start runs server in an inner goroutine.
func (s *HTTPFileServer) Start() {
	fs := http.FileServer(http.Dir(s.dir))
	s.serveMux.Handle("/", http.StripPrefix("/", fs))

	s.serverDone.Add(1)
	go func() {
		defer s.serverDone.Done()
		s.mutex.Lock()
		s.started = true
		s.mutex.Unlock()
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.mutex.Lock()
			s.failed = true
			s.mutex.Unlock()
		}
	}()
}

// Stop breaks execution of server and waits until internal goroutine is finished.
func (s *HTTPFileServer) Stop() {
	err := s.server.Close()
	if err != nil {
		s.mutex.Lock()
		s.failed = true
		s.mutex.Unlock()
	}
	s.serverDone.Wait()

}

// IsStarted returns started server flag. It is set to true when internal goroutine is started.
func (s *HTTPFileServer) IsStarted() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.started
}

// IsFailed returns failed server flag. It is set in case of server setup or closure error.
func (s *HTTPFileServer) IsFailed() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.failed
}
