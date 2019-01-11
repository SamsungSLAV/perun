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

const (
	masterSelect = `SELECT sql FROM sqlite_master WHERE type = $1 AND name = $2`

	metaCreate = `CREATE TABLE meta (
key TEXT PRIMARY KEY ASC NOT NULL UNIQUE,
value INTEGER)`

	metaRevisionInit = `INSERT OR IGNORE INTO meta VALUES ('revision', 0)`

	metaRevisionGet = `SELECT value FROM meta WHERE key = 'revision'`

	imagesCreate = `CREATE TABLE images (
url TEXT PRIMARY KEY ASC NOT NULL UNIQUE,
server TEXT,
imagetype TEXT,
profile TEXT,
snapshot TEXT,
prerelease TEXT,
repository TEXT,
imagename TEXT,
filename TEXT,
length INTEGER,
modified INTEGER,
revision INTEGER)`

	imagesFilterCreate = `CREATE TRIGGER imagesFilter
BEFORE INSERT ON images
WHEN NEW.modified <= (
	SELECT modified FROM images WHERE url = NEW.url
	UNION
	SELECT 0
	ORDER BY modified DESC LIMIT 1
) BEGIN
	SELECT RAISE(IGNORE) ;
END`

	imagesRevisionCreate = `CREATE TRIGGER imagesRevision
AFTER INSERT ON images
BEGIN
	UPDATE images
		SET revision = (SELECT value + 1 FROM meta WHERE key = 'revision')
		WHERE url = NEW.url ;
	UPDATE meta
		SET value = (SELECT value + 1 FROM meta WHERE key = 'revision')
		WHERE key = 'revision' ;
END`

	imagesInsert = `INSERT OR REPLACE INTO images
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, -1)`

	imageSelect = `SELECT url, server, imagetype, profile, snapshot, prerelease,
repository, imagename, filename, length, modified
FROM images
WHERE revision > $1
AND server GLOB $2
AND imagetype GLOB $3
AND profile GLOB $4
AND snapshot GLOB $5
AND prerelease GLOB $6
AND repository GLOB $7
AND imagename GLOB $8
AND filename GLOB $9`
)
