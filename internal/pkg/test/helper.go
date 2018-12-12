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

package test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

//
var StackTraceDepth = 1

//
type TestHelper struct {
	*testing.T
}

//
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t}
}

//
func (t *TestHelper) GetFixture(fx string) string {
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot determine fixture path")
	}
	p := filepath.Join(filepath.Dir(f), "../../../test/fixtures", fx)
	return p
}

//
func (t *TestHelper) AssertTrue(got bool) {
	t.AssertEqual(true, got)
}

//
func (t *TestHelper) AssertFalse(got bool) {
	t.AssertEqual(false, got)
}

//
func (t *TestHelper) AssertNil(i interface{}) {
	if i != nil && !reflect.ValueOf(i).IsZero() {
		t.raiseError("want nil, not \"%v\"", i)
	}
}

//
func (t *TestHelper) AssertNotNil(i interface{}) {
	if i == nil || reflect.ValueOf(i).IsZero() {
		t.raiseError("want non-nil, not nil")
	}
}

//
func (t *TestHelper) AssertError(e error, msg string) {
	if e == nil {
		t.raiseError("want error, but got none")
	} else if !strings.Contains(e.Error(), msg) {
		t.raiseError(
			"want error message to contain '%s', but got '%s'", msg, e.Error())
	}
}

//
func (t *TestHelper) AssertNoError(e error) {
	if e != nil {
		t.raiseError("don't want error: %v", e)
	}
}

//
func (t *TestHelper) AssertEqual(want, got interface{}) {
	if want != got {
		t.raiseError("want \"%v\", not \"%v\"", want, got)
	}
}

//
func (t *TestHelper) AssertNotEqual(want, got interface{}) {
	if want == got {
		t.raiseError("don't want \"%v\"", want)
	}
}

//
func (t *TestHelper) AssertQuiet(message string) {
	if message != "" {
		t.raiseError(message)
	}
}

//
func (t *TestHelper) AssertOneOf(want []string, got string) {
	for _, w := range want {
		if w == got {
			return
		}
	}
	t.raiseError("value \"%v\" is not in wanted set \"%v\"", got, want)
}

//
func (t *TestHelper) AssertEqualSlices(want, got []string) {

	e := len(want) != len(got)

	if !e {
		for ix := range want {
			if want[ix] != got[ix] {
				e = true
				break
			}
		}
	}

	if e {
		t.raiseError("want \"%v\", not \"%v\"", want, got)
	}
}

//
func (t *TestHelper) AssertEquivalentSlices(want, got []string) {

	e := len(want) != len(got)

	mWant := make(map[string]bool, len(want))

	if !e {
		for _, w := range want {
			mWant[w] = true
		}
		for _, g := range got {
			if !mWant[g] {
				e = true
				break
			}
		}
	}

	if e {
		t.raiseError("want \"%v\", not \"%v\"", want, got)
	}
}

//
func (t *TestHelper) AssertEqualMaps(want, got map[string]string) {
	if len(want) != len(got) {
		t.raiseError("maps of different size: want %d, not %d",
			len(want), len(got))
	}
	for k, v := range want {
		val, ok := got[k]
		if !ok {
			t.raiseError("expected map to contain key '%s'", k)
		}
		if val != v {
			t.raiseError("key '%s' mapped to wrong value: want '%s', not '%s'",
				k, v, val)
		}
	}
}

//
func (t *TestHelper) raiseError(format string, args ...interface{}) {
	stack := ""
	for _, s := range caller(StackTraceDepth) {
		stack = fmt.Sprintf("%s%s", stack, s)
	}
	t.Errorf("%s\n%s\n\n", stack, fmt.Sprintf(format, args...))
}

//
func caller(depth int) []string {

	// check where we are
	fpcs := make([]uintptr, 1)
	n := runtime.Callers(2, fpcs)

	if n == 0 {
		return []string{"n/a"}
	}

	pc := fpcs[0]
	thisFile, thisLine := runtime.FuncForPC(pc).FileLine(pc)

	// calculate number of required backspaces to remove `helper.go` prompt
	_, thisFileComp := filepath.Split(thisFile)
	back := len(thisFileComp) + len(strconv.Itoa(thisLine)) + 3

	// collect stack, starting at first file in call stack that's not this file,
	// and containing depth number of levels

	var file string
	var line int

	var ret []string
	collect := false
	ok := true
	skip := 0

	for {
		pc, file, line, ok = runtime.Caller(skip)
		if !ok {
			return []string{"n/a"}
		}
		if file != thisFile {
			collect = true
		}
		if collect {
			fun := strings.Split(runtime.FuncForPC(pc).Name(), "/")
			ret = append(ret, fmt.Sprintf("%s%s:%d  ::  %s()\n",
				strings.Repeat("\b", back), file, line, fun[len(fun)-1]))
			if len(ret) == depth {
				break
			}
			back = 4 // one tab stop from now on
		}
		skip++
	}

	return ret
}
