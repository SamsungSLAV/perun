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

package watcher

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SamsungSLAV/perun"
	"github.com/SamsungSLAV/slav/logger"
	"golang.org/x/net/html"
)

// crawler sends a HTTP request (GET or HEAD) for a single URL and parses the HTML response.
// It can be used to collect:
// * <a> tags,
// * href attributes,
// * url properties,
// * file properties,
// from gotten page. The list can be filtered using ignoredPrefixes (black list)
// or requiredSuffixes (white list if not empty).
type crawler struct {
	url              string
	ignoredPrefixes  []string
	requiredSuffixes []string
}

// defaultIgnoredPrefixes are ignored in most of the cases.
var defaultIgnoredPrefixes = []string{"?", "/", "..", "latest"}

// defaultImageSuffixes defines the default image suffix.
var defaultImageSuffixes = []string{".tar.gz"}

// newCrawler creates a new crawler with given url and default settings.
func newCrawler(url string) *crawler {
	return &crawler{
		url:             url,
		ignoredPrefixes: defaultIgnoredPrefixes,
	}
}

// requireSuffixes allows modification of crawler's requiredSuffixes filter.
func (c *crawler) requireSuffixes(suffixes []string) *crawler {
	c.requiredSuffixes = suffixes
	return c
}

// getLinks extracts list of links urls from the HTML page received with GET method.
func (c *crawler) getLinks() (ret []string) {
	resp, err := http.Get(c.url)
	if err != nil {
		logger.WithError(err).WithProperty("url", c.url).Warning("Cannot GET requested url.")
		return
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.WithError(err).WithProperty("url", c.url).
				Warning("Cannot close GET response body.")
		}
	}()

	return c.parseATags(html.NewTokenizer(resp.Body))
}

// parseATags extracts links from HTML <a> tags.
func (c *crawler) parseATags(tok *html.Tokenizer) (ret []string) {
	for {
		token := tok.Next()

		switch token {
		case html.ErrorToken:
			err := tok.Err()
			if err != io.EOF {
				logger.WithError(err).WithProperty("url", c.url).
					Warning("Error during parsing HTML token.")
			}
			return
		case html.StartTagToken:
			name, attr := tok.TagName()
			if attr && len(name) == 1 && name[0] == 'a' {
				links := c.parseHrefAttr(tok)
				ret = append(ret, links...)
			}
		}
	}
}

// parseHrefAttr extracts href attributes from <a> tags.
func (c *crawler) parseHrefAttr(tok *html.Tokenizer) (ret []string) {
	for {
		key, val, more := tok.TagAttr()
		if string(key) != "href" {
			continue
		}
		value := strings.TrimSuffix(string(val), "/")
		if c.isMatching(value) {
			ret = append(ret, value)
		}
		if !more {
			break
		}
	}
	return
}

// isMatching verifies if found link matches crawler's filters (prefix and suffix).
func (c *crawler) isMatching(value string) bool {
	for _, prefix := range c.ignoredPrefixes {
		if strings.HasPrefix(value, prefix) {
			return false
		}
	}
	if len(c.requiredSuffixes) == 0 {
		return true
	}
	for _, suffix := range c.requiredSuffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

// getFileInfo extracts file information from HTTP header acquired with HEAD method.
func (c *crawler) getFileInfo() *perun.FileInfo {
	resp, err := http.Head(c.url)
	if err != nil {
		logger.WithError(err).WithProperty("url", c.url).Warning("Cannot HEAD requested url.")
		return nil
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.WithError(err).WithProperty("url", c.url).
				Warning("Cannot close HEAD response body.")
		}
	}()

	val, ok := resp.Header["Content-Length"]
	if !ok || len(val) < 1 {
		logger.WithProperty("url", c.url).Warning("No valid Content-Length value.")
		return nil
	}
	length, err := strconv.ParseInt(val[0], 10, 64)
	if err != nil {
		logger.WithError(err).WithProperty("url", c.url).WithProperty("value", val[0]).
			Warning("Parsing Content-Length value failed.")
		return nil
	}

	val, ok = resp.Header["Last-Modified"]
	if !ok || len(val) < 1 {
		logger.WithProperty("url", c.url).Warning("No valid Last-Modified value.")
		return nil
	}
	modified, err := time.Parse(time.RFC1123, val[0])
	if err != nil {
		logger.WithError(err).WithProperty("url", c.url).WithProperty("value", val[0]).
			Warning("Parsing Last-Modified value failed.")
		return nil
	}
	return &perun.FileInfo{
		Length:   length,
		Modified: modified,
	}
}
