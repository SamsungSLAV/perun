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

	// GetImages returns collection of information about images maching given filter
	// (1st parameter) that have changed since given revision (2nd parameter).
	// The function also returns current revision, so further calls can use it to ask
	// only for further updates of image list.
	// In case of error function returns an empty collection and -1 revision and an error.
	//
	// Filter must be a non-nil pointer to perun.Image structure.
	// It's fields should be valid glob strings. Image in database passes filter
	// (and is returned) if all fields match defined glob expressions.
	//
	// Following wildcards can be used to construct glob expressions:
	// +----------+-----------------------+---------+----------------------+--------------------+
	// | Wildcard | Description           | Example | Matches              | Does not match     |
	// +----------+-----------------------+---------+----------------------+--------------------+
	// | *        | matches any number    | Law*    | Law, Laws, Lawyer    | GrokLaw, La, or aw |
	// |          | of any characters     +- - - - -+- - - - - - - - - - - + - - - - - - - - - -+
	// |          | including none        | *Law*   | Law, GrokLaw, Lawyer | La, or aw          |
	// +----------+-----------------------+---------+----------------------+--------------------+
	// | ?        | matches any single    | ?at     | Cat, cat, Bat, bat   | at                 |
	// |          | character             |         |                      |                    |
	// |          |                       |         |                      |                    |
	// |          |                       |         |                      |                    |
	// +----------+-----------------------+---------+----------------------+--------------------+
	// | [abc]    | matches one character | [CB]at  | Cat, Bat             | cat or bat         |
	// |          | given in the bracket  |         |                      |                    |
	// +----------+-----------------------+---------+----------------------+--------------------+
	// | [a-z]    | matches one character | L[0-9]  | L0, L1, L2 up to L9  | Ls, L or L10       |
	// |          | from the              |         |                      |                    |
	// |          | (locale-dependent)    |         |                      |                    |
	// |          | range given           |         |                      |                    |
	// |          | in the bracket        |         |                      |                    |
	// +----------+-----------------------+---------+----------------------+--------------------+
	// source: https://en.wikipedia.org/wiki/Glob_(programming)
	//
	GetImages(*perun.Image, int) ([]perun.ImageFileInfo, int, error)
}
