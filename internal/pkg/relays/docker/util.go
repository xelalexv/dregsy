/*
	Copyright 2025 Alexander Vollschwitz <xelalex@gmx.net>

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

package docker

import (
	"strings"

	spec "github.com/opencontainers/image-spec/specs-go/v1"
)

//-
func toPlatform(p string) *spec.Platform {

	if p == "all" {
		return nil
	}

	var ret *spec.Platform

	if parts := strings.Split(p, "/"); len(parts) > 0 && parts[0] != "" {
		ret = &spec.Platform{OS: parts[0]}
		if len(parts) > 1 && parts[1] != "" {
			ret.Architecture = parts[1]
			if len(parts) > 2 && parts[2] != "" {
				ret.Variant = parts[2]
			}
		}
	}

	return ret
}
