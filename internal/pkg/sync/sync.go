/*
 *
 */

package sync

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
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
type sync struct {
	relay Relay
}

//
func New(conf *syncConfig) (*sync, error) {

	sync := &sync{}

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
func (s *sync) Dispose() {
	s.relay.Dispose()
}

//
func (s *sync) SyncFromConfig(conf *syncConfig) error {

	if err := s.relay.Prepare(); err != nil {
		return err
	}
	log.Println()

	// one-off tasks
	for _, t := range conf.Tasks {
		if t.Interval == 0 {
			s.syncTask(t)
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for ticking {
		log.Info("waiting for next sync task...")
		log.Println()
		select {
		case t := <-c:
			s.syncTask(t)
		case sig := <-sigs:
			log.Info("\nreceived '%v' signal, stopping ...\n", sig)
			ticking = false
		}
	}

	errs := false
	for _, t := range conf.Tasks {
		t.stopTicking(c)
		errs = errs || t.failed
	}

	if errs {
		return fmt.Errorf(
			"one or more tasks had errors, please see log for details")
	}

	log.Info("all done")
	return nil
}

//
func (s *sync) syncTask(t *task) {

	if t.tooSoon() {
		log.Info("task '%s' fired too soon, skipping", t.Name)
		return
	}

	log.Info("syncing task '%s': '%s' --> '%s'",
		t.Name, t.Source.Registry, t.Target.Registry)
	t.failed = false

	for _, m := range t.Mappings {
		log.Info("mapping '%s' to '%s'", m.From, m.To)
		src, trgt := t.mappingRefs(m)
		t.fail(log.Error(t.Source.refreshAuth()))
		t.fail(log.Error(t.Target.refreshAuth()))
		t.fail(log.Error(t.ensureTargetExists(trgt)))
		t.fail(log.Error(s.relay.Sync(
			src, t.Source.Auth, t.Source.SkipTLSVerify,
			trgt, t.Target.Auth, t.Target.SkipTLSVerify,
			m.Tags, t.Verbose)))
	}

	t.lastTick = time.Now()
	log.Println()
}

//
func (s *sync) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}
