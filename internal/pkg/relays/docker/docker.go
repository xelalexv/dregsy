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

package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/blang/semver/v4"
	"github.com/distribution/reference"
	"github.com/docker/docker/api/types"
	typesimg "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/xelalexv/dregsy/internal/pkg/util"
)

//-
type image struct {
	id   string
	reg  string
	repo string
	tags []string
}

//-
func (s *image) ref() string {
	return fmt.Sprintf("%s/%s", s.reg, s.repo)
}

//-
func (s *image) refWithTags() string {
	return fmt.Sprintf("%s/%s:%v", s.reg, s.repo, s.tags)
}

//-
type dockerClient struct {
	host    string
	version string
	env     bool
	client  *client.Client
	wrOut   io.Writer
}

//-
func newClient(host, version string, out io.Writer) (*dockerClient, error) {
	dc := &dockerClient{
		host:    host,
		version: version,
		wrOut:   os.Stdout,
	}
	if out != nil {
		dc.wrOut = out
	}
	e := dc.open()
	return dc, e
}

//-
func newEnvClient() (*dockerClient, error) {
	dc := &dockerClient{
		env:   true,
		wrOut: os.Stdout,
	}
	err := dc.open()
	return dc, err
}

//-
func (dc *dockerClient) open() (err error) {
	if dc.client == nil {
		if dc.env {
			dc.client, err = client.NewEnvClient()
		} else {
			dc.client, err = client.NewClient(dc.host, dc.version, nil, nil)
		}
	}
	return
}

//-
func (dc *dockerClient) ping(attempts int, sleep time.Duration) (
	res types.Ping, err error) {

	for i := 1; ; i++ {
		if res, err = dc.client.Ping(context.Background()); err == nil {
			return
		}
		if i >= attempts {
			break
		}
		time.Sleep(sleep)
	}

	return types.Ping{}, fmt.Errorf(
		"unsuccessfully pinged Docker server %d times, last error: %s",
		attempts, err)
}

//-
func (dc *dockerClient) close() error {
	if dc.client != nil {
		return dc.client.Close()
	}
	return nil
}

//-
func (dc *dockerClient) listImages(ref string) (list []*image, err error) {

	log.WithField("ref", ref).Debug("listing images")
	imgs, err := dc.client.ImageList(
		context.Background(), typesimg.ListOptions{})
	if err != nil {
		return
	}

	fReg, fRepo, tag := util.SplitRef(ref) // the filter components
	name, dig := util.SplitTag(tag)
	fTag := name
	if dig != "" {
		fTag = dig
	}

	for _, img := range imgs {

		col := img.RepoTags // switch between the two lists depending on
		if dig != "" {      // whether we have a digest as tag filter
			col = img.RepoDigests
		}

		var i *image

		for _, rt := range col {

			if matched, err := match(fReg, fRepo, fTag, rt); err != nil {
				return list, err

			} else if matched {
				rg, rp, tg := util.SplitRef(rt)
				if i == nil {
					i = &image{
						id:   img.ID,
						reg:  rg,
						repo: rp,
					}
					list = append(list, i)
				}
				if tg != "" { // match has tag
					if dig != "" {
						// if we're using a digest as filter tag, we need to use
						// the full tag from ref, since it may also contain a
						// name, which would however be missing in tg; otherwise
						// we take the tag as is from the match
						tg = tag
					}
					i.tags = append(i.tags, tg)
				}
			}
		}
	}

	return
}

//-
func match(filterReg, filterRepo, filterTag, ref string) (bool, error) {

	if ref == "<none>:<none>" || ref == "<none>@<none>" {
		return false, nil
	}

	filter := fmt.Sprintf("%s/%s", filterReg, filterRepo)
	filterCanon, err := reference.ParseAnyReference(filter)
	if err != nil {
		return false, fmt.Errorf("malformed ref in filter '%s', %v", filter, err)
	}
	filterReg, filterRepo, _ = util.SplitRef(filterCanon.String())

	refCanon, err := reference.ParseAnyReference(ref)
	if err != nil {
		return false, fmt.Errorf("malformed image ref '%s': %v", ref, err)
	}

	reg, repo, tag := util.SplitRef(refCanon.String())
	return (filterReg == "" || filterReg == reg) &&
		(filterRepo == "" || filterRepo == repo) &&
		(filterTag == "" || filterTag == tag), nil
}

//-
func (dc *dockerClient) pullImage(ref string, allTags bool, platform, auth string,
	verbose bool) error {
	opts := &typesimg.PullOptions{
		All:          allTags,
		RegistryAuth: auth,
		Platform:     platform,
	}
	rc, err := dc.client.ImagePull(context.Background(), ref, *opts)
	return dc.handleLog(rc, err, verbose)
}

//-
func (dc *dockerClient) pushImage(image string, allTags bool, platform, auth string,
	verbose bool) error {

	plat := toPlatform(platform)
	if plat != nil && !dc.supportsPlatform() {
		log.Warnf(
			"'platform' requires at least API version 1.46, but you're using %s; dropping platform constraint",
			dc.client.ClientVersion())
		plat = nil
	}

	opts := &typesimg.PushOptions{
		All:          allTags,
		RegistryAuth: auth,
		Platform:     plat,
	}
	rc, err := dc.client.ImagePush(context.Background(), image, *opts)
	return dc.handleLog(rc, err, verbose)
}

//-
func (dc *dockerClient) tagImage(source, target string) error {
	return dc.client.ImageTag(context.Background(), source, target)
}

//-
func (dc *dockerClient) handleLog(rc io.ReadCloser, err error, verbose bool) error {

	if err != nil {
		return err
	}
	defer rc.Close()
	out := dc.wrOut
	if !verbose {
		out = ioutil.Discard
	}
	terminalFd := os.Stdout.Fd()
	isTerminal := dc.wrOut == os.Stdout && terminal.IsTerminal(int(terminalFd))
	return jsonmessage.DisplayJSONMessagesStream(
		rc, out, terminalFd, isTerminal, nil)
}

//-
func (dc *dockerClient) supportsPlatform() bool {

	ver := dc.client.ClientVersion()

	if v, err := semver.ParseTolerant(ver); err != nil {
		log.Errorf("could not parse Docker API version: %s - %v", ver, err)
		return false
	} else {
		min, _ := semver.ParseTolerant("1.46.0")
		return v.GE(min)
	}
}
