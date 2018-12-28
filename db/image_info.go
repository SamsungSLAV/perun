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

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/slav/logger"
)

// ImageInfo wraps binary image information perun.ImageFileInfo, allowing methods definition.
type ImageInfo perun.ImageFileInfo

// insert inserts a single row containing image info into images table.
func (info *ImageInfo) insert(tx *sql.Tx) error {
	_, err := tx.Exec(imagesInsert, info.URL, info.Image.Server,
		info.Image.ImageType, info.Image.Profile,
		info.Image.Snapshot, info.Image.Prerelease,
		info.Image.Repository, info.Image.ImageName, info.Image.FileName,
		info.Info.Length, info.Info.Modified.UTC().Unix())
	if err != nil {
		logger.WithError(err).WithProperty("sql", imagesInsert).
			Error("Failed to insert image.")
	}
	return err
}
