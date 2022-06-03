/*
	Copyright 2022 Alexander Vollschwitz <xelalex@gmx.net>

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

package util

import (
	"fmt"
	"regexp"
	"strings"
)

//
func CompileRegex(v string, lineMatch bool) (*regexp.Regexp, error) {
	if lineMatch {
		if !strings.HasPrefix(v, "^") {
			v = fmt.Sprintf("^%s", v)
		}
		if !strings.HasSuffix(v, "$") {
			v = fmt.Sprintf("%s$", v)
		}
	}
	return regexp.Compile(v)
}

//
func NewRegex(r string) (*Regex, error) {

	r = strings.TrimSpace(r)
	inverted := strings.HasPrefix(r, "!")
	if inverted {
		r = r[1:]
	}

	reg, err := CompileRegex(r, true)
	if err != nil {
		return nil, err
	}

	return &Regex{expr: reg, inverted: inverted}, nil
}

//
type Regex struct {
	expr     *regexp.Regexp
	inverted bool
}

//
func (r *Regex) Matches(s string) bool {
	if r.inverted {
		return !r.expr.MatchString(s)
	}
	return r.expr.MatchString(s)
}
