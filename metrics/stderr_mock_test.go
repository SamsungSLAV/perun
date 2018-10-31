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
package metrics

import (
	"io/ioutil"
	"os"
)

func withStderrMocked(testFunction func()) string {
	r, w, _ := os.Pipe()

	tmp := os.Stderr
	defer func() {
		os.Stderr = tmp
	}()
	os.Stderr = w

	go func() {
		testFunction()
		w.Close()
	}()

	buffer, _ := ioutil.ReadAll(r)
	return string(buffer)
}
