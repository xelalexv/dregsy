/*
 *
 */

package sync

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/xelalexv/dregsy/internal/pkg/log"
	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
)

//
type Relay interface {
	Prepare() error
	Dispose()
	Sync(srcRef, srcAuth string, srcSkiptTLSVerify bool,
		trgtRef, trgtAuth string, trgtSkiptTLSVerify bool,
		tags []string, verbose bool) error
}

//
type Sync struct {
	relay Relay
}

//
func New(conf *syncConfig) (*Sync, error) {

	sync := &Sync{}

	var out io.Writer = sync
	if log.ToTerminal {
		out = nil
	}

	var relay Relay
	var err error

	switch conf.Relay {

	case docker.RelayID:
		relay, err = docker.NewDockerRelay(conf.Docker, out)

	case skopeo.RelayID:
		relay = skopeo.NewSkopeoRelay(conf.Skopeo, out)

	default:
		err = fmt.Errorf("relay type '%s' not supported", conf.Relay)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot create sync relay: %v", err)
	}

	sync.relay = relay
	return sync, nil
}

//
func (s *Sync) Dispose() {
	s.relay.Dispose()
}

//
func (s *Sync) SyncFromConfig(conf *syncConfig) error {

	if log.Error(s.relay.Prepare()) {
		os.Exit(1)
	}

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
		log.Info("waiting for next sync task...")
		log.Println()
		s.SyncTask(<-c)
	}

	log.Info("all done")
	return nil
}

//
func (s *Sync) SyncTask(t *task) {

	if t.tooSoon() {
		log.Info("task '%s' fired too soon, skipping", t.Name)
		return
	}

	log.Info("syncing task '%s': '%s' --> '%s'",
		t.Name, t.Source.Registry, t.Target.Registry)

	for _, m := range t.Mappings {
		log.Info("mapping '%s' to '%s'", m.From, m.To)
		src, trgt := t.mappingRefs(m)
		log.Error(t.Source.refreshAuth())
		log.Error(t.Target.refreshAuth())
		log.Error(t.ensureTargetExists(trgt))
		log.Error(s.relay.Sync(
			src, t.Source.Auth, t.Source.SkipTLSVerify,
			trgt, t.Target.Auth, t.Target.SkipTLSVerify,
			m.Tags, t.Verbose))
	}

	t.lastTick = time.Now()
	log.Println()
}

//
func (s *Sync) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}
