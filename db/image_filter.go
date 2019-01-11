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

import (
	"database/sql"
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/slav/logger"
)

type scanOneFunc func(*sql.Rows) (*perun.ImageFileInfo, error)
type scanAllFunc func(*sql.Rows, scanOneFunc) ([]perun.ImageFileInfo, error)

// ImageFilter describes filtered image query parameters and response placeholder.
type ImageFilter struct {
	// Request fields.
	image *perun.Image
	since int
	// Response field.
	result []perun.ImageFileInfo
	// Scan functions isolated for test purposes.
	scanOne scanOneFunc
	scanAll scanAllFunc
}

// newImageFilter creates and initializes new ImageFilter structure.
func newImageFilter(filter *perun.Image, since int) (*ImageFilter, error) {
	if filter == nil {
		err := ErrNilPointer
		logger.WithError(err).Critical("Nil filter passed to newImageFilter function.")
		return nil, err
	}
	return &ImageFilter{
		image:   filter,
		since:   since,
		scanOne: scanOne,
		scanAll: scanAll,
	}, nil
}

// get queries database about matching images.
func (filter *ImageFilter) get(tx *sql.Tx) error {
	if filter == nil || filter.image == nil {
		err := ErrNilPointer
		logger.WithError(err).Critical("ImageFilter not properly initialized.")
		return err
	}
	rows, err := tx.Query(imageSelect,
		filter.since, filter.image.Server, filter.image.ImageType,
		filter.image.Profile, filter.image.Snapshot, filter.image.Prerelease,
		filter.image.Repository, filter.image.ImageName, filter.image.FileName)
	if err != nil {
		logger.WithError(err).WithProperty("sql", imageSelect).
			Error("Failed to select images.")
		return err
	}
	defer func() {
		// There is nothing to be done if rows closing fail.
		// The rows object usage has already been completed.
		// If any errors appeared earlier, they are already logged and handled.
		_ = rows.Close() //nolint:gosec
	}()
	result, err := filter.scanAll(rows, filter.scanOne)
	if err == nil {
		filter.result = result
	}
	return err
}

// scanOne reads a single image information from sql rows current cursor.
func scanOne(rows *sql.Rows) (*perun.ImageFileInfo, error) {
	var t int64
	var ifi perun.ImageFileInfo
	err := rows.Scan(&ifi.URL, &ifi.Image.Server, &ifi.Image.ImageType,
		&ifi.Image.Profile, &ifi.Image.Snapshot, &ifi.Image.Prerelease,
		&ifi.Image.Repository, &ifi.Image.ImageName, &ifi.Image.FileName,
		&ifi.Info.Length, &t)
	if err != nil {
		logger.WithError(err).WithProperty("sql", imageSelect).
			Error("Failed to scan row while selecting images.")
		return nil, err
	}
	ifi.Info.Modified = time.Unix(t, 0).UTC()
	return &ifi, nil
}

// scanAll iterates on image query result rows and read them all.
func scanAll(rows *sql.Rows, scan scanOneFunc) ([]perun.ImageFileInfo, error) {
	var result []perun.ImageFileInfo
	for rows.Next() {
		ifi, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *ifi)
	}
	err := rows.Err()
	if err != nil {
		logger.WithError(err).WithProperty("sql", imageSelect).
			Error("Failed to iterate rows while selecting images.")
		return nil, err
	}
	return result, nil
}
