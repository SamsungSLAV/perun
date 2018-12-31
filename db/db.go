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

// Package db contains implementation of Perun's database for storing permanent information.
// Database should be accessed only with DB interface.
package db

import (
	"github.com/SamsungSLAV/perun"
)

// DB interface defines API for accessing Perun's database.
type DB interface {
	// Start inilializes DB connection to be used by Perun.
	Start() error

	// Close finishes usage of DB connection.
	Close() error

	// GetRevision returns perun's database global revision.
	GetRevision() (int, error)

	// UpdateImage updates database with new information about image file.
	UpdateImage(*perun.ImageFileInfo) error
}
