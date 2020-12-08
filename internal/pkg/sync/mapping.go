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
	"errors"
)

//
type Mapping struct {
	From string   `yaml:"from"`
	To   string   `yaml:"to"`
	Tags []string `yaml:"tags"`
}

//
func (m *Mapping) validate() error {

	if m == nil {
		return errors.New("mapping is nil")
	}

	if m.From == "" {
		return errors.New("mapping without 'From' path")
	}

	if m.To == "" {
		m.To = m.From
	}

	return nil
}
