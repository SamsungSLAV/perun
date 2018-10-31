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

package util

import "sync"

// GoRoutineSync is an aggregation of channel used for stoping goroutine
// and for wait group used by parent to wait until goroutine is done.
type GoRoutineSync struct {
	// Stop is a channel which closure indicates that goroutine should end.
	Stop <-chan interface{}
	// Done is a wait group that should be notified, when goroutine is done.
	Done *sync.WaitGroup
}
