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
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays"
	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
	"github.com/xelalexv/dregsy/internal/pkg/util"
)

const RelayID = "docker"

//-
type RelayConfig struct {
	DockerHost string `yaml:"dockerhost"`
	APIVersion string `yaml:"api-version"`
}

//-
type Support struct{}

//-
func (s *Support) Platform(p string) error {
	if p == "all" {
		return fmt.Errorf(
			"relay '%s' does not support mappings with 'platform: all'", RelayID)
	}
	return nil
}

//-
type DockerRelay struct {
	client *dockerClient
}

//-
func NewDockerRelay(conf *RelayConfig, out io.Writer) (*DockerRelay, error) {

	relay := &DockerRelay{}

	dockerHost := client.DefaultDockerHost
	apiVersion := "1.41"

	if conf != nil {
		if conf.DockerHost != "" {
			dockerHost = conf.DockerHost
		}
		if conf.APIVersion != "" {
			apiVersion = conf.APIVersion
		}
	}

	cli, err := newClient(dockerHost, apiVersion, out)
	if err != nil {
		return nil, fmt.Errorf("cannot create Docker client: %v", err)
	}

	relay.client = cli
	return relay, nil
}

//-
func (r *DockerRelay) Prepare() error {

	// when we begin, Docker daemon may not be ready yet, e.g. when dregsy runs
	// side by side with a Docker-in-Docker container inside a pod on k8s
	log.Info("pinging Docker daemon...")

	if _, err := r.client.ping(30, 10*time.Second); err != nil {
		return err
	}

	log.WithField("relay", RelayID).Info("ok, relay ready")
	return nil
}

//-
func (r *DockerRelay) Dispose() error {
	log.WithField("relay", RelayID).Info("disposing relay")
	return r.client.close()
}

//-
func (r *DockerRelay) Sync(opt *relays.SyncOptions) error {

	log.WithFields(log.Fields{
		"ref":      opt.SrcRef,
		"platform": opt.Platform}).Info("pulling source image")

	if opt.Platform == "all" {
		return fmt.Errorf("'Platform: all' sync option not supported")
	}

	var tags []string
	var err error

	// When no tags are specified, a simple docker pull without a tag will get
	// all tags. So for that case, we don't need to list tags.

	if !opt.Tags.IsEmpty() {
		var certs string
		reg, _, _ := util.SplitRef(opt.SrcRef)
		if reg != "" {
			certs = skopeo.CertsDirForRegistry(reg)
		}
		tags, err = opt.Tags.Expand(func() ([]string, error) {
			return skopeo.ListAllTags(
				opt.SrcRef, util.DecodeJSONAuth(opt.SrcAuth),
				certs, opt.SrcSkipTLSVerify)
		})

		if err != nil {
			return fmt.Errorf("error expanding tags: %v", err)
		}
	}

	if len(tags) == 0 { // pull all tags
		if err = r.pull(opt.SrcRef, opt.Platform, opt.SrcAuth,
			true, opt.Verbose); err != nil {
			return fmt.Errorf(
				"error pulling source image '%s': %v", opt.SrcRef, err)
		}

	} else { // pull tag by tag; tags with digest are pulled by digest only
		for _, tag := range tags {
			tagged, _ := util.JoinRefsAndTag(opt.SrcRef, "", tag)
			if err = r.pull(tagged, opt.Platform, opt.SrcAuth,
				false, opt.Verbose); err != nil {
				return fmt.Errorf(
					"error pulling source image '%s': %v", tagged, err)
			}
		}
	}

	// Now that we've pulled all required tags, we need to get the image IDs
	// for pushing to the target registry. For this, we list all images that are
	// currently present, and then filter by registry, repo & tags. We do this
	// rather than using filter expressions in Docker list options, to maintain
	// better control over the reference matching. Docker may refer to images
	// originating from DockerHub in the short form, which would not match fully
	// qualified references that could be used in the sync config. In our own
	// filtering, we canonicalize all references before matching.

	log.Info("relevant tags:")
	var srcImages []*image

	if len(tags) == 0 { // use all local images that match source reference
		srcImages, err = r.list(opt.SrcRef)
		if err != nil {
			log.Errorf("error listing all tags of source image '%s': %v",
				opt.SrcRef, err)
		}

	} else { // filter local images by source reference and tags
		for _, tag := range tags {
			tagged := util.JoinRefAndTag(opt.SrcRef, tag)
			imgs, err := r.list(tagged)
			if err != nil {
				log.Errorf("error listing source image '%s': %v", tagged, err)
			}
			srcImages = append(srcImages, imgs...)
		}
	}

	for _, img := range srcImages {
		log.Infof(" - %s", img.refWithTags())
	}

	log.WithField("ref", opt.TrgtRef).Info("setting tags for target image")

	// We now tag the source images for the target registry.
	if err = r.tag(srcImages, opt.TrgtRef); err != nil {
		return fmt.Errorf("error setting tags: %v", err)
	}

	log.WithFields(log.Fields{
		"ref":      opt.TrgtRef,
		"platform": opt.Platform}).Info("pushing target image")

	// Finally, the images are pushed to the target. This includes all present
	// tags.
	//
	// FIXME: target tags should be removed to not interfere with tag count
	//        limiting
	//
	if err := r.push(
		opt.TrgtRef, opt.Platform, opt.TrgtAuth, opt.Verbose); err != nil {
		return fmt.Errorf("error pushing target image: %v", err)
	}

	return nil
}

//-
func (r *DockerRelay) pull(ref, platform, auth string, allTags, verbose bool) error {
	return r.client.pullImage(ref, allTags, platform, auth, verbose)
}

//-
func (r *DockerRelay) list(ref string) ([]*image, error) {
	return r.client.listImages(ref)
}

//-
func (r *DockerRelay) tag(images []*image, targetRef string) error {

	for _, img := range images {
		for _, tag := range img.tags {
			if tag != "" {
				n, d := util.SplitTag(tag)
				if n == "" {
					// Docker does not support pushing by digest only ref; we
					// therefore auto-generate a tag using the digest value
					log.Debug("generating tag for digest only ref")
					n = tagFromDigest(d)
				}

				log.WithFields(
					log.Fields{"ref": targetRef, "tag": n}).Debug("tagging")

				if err := r.client.tagImage(
					img.id, fmt.Sprintf("%s:%s", targetRef, n)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

//-
func (r *DockerRelay) push(ref, platform, auth string, verbose bool) error {
	return r.client.pushImage(ref, true, platform, auth, verbose)
}

//-
func tagFromDigest(d string) string {

	if util.IsDigest(d) {
		d = d[len(util.DigestPrefix):]
	}

	ret := fmt.Sprintf("dregsy-%s", d)
	if len(ret) > 128 {
		ret = ret[:128]
	}
	return ret
}
