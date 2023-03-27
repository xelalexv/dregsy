/*
	Copyright 2023 Alexander Vollschwitz <xelalex@gmx.net>

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
	"sort"

	"github.com/blang/semver/v4"
)

//
type versions []*version

//
func (s versions) Len() int {
	return len(s)
}

//
func (s versions) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

//
func (s versions) Less(i, j int) bool {
	return !s[i].LT(s[j].Version) // we want descending sort
}

//
func (s versions) sort() {
	sort.Sort(s)
}

//
type version struct {
	semver.Version
	tag string
}
