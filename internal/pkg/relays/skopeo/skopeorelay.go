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

package skopeo

import (
	"bytes"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays"
	"github.com/xelalexv/dregsy/internal/pkg/util"
)

const RelayID = "skopeo"

type RelayConfig struct {
	Binary   string `yaml:"binary"`
	CertsDir string `yaml:"certs-dir"`
}

type Support struct{}

func (s *Support) Platform(p string) error {
	return nil
}

type SkopeoRelay struct {
	wrOut io.Writer
}

func NewSkopeoRelay(conf *RelayConfig, out io.Writer) *SkopeoRelay {

	relay := &SkopeoRelay{}

	if out != nil {
		relay.wrOut = out
	}
	if conf != nil {
		if conf.Binary != "" {
			skopeoBinary = conf.Binary
		}
		if conf.CertsDir != "" {
			certsBaseDir = conf.CertsDir
		}
	}

	return relay
}

func (r *SkopeoRelay) Prepare() error {

	bufOut := new(bytes.Buffer)
	if err := runSkopeo(bufOut, nil, true, "--version"); err != nil {
		return fmt.Errorf("cannot execute skopeo: %v", err)
	}

	log.Info(bufOut.String())
	log.WithField("relay", RelayID).Info("relay ready")

	return nil
}

func (r *SkopeoRelay) Dispose() error {
	return nil
}

// Sync with support for digests and tags.
// Warning: Skopeo Docker references with both a tag and digest are
// currently not supported.
//
// The `digests` list & `tags` list are stored in a mapping struct.
//
// | `digests` list | `tags` list | `dregsy` behavior                             | diff with 0.4.4 |
// |----------------|-------------|-----------------------------------------------|-----------------|
// | empty          | empty       | pulls all tags                                | same            |
// | empty          | NOT empty   | pulls filtered tags only                      | same            |
// | NOT empty      | NOT empty   | pulls filtered tags AND pulls correct digests | different       |
// | NOT empty      | empty       | pulls correct digests only, ignores tags      | different       |
//
// A "correct digest" is a crrectly formated AND an existing digest.
// Skopeo is used to verify the digest exists.
func (r *SkopeoRelay) Sync(opt *relays.SyncOptions) error {

	srcCreds := util.DecodeJSONAuth(opt.SrcAuth)
	destCreds := util.DecodeJSONAuth(opt.TrgtAuth)

	cmd := []string{
		"--insecure-policy",
		"copy",
	}

	// Adding preserve digest option
	// $ skopeo copy --preserve-digests [...]
	if opt.PreserveDigests {
		cmd = append(cmd, "--preserve-digests=true")
		log.Debug("--preserve-digests=true")
	}

	if opt.SrcSkipTLSVerify {
		cmd = append(cmd, "--src-tls-verify=false")
	}
	if opt.TrgtSkipTLSVerify {
		cmd = append(cmd, "--dest-tls-verify=false")
	}

	srcCertDir := ""
	repo, _, _ := util.SplitRef(opt.SrcRef)
	if repo != "" {
		srcCertDir = CertsDirForRepo(repo)
		cmd = append(cmd, fmt.Sprintf("--src-cert-dir=%s", srcCertDir))
	}
	repo, _, _ = util.SplitRef(opt.TrgtRef)
	if repo != "" {
		cmd = append(cmd, fmt.Sprintf(
			"--dest-cert-dir=%s/%s", certsBaseDir, withoutPort(repo)))
	}

	if srcCreds != "" {
		cmd = append(cmd, fmt.Sprintf("--src-creds=%s", srcCreds))
	}
	if destCreds != "" {
		cmd = append(cmd, fmt.Sprintf("--dest-creds=%s", destCreds))
	}

	// Sync with support for digests and tags.
	if !opt.Digests.IsEmpty() && opt.Tags.IsEmpty() {
		log.Debug("Digest list is not empty but Tag list is empty: pulling image digest only - do not expand tags")
	} else {
		// Expand tags
		log.Debug("Tag list is not empty: expanding tags")
		tags, err := opt.Tags.Expand(func() ([]string, error) {
			return ListAllTags(opt.SrcRef, srcCreds, srcCertDir, opt.SrcSkipTLSVerify)
		})

		if err != nil {
			return fmt.Errorf("error expanding tags: %v", err)
		}

		errs := false

		// Syncing images based on the Tags
		for _, t := range tags {

			log.WithFields(
				log.Fields{"tag": t, "platform": opt.Platform}).Info("syncing tag")

			rc := append(cmd,
				fmt.Sprintf("docker://%s:%s", opt.SrcRef, t),
				fmt.Sprintf("docker://%s:%s", opt.TrgtRef, t))

			switch opt.Platform {
			case "":
			case "all":
				rc = append(rc, "--all")
			default:
				rc = addPlatformOverrides(rc, opt.Platform)
			}

			if err := runSkopeo(r.wrOut, r.wrOut, opt.Verbose, rc...); err != nil {
				log.Error(err)
				errs = true
			}
		}

		if errs {
			return fmt.Errorf("errors during sync with tags")
		}
	}

	// Syncing Images based on the Digests.
	// Warning: Skopeo Docker references with both a tag and digest are
	// currently not supported.
	//
	// Example of a skopeo copy command with digest:
	// $ skopeo copy --preserve-digests docker://docker.io/registry@sha256:cc6393207bf9d3e032c4d9277834c1695117532c9f7e8c64e7b7adcda3a85f39 docker-archive:./registry-linux-amd64-2.8.1.tar:docker.io/monproprechemin/registry:2.8.1
	for _, dig := range opt.Digests.Digests {
		// test if the image digest exists on the source registry
		ret, err := digestExist(
			dig,
			opt.SrcRef,
			srcCreds,
			srcCertDir,
			opt.SrcSkipTLSVerify)
		// case: image digest exist
		if ret == true && err == nil {
			log.WithFields(
				log.Fields{"digest": dig}).Info("syncing digest")

			rc := append(cmd,
				fmt.Sprintf("docker://%s@%s", opt.SrcRef, dig),
				fmt.Sprintf("docker://%s", opt.TrgtRef))
			log.Debug(rc)

			if e := runSkopeo(r.wrOut, r.wrOut, opt.Verbose, rc...); e != nil {
				log.Error(e)
				return fmt.Errorf("errors during sync with digests")
			}
		} else {
			log.Error(err)
		}
	}

	return nil
}
