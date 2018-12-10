package docker

import (
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/client"

	"github.com/xelalexv/dregsy/internal/pkg/log"
)

const RelayID = "docker"

//
type RelayConfig struct {
	DockerHost string `yaml:"dockerhost"`
	APIVersion string `yaml:"api-version"`
}

//
type DockerRelay struct {
	client *dockerClient
}

//
func NewDockerRelay(conf *RelayConfig, out io.Writer) (*DockerRelay, error) {

	relay := &DockerRelay{}

	dockerHost := client.DefaultDockerHost
	apiVersion := "1.24"

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

//
func (r *DockerRelay) Prepare() error {

	// when we begin, Docker daemon may not be ready yet, e.g. when dregsy runs
	// side by side with a Docker-in-Docker container inside a pod on k8s
	log.Println()
	log.Info("pinging Docker daemon...")

	if _, err := r.client.ping(30, 10*time.Second); err != nil {
		return err
	}

	log.Info("ok, %s relay ready", RelayID)
	return nil
}

//
func (r *DockerRelay) Dispose() {
	r.client.close()
}

//
func (r *DockerRelay) Sync(srcRef, srcAuth string, srcSkipTLSVerify bool,
	trgtRef, trgtAuth string, trgtSkipTLSVerify bool,
	tags []string, verbose bool) error {

	log.Info("pulling source image '%s'", srcRef)
	var err error

	if len(tags) == 0 {
		if err = r.pull(srcRef, srcAuth, true, verbose); err != nil {
			return fmt.Errorf(
				"error pulling source image '%s': %v", srcRef, err)
		}

	} else {
		for _, tag := range tags {
			srcRefTagged := fmt.Sprintf("%s:%s", srcRef, tag)
			if err = r.pull(srcRefTagged, srcAuth, false, verbose); err != nil {
				return fmt.Errorf(
					"error pulling source image '%s': %v", srcRefTagged, err)
			}
		}
	}

	log.Info("relevant tags")
	var srcImages []*image

	if len(tags) == 0 {
		srcImages, err = r.list(srcRef)
		if err != nil {
			log.Error(
				fmt.Errorf("error listing all tags of source image '%s': %v",
					srcRef, err))
		}

	} else {
		for _, tag := range tags {
			srcRefTagged := fmt.Sprintf("%s:%s", srcRef, tag)
			srcImageTagged, err := r.list(srcRefTagged)
			if err != nil {
				log.Error(
					fmt.Errorf("error listing source image '%s': %v",
						srcRefTagged, err))
			}
			srcImages = append(srcImages, srcImageTagged...)
		}
	}

	for _, img := range srcImages {
		log.Info(" - %s", img.refWithTags())
	}

	log.Info("setting tags for target image '%s'", trgtRef)

	_, err = r.tag(srcImages, trgtRef)
	if err != nil {
		return fmt.Errorf("error setting tags: %v", err)
	}

	log.Info("pushing target image '%s'", trgtRef)

	if err := r.push(trgtRef, trgtAuth, verbose); err != nil {
		return fmt.Errorf("error pushing target image: %v", err)
	}

	return nil
}

//
func (r *DockerRelay) pull(ref, auth string, allTags, verbose bool) error {
	return r.client.pullImage(ref, allTags, auth, verbose)
}

//
func (r *DockerRelay) list(ref string) ([]*image, error) {
	return r.client.listImages(ref)
}

//
func (r *DockerRelay) tag(images []*image, targetRef string) (
	[]*image, error) {

	taggedImages := []*image{}
	targetRepo, targetPath, _ := SplitRef(targetRef)

	for _, img := range images {
		tagged := &image{
			ID:   img.ID,
			Repo: targetRepo,
			Path: targetPath,
			Tags: img.Tags,
		}
		for _, tag := range img.Tags {
			if err := r.client.tagImage(img.ID, fmt.Sprintf("%s:%s",
				tagged.ref(), tag)); err != nil {
				return nil, err
			}
		}
		taggedImages = append(taggedImages, tagged)
	}

	return taggedImages, nil
}

//
func (r *DockerRelay) push(ref, auth string, verbose bool) error {
	return r.client.pushImage(ref, true, auth, verbose)
}
