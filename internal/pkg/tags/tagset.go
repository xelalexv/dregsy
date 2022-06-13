/*
	Copyright 2021 Alexander Vollschwitz <xelalex@gmx.net>

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

package tags

import (
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/util"
)

//
const SemverPrefix = "semver:"
const RegexpPrefix = "regex:"
const KeepPrefix = "keep:"

//
func NewTagSet(tags []string) (*TagSet, error) {
	ret := &TagSet{}
	if err := ret.add(tags); err != nil {
		return nil, err
	}
	return ret, nil
}

//
type TagSet struct {
	verbatim []string
	semver   []semver.Range
	regex    []*util.Regex
	keep     []*util.Regex
}

//
func (ts *TagSet) add(tags []string) error {
	for _, t := range tags {
		if isSemver(t) {
			if err := ts.addSemver(t); err != nil {
				return err
			}
		} else if isRegex(t) {
			if err := ts.addRegex(t); err != nil {
				return err
			}
		} else if isKeep(t) {
			if err := ts.addKeep(t); err != nil {
				return err
			}
		} else {
			ts.addVerbatim(t)
		}
	}
	return nil
}

//
func (ts *TagSet) addVerbatim(v string) {
	ts.verbatim = append(ts.verbatim, v)
}

//
func (ts *TagSet) addSemver(s string) error {
	if r, e := semver.ParseRange(s[len(SemverPrefix):]); e != nil {
		return e
	} else {
		ts.semver = append(ts.semver, r)
		return nil
	}
}

//
func (ts *TagSet) addRegex(r string) (err error) {
	ts.regex, err = ts.addFilter(r, RegexpPrefix, ts.regex)
	return
}

//
func (ts *TagSet) addKeep(r string) (err error) {
	ts.keep, err = ts.addFilter(r, KeepPrefix, ts.keep)
	return
}

//
func (ts *TagSet) addFilter(regex, prefix string, list []*util.Regex) (
	[]*util.Regex, error) {

	if reg, err := util.NewRegex(
		strings.TrimSpace(regex[len(prefix):])); err != nil {
		return nil, err
	} else {
		return append(list, reg), nil
	}
}

//
func (ts *TagSet) IsEmpty() bool {
	return !ts.HasVerbatim() && !ts.HasSemver() && !ts.HasRegex()
}

//
func (ts *TagSet) HasVerbatim() bool {
	return len(ts.verbatim) > 0
}

//
func (ts *TagSet) HasSemver() bool {
	return len(ts.semver) > 0
}

//
func (ts *TagSet) HasRegex() bool {
	return len(ts.regex) > 0
}

//
func (ts *TagSet) NeedsExpansion() bool {
	return ts.IsEmpty() || ts.HasSemver() || ts.HasRegex()
}

//
func (ts *TagSet) Expand(lister func() ([]string, error)) ([]string, error) {

	set := make(map[string]string)

	if ts.NeedsExpansion() {

		tags, err := lister()
		if err != nil {
			return nil, fmt.Errorf(
				"failed listing tags during tag set expansion: %v", err)
		}

		if !ts.HasSemver() && !ts.HasRegex() { // tag set is completely empty
			addToSet(set, tags)

		} else {
			if ts.HasSemver() {
				addToSet(set, ts.expandSemver(tags))
			}
			if ts.HasRegex() {
				addToSet(set, ts.expandRegex(tags))
			}
		}
	}

	if ts.HasVerbatim() {
		log.Debugf("verbatim tags: %v", ts.verbatim)
		addToSet(set, ts.verbatim)
	}

	ret := make([]string, 0, len(set))
	var pruned []string

	for t := range set {
		if ts.keepTag(t) {
			ret = append(ret, t)
		} else {
			if log.IsLevelEnabled(log.DebugLevel) {
				pruned = append(pruned, t)
			}
		}
	}

	log.Debugf("pruned tags: %v", pruned)

	sort.Strings(ret)
	log.Debugf("expanded tags: %v", ret)

	return ret, nil
}

//
func (ts *TagSet) expandSemver(tags []string) []string {

	var vers semver.Versions
	var used []string

	for _, t := range tags {
		if v, err := semver.ParseTolerant(t); err != nil {
			log.Debugf("skipping tag '%s', not a valid semver: %v", t, err)
		} else {
			vers = append(vers, v)
			used = append(used, t)
		}
	}

	var ret []string
	for ix, v := range vers {
		for _, r := range ts.semver {
			if r(v) {
				ret = append(ret, used[ix])
				break
			}
		}
	}

	log.Debugf("tags expanded from semver: %v", ret)
	return ret
}

//
func (ts *TagSet) expandRegex(tags []string) []string {

	var ret []string
	for _, t := range tags {
		for _, regex := range ts.regex {
			if regex.Matches(t) {
				ret = append(ret, t)
				break
			}
		}
	}

	log.Debugf("tags expanded from regex: %v", ret)
	return ret
}

//
func (ts *TagSet) keepTag(t string) bool {
	for _, regex := range ts.keep {
		if !regex.Matches(t) {
			return false
		}
	}
	return true
}

//
func addToSet(s map[string]string, tags []string) {
	for _, t := range tags {
		s[t] = t
	}
}

//
func isSemver(tag string) bool {
	return strings.HasPrefix(tag, SemverPrefix)
}

//
func isRegex(tag string) bool {
	return strings.HasPrefix(tag, RegexpPrefix)
}

//
func isKeep(tag string) bool {
	return strings.HasPrefix(tag, KeepPrefix)
}
