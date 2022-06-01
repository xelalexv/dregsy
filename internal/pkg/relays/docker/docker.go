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

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/xelalexv/dregsy/internal/pkg/util"
)

//
type image struct {
	ID   string
	Repo string
	Path string
	Tags []string
}

//
func (s *image) ref() string {
	return fmt.Sprintf("%s/%s", s.Repo, s.Path)
}

//
func (s *image) refWithTags() string {
	return fmt.Sprintf("%s/%s:%v", s.Repo, s.Path, s.Tags)
}

//
type dockerClient struct {
	host    string
	version string
	env     bool
	client  *client.Client
	wrOut   io.Writer
}

//
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

//
func newEnvClient() (*dockerClient, error) {
	dc := &dockerClient{
		env:   true,
		wrOut: os.Stdout,
	}
	err := dc.open()
	return dc, err
}

//
func (dc *dockerClient) open() error {
	var err error
	if dc.client == nil {
		if dc.env {
			dc.client, err = client.NewEnvClient()
		} else {
			dc.client, err = client.NewClient(dc.host, dc.version, nil, nil)
		}
	}
	return err
}

//
func (dc *dockerClient) ping(attempts int, sleep time.Duration) (
	types.Ping, error) {
	var err error
	for i := 1; ; i++ {
		if res, err := dc.client.Ping(context.Background()); err == nil {
			return res, err
		}
		if i >= attempts {
			break
		}
		time.Sleep(sleep)
	}
	return types.Ping{},
		fmt.Errorf(
			"unsuccessfully pinged Docker server %d times, last error: %s",
			attempts, err)
}

//
func (dc *dockerClient) close() error {
	var err error
	if dc.client != nil {
		err = dc.client.Close()
	}
	return err
}

//
func (dc *dockerClient) listImages(ref string) ([]*image, error) {

	imgs, err := dc.client.ImageList(
		context.Background(), types.ImageListOptions{})
	ret := []*image{}

	if err == nil {
		fRepo, fPath, fTag := util.SplitRef(ref)
		for _, img := range imgs {
			var i *image
			for _, rt := range img.RepoTags {
				matched, err := match(fRepo, fPath, fTag, rt)
				if err != nil {
					return ret, err
				}
				if matched {
					repo, path, tag := util.SplitRef(rt)
					if i == nil {
						i = &image{
							ID:   img.ID,
							Repo: repo,
							Path: path,
						}
						ret = append(ret, i)
					}
					if tag != "" {
						i.Tags = append(i.Tags, tag)
					}
				}
			}
		}
	}

	return ret, err
}

//
func match(filterRepo, filterPath, filterTag, ref string) (bool, error) {

	filter := fmt.Sprintf("%s/%s", filterRepo, filterPath)
	filterCanon, err := reference.ParseAnyReference(filter)
	if err != nil {
		return false, fmt.Errorf("malformed ref in filter '%s', %v", filter, err)
	}
	filterRepo, filterPath, _ = util.SplitRef(filterCanon.String())

	refCanon, err := reference.ParseAnyReference(ref)
	if err != nil {
		return false, fmt.Errorf("malformed image ref '%s': %v", ref, err)
	}

	repo, path, tag := util.SplitRef(refCanon.String())
	return (filterRepo == "" || filterRepo == repo) &&
		(filterPath == "" || filterPath == path) &&
		(filterTag == "" || filterTag == tag), nil
}

//
func (dc *dockerClient) pullImage(ref string, allTags bool, platform, auth string,
	verbose bool) error {
	opts := &types.ImagePullOptions{
		All:          allTags,
		RegistryAuth: auth,
		Platform:     platform,
	}
	rc, err := dc.client.ImagePull(context.Background(), ref, *opts)
	return dc.handleLog(rc, err, verbose)
}

//
func (dc *dockerClient) pushImage(image string, allTags bool, platform, auth string,
	verbose bool) error {

	opts := &types.ImagePushOptions{
		All:          allTags,
		RegistryAuth: auth,
		// NOTE: Platform currently does not seem to be used by
		//       the Docker client lib
		Platform: platform,
	}
	rc, err := dc.client.ImagePush(context.Background(), image, *opts)
	return dc.handleLog(rc, err, verbose)
}

//
func (dc *dockerClient) tagImage(source, target string) error {
	return dc.client.ImageTag(context.Background(), source, target)
}

//
func (dc *dockerClient) handleLog(rc io.ReadCloser, err error,
	verbose bool) error {

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
