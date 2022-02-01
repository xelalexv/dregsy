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

	"github.com/xelalexv/dregsy/internal/pkg/tags"
	"github.com/xelalexv/dregsy/internal/pkg/util"
)

const RelayID = "skopeo"

//
type RelayConfig struct {
	Binary   string `yaml:"binary"`
	CertsDir string `yaml:"certs-dir"`
}

//
type SkopeoRelay struct {
	wrOut io.Writer
}

//
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

//
func (r *SkopeoRelay) Prepare() error {
	bufOut := new(bytes.Buffer)
	if err := runSkopeo(bufOut, nil, true, "--version"); err != nil {
		return fmt.Errorf("cannot execute skopeo: %v", err)
	}
	log.Info(bufOut.String())
	log.WithField("relay", RelayID).Info("relay ready")
	return nil
}

//
func (r *SkopeoRelay) Dispose() error {
	return nil
}

//
func (r *SkopeoRelay) Sync(srcRef, srcAuth string, srcSkipTLSVerify bool,
	destRef, destAuth string, destSkipTLSVerify bool, ts *tags.TagSet,
	verbose bool) error {

	srcCreds := util.DecodeJSONAuth(srcAuth)
	destCreds := util.DecodeJSONAuth(destAuth)

	cmd := []string{
		"--insecure-policy",
		"--all",
		"copy",
	}

	if srcSkipTLSVerify {
		cmd = append(cmd, "--src-tls-verify=false")
	}
	if destSkipTLSVerify {
		cmd = append(cmd, "--dest-tls-verify=false")
	}

	srcCertDir := ""
	repo, _, _ := util.SplitRef(srcRef)
	if repo != "" {
		srcCertDir = CertsDirForRepo(repo)
		cmd = append(cmd, fmt.Sprintf("--src-cert-dir=%s", srcCertDir))
	}
	repo, _, _ = util.SplitRef(destRef)
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

	tags, err := ts.Expand(func() ([]string, error) {
		return ListAllTags(srcRef, srcCreds, srcCertDir, srcSkipTLSVerify)
	})

	if err != nil {
		return fmt.Errorf("error expanding tags: %v", err)
	}

	errs := false
	for _, tag := range tags {
		log.WithField("tag", tag).Info("syncing tag")
		if err := runSkopeo(r.wrOut, r.wrOut, verbose, append(cmd,
			fmt.Sprintf("docker://%s:%s", srcRef, tag),
			fmt.Sprintf("docker://%s:%s", destRef, tag))...); err != nil {
			log.Error(err)
			errs = true
		}
	}

	if errs {
		return fmt.Errorf("errors during sync")
	}

	return nil
}
