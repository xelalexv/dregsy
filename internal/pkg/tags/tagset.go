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
	"regexp"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/util"

	"github.com/robertkrimen/otto"
)

//
const SemverPrefix = "semver:"
const RegexpPrefix = "regex:"
const JsPrefix = "js:"

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
	regex    []*regex
	js       []*js
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
		} else if isJs(t) {
			if err := ts.addJs(t); err != nil {
				return err
			}
		} else {
			if err := ts.addVerbatim(t); err != nil {
				return err
			}
		}
	}
	return nil
}

//
func (ts *TagSet) addVerbatim(v string) error {
	ts.verbatim = append(ts.verbatim, v)
	return nil
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
func (ts *TagSet) addRegex(r string) error {
	reg, err := newRegex(strings.TrimSpace(r[len(RegexpPrefix):]))
	if err != nil {
		return err
	}
	ts.regex = append(ts.regex, reg)
	return nil
}

//
func (ts *TagSet) addJs(r string) error {
	jsInst, err := newJs(strings.TrimSpace(r[len(JsPrefix):]))
	if err != nil {
		return err
	}
	ts.js = append(ts.js, jsInst)
	return nil
}

//
func (ts *TagSet) IsEmpty() bool {
	return !ts.HasVerbatim() && !ts.HasSemver() && !ts.HasRegex() && !ts.HasJs()
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
func (ts *TagSet) HasJs() bool {
	return len(ts.js) > 0
}

//
func (ts *TagSet) NeedsExpansion() bool {
	return ts.IsEmpty() || ts.HasSemver() || ts.HasRegex() || ts.HasJs()
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

		if !ts.HasSemver() && !ts.HasRegex() && !ts.HasJs() { // tag set is completely empty
			addToSet(set, tags)

		} else {
			if ts.HasSemver() {
				addToSet(set, ts.expandSemver(tags))
			}
			if ts.HasRegex() {
				addToSet(set, ts.expandRegex(tags))
			}
			if ts.HasJs() {
				addToSet(set, ts.expandJs(tags))
			}
		}
	}

	if ts.HasVerbatim() {
		log.Debugf("verbatim tags: %v", ts.verbatim)
		addToSet(set, ts.verbatim)
	}

	ret := make([]string, 0, len(set))
	for t := range set {
		ret = append(ret, t)
	}

	sort.Strings(ret)
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
			if regex.matches(t) {
				ret = append(ret, t)
				break
			}
		}
	}

	log.Debugf("tags expanded from regex: %v", ret)
	return ret
}

//
func (ts *TagSet) expandJs(tags []string) []string {

	var ret []string
	for _, t := range tags {
		for _, jsInst := range ts.js {
			if jsInst.matches(t) {
				ret = append(ret, t)
				break
			}
		}
	}

	log.Debugf("tags expanded from js: %v", ret)
	return ret
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
func isJs(tag string) bool {
	return strings.HasPrefix(tag, JsPrefix)
}

//
func newRegex(r string) (*regex, error) {

	r = strings.TrimSpace(r)
	inverted := strings.HasPrefix(r, "!")
	if inverted {
		r = r[1:]
	}

	reg, err := util.CompileRegex(r, true)
	if err != nil {
		return nil, err
	}

	return &regex{expr: reg, inverted: inverted}, nil
}

//
type regex struct {
	expr     *regexp.Regexp
	inverted bool
}

//
func (r *regex) matches(s string) bool {
	if r.inverted {
		return !r.expr.MatchString(s)
	}
	return r.expr.MatchString(s)
}

//
func newJs(r string) (*js, error) {

	r = strings.TrimSpace(r)
	return &js{
		vm:   otto.New(),
		jsfn: fmt.Sprintf(`(function(v){%s})(v)`, r),
	}, nil
}

type js struct {
	vm   *otto.Otto
	jsfn string
}

//
func (r *js) matches(s string) bool {
	err := r.vm.Set("v", s)
	if err != nil {
		log.Debugf("skipping tag '%s', js prepare error: %v", s, err)
		return false
	}

	v, err := r.vm.Eval(r.jsfn)
	if err != nil {
		log.Debugf("skipping tag '%s', js call error: %v", s, err)
		return false
	}
	if v.IsBoolean() {
		rv, _ := v.ToBoolean()
		return rv
	}

	if v.IsNumber() {
		rv, _ := v.ToInteger()
		return rv != 0
	}
	log.Debugf("skipping tag '%s', js returns unsupported result: %v", s, v)

	return false
}
