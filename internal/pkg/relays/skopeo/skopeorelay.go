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

//
type RelayConfig struct {
	Binary   string `yaml:"binary"`
	CertsDir string `yaml:"certs-dir"`
}

//
type Support struct{}

//
func (s *Support) Platform(p string) error {
	return nil
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
func (r *SkopeoRelay) Sync(opt *relays.SyncOptions) error {

	srcCreds := util.DecodeJSONAuth(opt.SrcAuth)
	destCreds := util.DecodeJSONAuth(opt.TrgtAuth)

	cmd := []string{
		"--insecure-policy",
		"copy",
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

	tags, err := opt.Tags.Expand(func() ([]string, error) {
		return ListAllTags(opt.SrcRef, srcCreds, srcCertDir, opt.SrcSkipTLSVerify)
	})

	if err != nil {
		return fmt.Errorf("error expanding tags: %v", err)
	}

	errs := false

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
		return fmt.Errorf("errors during sync")
	}

	return nil
}
