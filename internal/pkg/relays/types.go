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

/*
	An image digest is formated like this:
	sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a
*/

package relays

import (
	"github.com/xelalexv/dregsy/internal/pkg/digests"
	"github.com/xelalexv/dregsy/internal/pkg/tags"
)

type SyncOptions struct {
	//
	SrcRef           string
	SrcAuth          string
	SrcSkipTLSVerify bool
	//
	TrgtRef           string
	TrgtAuth          string
	TrgtSkipTLSVerify bool
	// Skopeo Docker references with both a tag and digest are currently not supported
	Tags    *tags.TagSet
	Digests *digests.DigestList
	// skopeo copy --preserve-digests option
	// TODO change location of PreserveDigests ? see internal/pkg/sync/task.go
	PreserveDigests bool
	Platform        string
	Verbose         bool
}

type Support interface {
	Platform(p string) error
}
