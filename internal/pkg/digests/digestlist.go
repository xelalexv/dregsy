/*
	Copyright 2023

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
	How to get an image digest with skopeo inspect:
		$ skopeo inspect --format "{{.Digest}}" docker://docker.io/alpine:3.17.1
		sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a

	/!\ Skopeo does not support using tags and digest at the same time.
	This is valid (tag only):
		$ skopeo inspect --format "{{.Digest}}" docker://docker.io/alpine:3.17.1
		sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a

	This is valid (digest only):
		$ skopeo inspect --format "{{.Digest}}" docker://docker.io/alpine@sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a
		sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a

	This is not valid (tag and digest):
		$ skopeo inspect --format "{{.Digest}}" docker://docker.io/alpine:3.17.1@sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a
		FATA[0000] Error parsing image name "docker://docker.io/alpine:3.17.1@sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a": Docker references with both a tag and digest are currently not supported

*/

package digests

import (
	"errors"
	"regexp"

	log "github.com/sirupsen/logrus"
)

// An image digest has the following structure
// sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a
type DigestList struct {
	Digests []string
}

func NewDigestList(digests []string) (*DigestList, error) {
	ret := &DigestList{}
	if err := ret.addDigests(digests); err != nil {
		return nil, err
	}
	return ret, nil
}

// Adds one digest to the list of digests
func (ds *DigestList) addOneDigest(dig string) error {
	if isDigest(dig) {
		log.Debugf("adding digest: %v", dig)
		ds.Digests = append(ds.Digests, dig)
		return nil
	} else {
		err := errors.New("Error adding digest: bad format - only lowercase is supported")
		log.Debugf("wrong digest format: %v", dig)
		return err
	}
}

// Add a list of digests (string array)
func (ds *DigestList) addDigests(digests []string) error {
	for _, dig := range digests {
		if err := ds.addOneDigest(dig); err != nil {
			// this means this digest is not well formated
			log.Errorf("error bad formated digest: %s", dig)
			//return err
		}
	}
	return nil
}

// Return true if list of digest is empty.
func (ds *DigestList) IsEmpty() bool {
	if ds.Digests == nil {
		return true
	} else {
		return false
	}
}

// This function verifies if the image digest string is properly formated.
// An image digest has the following structure:
// sha256:f271e74b17ced29b915d351685fd4644785c6d1559dd1f2d4189a5e851ef753a
// Does not accept UPPERCASE in the digest, only lowercase.
func isDigest(digest string) bool {
	var re = regexp.MustCompile(`(?m)^sha256:[a-f0-9]{64}$`)
	return re.MatchString(digest)
}
