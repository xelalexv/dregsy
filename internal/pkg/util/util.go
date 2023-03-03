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

package util

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

//
const DigestPrefix = "sha256:"

const FormatDigest = "%s@%s"
const FormatName = "%s:%s"

//
func SplitRef(ref string) (reg, repo, tag string) {

	ix := strings.Index(ref, "/")

	if ix == -1 {
		reg = ""
		repo = ref
	} else {
		reg = ref[:ix]
		repo = ref[ix+1:]
	}

	// note: if ref contains a colon for specifying registry port, it is left of
	//       the first slash, and hence no longer included in repo at this point
	ixC := strings.Index(repo, ":")
	ixA := strings.Index(repo, "@")

	if ixC > -1 && ixA > -1 { // we have both : and @
		if ixC < ixA {
			ix = ixC
		} else {
			ix = ixA
		}
	} else if ixA > -1 { // only @ (actually invalid)
		ix = ixA
	} else if ixC > -1 { // only :
		ix = ixC
	} else {
		return // no tag
	}

	tag = repo[ix+1:]
	repo = repo[:ix]

	return
}

// HasName returns true if tag HAS or IS a name
func HasName(tag string) bool {
	n, _ := SplitTag(tag)
	return n != ""
}

// HasDigest returns true if tag HAS or IS a digest
func HasDigest(tag string) bool {
	_, d := SplitTag(tag)
	return d != ""
}

// IsDigest returns true if d is a digest string. For performance reasons, this
// currently only checks for the presence of the `sha256:` prefix, and does not
// run a regex against d for full compliance check. This way, incorrect digests
// lead to errors, but correct digests do not incur a computational overhead.
func IsDigest(d string) bool {
	return strings.HasPrefix(d, DigestPrefix)
}

//
func SplitTag(tag string) (name, digest string) {

	if strings.HasPrefix(tag, ":") {
		tag = tag[1:]
	}
	if tag == "" {
		return
	}

	if p := strings.Split(tag, "@"); len(p) == 1 { // either tag or digest
		if IsDigest(p[0]) {
			digest = p[0]
		} else {
			name = p[0]
		}
	} else { // both
		name = p[0]
		digest = p[1]
	}

	return
}

//
func JoinTag(name, digest string) string {
	if digest == "" {
		return name
	}
	if name == "" {
		return digest
	}
	return fmt.Sprintf(FormatDigest, name, digest)
}

//
func SplitPlatform(p string) (os, arch, variant string) {

	ix := strings.Index(p, "/")

	if ix == -1 {
		os = p
		arch = ""
	} else {
		os = p[:ix]
		arch = p[ix+1:]
	}

	ix = strings.Index(arch, "/")

	if ix > -1 {
		variant = arch[ix+1:]
		arch = arch[:ix]
	}

	return
}

// JoinRefAndTag joins ref with tag, inserting the correct separator ':' or '@',
// depending on whether tag contains a name part or is purely a digest.
func JoinRefAndTag(ref, tag string) string {
	if HasName(tag) {
		return fmt.Sprintf(FormatName, ref, tag)
	}
	return fmt.Sprintf(FormatDigest, ref, tag)
}

// JoinRefsAndTag joins the source and target ref for a sync action each with
// tag, according to these rules:
//
// - If tag contains a digest, srcRef is joined with only the digest as tag, and
//   trgtRef with either only the name part of tag (if present), or the digest.
//
// - Otherwise, srcRef and trgtRef are joined with tag.
//
// This ensures that if a digest is present, we always use that when pulling an
// image, but still use the name if present when pushing.
//
func JoinRefsAndTag(srcRef, trgtRef, tag string) (src, trgt string) {
	if name, digest := SplitTag(tag); digest != "" {
		src = fmt.Sprintf(FormatDigest, srcRef, digest)
		if name != "" {
			trgt = fmt.Sprintf(FormatName, trgtRef, name)
		} else {
			trgt = fmt.Sprintf(FormatDigest, trgtRef, digest)
		}
	} else {
		src = fmt.Sprintf(FormatName, srcRef, name)
		trgt = fmt.Sprintf(FormatName, trgtRef, name)
	}
	return
}

//
type creds struct {
	Username string
	Password string
}

//
func DecodeJSONAuth(authBase64 string) string {

	if authBase64 == "" {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(authBase64)
	if err != nil {
		log.Error(err)
		return ""
	}

	var ret creds
	if err := json.Unmarshal([]byte(decoded), &ret); err != nil {
		log.Error(err)
		return ""
	}

	return fmt.Sprintf("%s:%s", ret.Username, ret.Password)
}

//
func ComputeSHA1(file string) ([]byte, error) {

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	hash := sha1.New()
	if _, err = io.Copy(hash, f); err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

//
func CompareSHA1(a, b []byte) bool {

	if len(a) != len(b) {
		return false
	}

	for ix, va := range a {
		if va != b[ix] {
			return false
		}
	}
	return true
}
