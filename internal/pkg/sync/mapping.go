/*
	Copyright 2020 Alexander Vollschwitz <xelalex@gmx.net>

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	  http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package sync

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xelalexv/dregsy/internal/pkg/tags"
	"github.com/xelalexv/dregsy/internal/pkg/util"
)

//
const RegexpPrefix = "regex:"

//
type Mapping struct {
	From     string   `yaml:"from"`
	To       string   `yaml:"to"`
	Tags     []string `yaml:"tags"`
	Platform string   `yaml:"platform"`
	//
	fromFilter *regexp.Regexp
	toFilter   *regexp.Regexp
	toReplace  string
	tagSet     *tags.TagSet
}

//
func (m *Mapping) validate() error {

	if m == nil {
		return fmt.Errorf("mapping is nil")
	}

	if m.From == "" {
		return fmt.Errorf("mapping without 'From' path")
	}

	if m.isRegexpFrom() {
		regex := m.From[len(RegexpPrefix):]
		var err error
		if m.fromFilter, err = util.CompileRegex(regex, true); err != nil {
			return fmt.Errorf(
				"'from' uses invalid regular expression '%s': %v", regex, err)
		}
	} else {
		m.From = normalizePath(m.From)
	}

	if m.isRegexpTo() {
		parts := strings.SplitN(m.To[len(RegexpPrefix):], ",", 2)
		regex := parts[0]
		if len(parts) < 2 {
			return fmt.Errorf("replacement expression missing in 'to'")
		}
		m.toReplace = parts[1]

		var err error
		if m.toFilter, err = util.CompileRegex(regex, false); err != nil {
			return fmt.Errorf(
				"'to' uses invalid regular expression '%s': %v", regex, err)
		}
	} else if m.To != "" {
		m.To = normalizePath(m.To)
	}

	if tags, err := tags.NewTagSet(m.Tags); err != nil {
		return fmt.Errorf("'tags' uses invalid format: %v", err)
	} else {
		m.tagSet = tags
	}

	return nil
}

//
func (m *Mapping) filterRepos(repos []string) []string {

	if m.isRegexpFrom() {
		ret := make([]string, 0, len(repos))
		for _, r := range repos {
			if m.fromFilter.MatchString(r) {
				ret = append(ret, normalizePath(r))
			}
		}
		return ret
	}

	return repos
}

//
func (m *Mapping) mapPath(p string) string {
	if m.isRegexpTo() {
		return m.toFilter.ReplaceAllString(p, m.toReplace)
	}
	if m.To != "" {
		if m.isRegexpFrom() {
			return m.To + p
		}
		return m.To
	}
	return p
}

//
func (m *Mapping) isRegexpFrom() bool {
	return isRegexp(m.From)
}

//
func (m *Mapping) isRegexpTo() bool {
	return isRegexp(m.To)
}

//
func isRegexp(expr string) bool {
	return strings.HasPrefix(expr, RegexpPrefix)
}

//
func normalizePath(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}
