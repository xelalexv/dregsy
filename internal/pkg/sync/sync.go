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
	Dispose() error
	Sync(srcRef, srcAuth string, srcSkiptTLSVerify bool,
		trgtRef, trgtAuth string, trgtSkiptTLSVerify bool,
		tags []string, verbose bool) error
}

//
type Sync struct {
	relay    Relay
	shutdown chan bool
	ticks    chan bool
}

//
func New(conf *SyncConfig) (*Sync, error) {

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
	sync.shutdown = make(chan bool)
	sync.ticks = make(chan bool, 1)

	return sync, nil
}

//
func (s *Sync) Shutdown() {
	s.shutdown <- true
	s.WaitForTick()
}

//
func (s *Sync) tick() {
	select {
	case s.ticks <- true:
	default:
	}
}

//
func (s *Sync) WaitForTick() {
	<-s.ticks
}

//
func (s *Sync) Dispose() {
	s.relay.Dispose()
}

//
func (s *Sync) SyncFromConfig(conf *SyncConfig) error {

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
	c := make(chan *Task)
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
		case t := <-c: // actual task
			s.syncTask(t)
			s.tick() // send a tick
		case sig := <-sigs: // interrupt signal
			log.Info("\nreceived '%v' signal, stopping ...\n", sig)
			ticking = false
		case <-s.shutdown: // shutdown flagged
			log.Info("\nshutdown flagged, stopping ...\n")
			ticking = false
			s.tick() // send a final tick to release shutdown client
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
func (s *Sync) syncTask(t *Task) {

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
		t.fail(log.Error(t.Source.RefreshAuth()))
		t.fail(log.Error(t.Target.RefreshAuth()))
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
func (s *Sync) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}
