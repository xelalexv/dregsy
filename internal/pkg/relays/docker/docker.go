/*
 *
 */

package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"golang.org/x/crypto/ssh/terminal"
)

//
type image struct {
	ID   string
	Repo string
	Path string
	Tags []string
}

// replace "docker.io/usr/appname:tag" with "usr/apname:tag" when searching locally
var re = regexp.MustCompile(`^docker.io/`)

//
func (s *image) ref() string {
	return fmt.Sprintf("%s/%s", s.Repo, s.Path)
}

//
func (s *image) refWithTags() string {
	return fmt.Sprintf("%s/%s:%v", s.Repo, s.Path, s.Tags)
}

//
func SplitRef(ref string) (repo, path, tag string) {

	ix := strings.Index(ref, "/")

	if ix == -1 {
		repo = ""
		path = ref
	} else {
		repo = ref[:ix]
		path = ref[ix+1:]
	}

	ix = strings.Index(path, ":")

	if ix > -1 {
		tag = path[ix+1:]
		path = path[:ix]
	}

	return
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
		ref = re.ReplaceAllString(ref, "")
		fRepo, fPath, fTag := SplitRef(ref)
		for _, img := range imgs {
			var i *image
			for _, rt := range img.RepoTags {
				if match(fRepo, fPath, fTag, rt) {
					repo, path, tag := SplitRef(rt)
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
func match(filterRepo, filterPath, filterTag, ref string) bool {
	repo, path, tag := SplitRef(ref)
	return (filterRepo == "" || filterRepo == repo) &&
		(filterPath == "" || filterPath == path) &&
		(filterTag == "" || filterTag == tag)
}

//
func (dc *dockerClient) pullImage(ref string, allTags bool, auth string,
	verbose bool) error {
	opts := &types.ImagePullOptions{
		All:          allTags,
		RegistryAuth: auth,
	}
	rc, err := dc.client.ImagePull(context.Background(), ref, *opts)
	return dc.handleLog(rc, err, verbose)
}

//
func (dc *dockerClient) pushImage(image string, allTags bool, auth string,
	verbose bool) error {

	opts := &types.ImagePushOptions{
		All:          allTags,
		RegistryAuth: auth,
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
