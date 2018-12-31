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

package db

import "errors"

var (
	// ErrSchemaMismatch is returned when trying to create a table in DB
	// with schema not matching existing table.
	ErrSchemaMismatch = errors.New("Table schema mismatch")
	// ErrDatabaseConnectionNotInitialized is returned when database module is misused.
	// This can happen when someone uses database without starting connection with Start()
	// method first.
	ErrDatabaseConnectionNotInitialized = errors.New("Database connection not initialized")
)
