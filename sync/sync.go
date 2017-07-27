/*
 *
 */

package sync

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/xelalexv/dregsy/docker"
)

//
var toTerminal bool

func init() {
	toTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
}

//
type Sync struct {
	client *docker.Client
}

//
func New(dockerhost, api string) (*Sync, error) {

	sync := &Sync{}

	var out io.Writer = sync
	if toTerminal {
		out = nil
	}

	if dockerhost == "" {
		dockerhost = client.DefaultDockerHost
	}
	if api == "" {
		api = "1.24"
	}

	cli, err := docker.NewClient(dockerhost, api, out)
	if err != nil {
		return nil, fmt.Errorf("cannot create Docker client: %v\n", err)
	}

	sync.client = cli
	return sync, nil
}

//
func (s *Sync) Dispose() {
	s.client.Close()
}

//
func (s *Sync) SyncFromConfig(conf *syncConfig) error {

	// one-off tasks
	for _, t := range conf.Tasks {
		if t.Interval == 0 {
			s.SyncTask(t)
		}
	}

	// periodic tasks
	c := make(chan *task)
	ticking := false
	for _, t := range conf.Tasks {
		if t.Interval > 0 {
			t.startTicking(c)
			ticking = true
		}
	}

	for ticking {
		LogPrintln()
		LogInfo("waiting for next sync task...")
		LogPrintln()
		s.SyncTask(<-c)
	}

	LogInfo("all done")
	return nil
}

//
func (s *Sync) SyncTask(t *task) {
	LogInfo("syncing task '%s': '%s' to '%s'...", t.Name, t.Source.Registry, t.Target.Registry)
	LogPrintln()
	for _, m := range t.Mappings {
		LogInfo("mapping '%s' to '%s'", m.From, m.To)
		LogPrintln()
		src, trgt := t.mappingRefs(m)
		if err := s.Sync(src, t.Source.Auth, trgt, t.Target.Auth, t.Verbose); err != nil {
			LogError(err)
		}
		LogPrintln()
	}
	LogPrintln()
}

//
func (s *Sync) Sync(srcRef, srcAuth, trgtRef, trgtAuth string, verbose bool) error {

	LogInfo("pulling all tags of source image '%s'", srcRef)
	if err := s.pull(srcRef, srcAuth, verbose); err != nil {
		return fmt.Errorf("error pulling source image '%s': %v\n", srcRef, err)
	}
	LogPrintln()

	LogInfo("relevant tags")
	srcImages, err := s.list(srcRef)
	if err != nil {
		LogError(fmt.Errorf("error listing all tags of source image '%s': %v\n", srcRef, err))
	}
	for _, img := range srcImages {
		LogInfo(" - %s", img.RefWithTags())
	}
	LogPrintln()

	LogInfo("setting tags for target image '%s'", trgtRef)
	_, err = s.tag(srcImages, trgtRef)
	if err != nil {
		return fmt.Errorf("error setting tags: %v\n", err)
	}
	LogPrintln()

	LogInfo("pushing all tags of target image")
	if err := s.push(trgtRef, trgtAuth, verbose); err != nil {
		return fmt.Errorf("error pushing target image: %v\n", err)
	}

	return nil
}

//
func (s *Sync) pull(ref, auth string, verbose bool) error {
	return s.client.PullImage(ref, true, auth, verbose)
}

//
func (s *Sync) list(ref string) ([]*docker.Image, error) {
	return s.client.ListImages(ref)
}

//
func (s *Sync) tag(images []*docker.Image, targetRef string) ([]*docker.Image, error) {

	taggedImages := []*docker.Image{}
	targetRepo, targetPath, _ := docker.SplitRef(targetRef)

	for _, img := range images {
		tagged := &docker.Image{
			ID:   img.ID,
			Repo: targetRepo,
			Path: targetPath,
			Tags: img.Tags,
		}
		for _, tag := range img.Tags {
			if err := s.client.TagImage(img.ID, fmt.Sprintf("%s:%s", tagged.Ref(), tag)); err != nil {
				return nil, err
			}
		}
		taggedImages = append(taggedImages, tagged)
	}

	return taggedImages, nil
}

//
func (s *Sync) push(ref, auth string, verbose bool) error {
	return s.client.PushImage(ref, true, auth, verbose)
}

// -----------------------------------------------------------------------------------------------------

//
func (s *Sync) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}

//
func LogPrintln() {
	LogInfo("")
}

//
func LogInfo(msg string, params ...interface{}) {
	msg = fmt.Sprintf(msg, params...)
	if !toTerminal {
		msg = fmt.Sprintf("[INFO] %s", msg)
	}
	fmt.Print(msg)
	if !strings.HasSuffix(msg, "\n") {
		fmt.Println()
	}
}

//
func LogError(err error) {
	if toTerminal {
		fmt.Fprintf(os.Stderr, "%v", err)
	} else {
		fmt.Fprintf(os.Stderr, "[ERROR] %v", err)
	}
}
