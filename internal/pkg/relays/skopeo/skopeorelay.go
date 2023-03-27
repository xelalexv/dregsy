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
	"strings"

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
	wrOut  io.Writer
	dryRun bool
}

//
func NewSkopeoRelay(conf *RelayConfig, out io.Writer, dry bool) *SkopeoRelay {

	relay := &SkopeoRelay{dryRun: dry}

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

	if r.dryRun {
		desCertDir := ""
		repo, _, _ := util.SplitRef(opt.TrgtRef)
		if repo != "" {
			desCertDir = CertsDirForRepo(repo)
		}
		trgtTags, err := ListAllTags(
			opt.TrgtRef, destCreds, desCertDir, opt.TrgtSkipTLSVerify)

		if err != nil {
			// Not so sure parsing the error is the best solution but
			// alt could be pre-check with an http request to `/v2/_catalog`
			// and check if the repository is in the list
			if strings.Contains(err.Error(), "registry 404 (Not Found)") {
				log.Warnf("[dry-run] Target repository not found. setting the target list as empty list.")
				trgtTags = []string{}
			} else {
				log.Errorf("[dry-run] unknon error trying to expand tags from target [%s]: %v", opt.TrgtRef, err)
			}
		}
		// not yet dumping the information into a file, will do later
		util.DumpMapAsJson(map[string]interface{}{
			"task name":                                    opt.Task,
			"task index":                                   opt.Index,
			"source reference":                             opt.SrcRef,
			"target reference":                             opt.TrgtRef,
			"tags to sync from source":                     tags,
			"amount of tags to be sync from source":        len(tags),
			"tags available on target":                     trgtTags,
			"tags available to be synced not synced yet":   util.DiffBetweenLists(tags, trgtTags),
			"amount of tags available on target":           len(trgtTags),
			"tags available on target that are not synced": util.DiffBetweenLists(trgtTags, tags),
		}, fmt.Sprintf("dregsy-%s-%d-dry-run-report.json", opt.Task, opt.Index))

		// stop here otherwise the amount of if/else would explode as every following action
		// will need to be skip
		return nil
	}

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
