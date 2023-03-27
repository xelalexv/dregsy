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
	"strconv"
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
var keepCount *util.Regex

//
func init() {
	var err error
	if keepCount, err = util.NewRegex(
		"keep:[[:space:]]+latest[[:space:]]+[[:digit:]]+"); err != nil {
		panic(fmt.Sprintf("invalid regex for keep latest: %v", err))
	}
}

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
	verbatim  []string
	semver    []semver.Range
	regex     []*util.Regex
	keep      []*util.Regex
	keepCount int
}

//
func (ts *TagSet) add(tags []string) error {

	for _, t := range tags {

		t = strings.TrimSpace(t)

		switch {

		case isSemver(t):
			if err := ts.addSemver(t); err != nil {
				return err
			}

		case isRegex(t):
			if err := ts.addRegex(t); err != nil {
				return err
			}

		case isKeepCount(t):
			if err := ts.setKeepCount(t); err != nil {
				return err
			}

		case isKeep(t):
			if err := ts.addKeep(t); err != nil {
				return err
			}

		default:
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
func (ts *TagSet) setKeepCount(c string) (err error) {
	if p := strings.Split(c, " "); len(p) > 1 {
		ts.keepCount, err = strconv.Atoi(p[len(p)-1])
	} else {
		err = fmt.Errorf("invalid keep count: %s", c)
	}
	return
}

//
func (ts *TagSet) addKeep(k string) (err error) {
	ts.keep, err = ts.addFilter(k, KeepPrefix, ts.keep)
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

	if ts.keepCount > 0 {
		log.WithField("limit", ts.keepCount).Debug("reducing tag set")
		ret = ts.reduce(ret, ts.keepCount)
	} else {
		sort.Strings(ret)
	}

	log.Debugf("expanded tags: %v", ret)

	return ret, nil
}

//
func (ts *TagSet) expandSemver(tags []string) []string {

	vers := make(versions, 0, len(tags))

	for _, t := range tags {
		if v, err := semver.ParseTolerant(t); err != nil {
			log.WithField("tag", t).Debugf(
				"skipping tag, not a valid semver: %v", err)
		} else {
			vers = append(vers, &version{Version: v, tag: t})
		}
	}

	var ret []string
	for _, v := range vers {
		for _, r := range ts.semver {
			if r(v.Version) {
				ret = append(ret, v.tag)
				break
			}
		}
	}

	log.Debugf("tags expanded from semver: %v", ret)
	return ret
}

//
func (ts *TagSet) reduce(tags []string, limit int) []string {

	vers := make(versions, 0, len(tags))

	// reorg tags list to contain all non-semver tags on the left, all semvers
	// on the right, which will then start at `pivot`
	pivot := 0
	for ix, t := range tags {
		if v, err := semver.ParseTolerant(t); err != nil {
			if ix != pivot {
				tags[pivot], tags[ix] = tags[ix], tags[pivot]
			}
			pivot++
		} else {
			vers = append(vers, &version{Version: v, tag: t})
		}
	}

	if len(vers) > 0 { // if there are semvers, we apply the limit only to those
		if end := pivot + limit; end < len(tags) {
			// there are more semvers than limit, need to reduce
			vers.sort() // descending, semver
			for ix, v := range vers {
				tags[pivot+ix] = v.tag
			}
			log.Debugf("removed tags: %v", tags[end:])
			tags = tags[:end]
		}
		sort.Strings(tags) // ascending, string

	} else { // otherwise the limit applies to the whole set
		sort.Strings(tags) // ascending, string
		if start := len(tags) - limit; start > 0 {
			log.Debugf("removed tags: %v", tags[:start])
			tags = tags[start:]
		}
	}

	return tags
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

//
func isKeepCount(tag string) bool {
	return keepCount.Matches(tag)
}
