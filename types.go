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

package perun

import (
	"time"
)

// ImageType defines type of binary image, e.g. snapshot or prerelease.
type ImageType = string

const (
	// SNAPSHOT is a release snapshot image type.
	SNAPSHOT ImageType = "snapshot"
	// PRERELEASE is an image type related with submit request.
	PRERELEASE ImageType = "prerelease"
)

// Profile defines a profile name, e.g. common-3.0 or unified.
type Profile = string

// Snapshot defines a snapshot version, e.g. tizen-unified_20181024.1
type Snapshot = string

// Prerelease defines a prerelease project version, e.g. tizen-unified_20181024.1.20181031.134751
type Prerelease = string

// Repository defines an OBS repository, e.g. standard or emulator
type Repository = string

// ImageName defines a name of the image, e.g. iot-headless-2parts-armv7l-artik530_710
type ImageName = string

// FileName defines a name of the image file,
// e.g. tizen-unified_20181024.1.20181031.134751_iot-headless-2parts-armv7l-artik530_710.tar.gz
type FileName = string

// Image aggregates all information about the image.
type Image struct {
	ImageType
	Profile
	Snapshot
	Prerelease
	Repository
	ImageName
	FileName
}

// FileInfo contains basic file information.
type FileInfo struct {
	Length   int64
	Modified time.Time
}

// ImageFileInfo contains complete information about image file with URL location.
type ImageFileInfo struct {
	// URL location of the image file.
	URL string
	// Image information.
	Image Image
	// Image file details.
	Info FileInfo
}
