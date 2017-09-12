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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/moby/moby/pkg/jsonmessage"
	"golang.org/x/crypto/ssh/terminal"
)

//
type Image struct {
	ID   string
	Repo string
	Path string
	Tags []string
}

//
func (s *Image) Ref() string {
	return fmt.Sprintf("%s/%s", s.Repo, s.Path)
}

//
func (s *Image) RefWithTags() string {
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

	return repo, path, tag
}

//
type Client struct {
	host    string
	version string
	env     bool
	client  *client.Client
	wrOut   io.Writer
}

//
func NewClient(host, version string, out io.Writer) (*Client, error) {
	dc := &Client{
		host:    host,
		version: version,
		wrOut:   os.Stdout,
	}
	if out != nil {
		dc.wrOut = out
	}
	e := dc.Open()
	return dc, e
}

//
func NewEnvClient() (*Client, error) {
	dc := &Client{
		env:   true,
		wrOut: os.Stdout,
	}
	err := dc.Open()
	return dc, err
}

//
func (dc *Client) Open() error {
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
func (dc *Client) Ping(attempts int, sleep time.Duration) (types.Ping, error) {
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
		fmt.Errorf("unsuccessfully pinged Docker server %d times, last error: %s", attempts, err)
}

//
func (dc *Client) Close() error {
	var err error
	if dc.client != nil {
		err = dc.client.Close()
	}
	return err
}

//
func (dc *Client) ListImages(ref string) ([]*Image, error) {

	imgs, err := dc.client.ImageList(context.Background(), types.ImageListOptions{})
	ret := []*Image{}

	if err == nil {
		fRepo, fPath, fTag := SplitRef(ref)
		for _, img := range imgs {
			var i *Image
			for _, rt := range img.RepoTags {
				if match(fRepo, fPath, fTag, rt) {
					repo, path, tag := SplitRef(rt)
					if i == nil {
						i = &Image{
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
func (dc *Client) PullImage(ref string, allTags bool, auth string, verbose bool) error {
	opts := &types.ImagePullOptions{
		All:          allTags,
		RegistryAuth: auth,
	}
	rc, err := dc.client.ImagePull(context.Background(), ref, *opts)
	return dc.handleLog(rc, err, verbose)
}

//
func (dc *Client) PushImage(image string, allTags bool, auth string, verbose bool) error {
	opts := &types.ImagePushOptions{
		All:          allTags,
		RegistryAuth: auth,
	}
	rc, err := dc.client.ImagePush(context.Background(), image, *opts)
	return dc.handleLog(rc, err, verbose)
}

//
func (dc *Client) TagImage(source, target string) error {
	return dc.client.ImageTag(context.Background(), source, target)
}

//
func (dc *Client) handleLog(rc io.ReadCloser, err error, verbose bool) error {
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
	return jsonmessage.DisplayJSONMessagesStream(rc, out, terminalFd, isTerminal, nil)
}
