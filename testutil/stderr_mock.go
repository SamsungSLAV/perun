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
	"io/ioutil"
	"os"
)

// WithStderrMocked runs given testFunction with substituted stderr.
// All content written to stderr by the function is captured and returned as a string.
func WithStderrMocked(testFunction func()) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	tmp := os.Stderr
	defer func() {
		os.Stderr = tmp
	}()
	os.Stderr = w

	go func() {
		defer w.Close() // nolint: errcheck
		testFunction()
	}()

	buffer, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}
