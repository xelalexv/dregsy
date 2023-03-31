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
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays"
	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
	"github.com/xelalexv/dregsy/internal/pkg/util"
)

//
type Relay interface {
	Prepare() error
	Dispose() error
	Sync(opt *relays.SyncOptions) error
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

	var relay Relay
	var err error

	switch conf.Relay {

	case docker.RelayID:
		if err = conf.ValidateSupport(&docker.Support{}); err == nil {
			relay, err = docker.NewDockerRelay(
				conf.Docker, log.StandardLogger().WriterLevel(log.DebugLevel))
		}

	case skopeo.RelayID:
		if err = conf.ValidateSupport(&skopeo.Support{}); err == nil {
			relay = skopeo.NewSkopeoRelay(
				conf.Skopeo, log.StandardLogger().WriterLevel(log.DebugLevel))
		}

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
func (s *Sync) SyncFromConfig(conf *SyncConfig, taskFilter string) (bool, error) {

	if taskFilter == "" {
		taskFilter = ".*"
	}

	tf, err := util.NewRegex(taskFilter)
	if err != nil {
		return false, fmt.Errorf("invalid task filter: %v", err)
	}

	if err := s.relay.Prepare(); err != nil {
		return false, err
	}

	// if the config file should not be watched, we receive an empty watcher
	// that will never produce any file events, but can still be used in the
	// main loop below
	watch, err := conf.watch()
	if err != nil {
		return false, err
	}
	defer watch.Close()

	restart := false

	for _, t := range conf.Tasks { // one-off tasks
		if t.Interval == 0 && tf.Matches(t.Name) {
			s.syncTask(t)
		}
	}

	c := make(chan *Task) // periodic tasks
	ticking := false
	for _, t := range conf.Tasks {
		if t.Interval > 0 && tf.Matches(t.Name) {
			t.startTicking(c)
			ticking = true
		}
	}

	sigs := make(chan os.Signal, 1) // watch for signals
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// rewrite of config file may result in many separate write events, we need
	// to use a timer to wait for a certain back off time until restarting; we
	// set up an initial, expired instance
	tChange := time.NewTimer(time.Millisecond)
	<-tChange.C

	var msg string

	for ticking { // main sync loop

		if msg != "" {
			log.Info(msg)
		}

		select {

		case t := <-c: // actual task
			s.syncTask(t)
			s.tick() // send a tick
			msg = "waiting for next sync task..."

		case sig := <-sigs: // signal
			log.WithField("signal", sig).Info("received signal")
			if sig == syscall.SIGHUP {
				log.Info("restarting ...")
				restart = true
			} else {
				log.Info("stopping ...")
			}
			ticking = false

		case evt, ok := <-watch.Events:
			msg = ""
			if ok && conf.isChanged(evt) {
				tChange.Stop()
				tChange = time.NewTimer(5 * time.Second)
			}

		case err, ok := <-watch.Errors: // config file watch errors
			tChange.Stop()
			msg = ""
			if ok {
				log.Warnf("error watching config file: %v", err)
			}

		case <-tChange.C: // back off time after last change expired, restart
			log.Info("config file changed, restarting ...")
			ticking = false
			restart = true

		case <-s.shutdown: // shutdown flagged
			log.Info("shutdown flagged, stopping ...")
			ticking = false
			s.tick() // send a final tick to release shutdown client
		}
	}

	log.Debug("stopping tasks")
	errs := false
	for _, t := range conf.Tasks {
		t.stopTicking()
		errs = errs || t.failed
	}

	if errs {
		return restart, fmt.Errorf(
			"one or more tasks had errors, please see log for details")
	}

	log.Info("all done")
	return restart, nil
}

//
func (s *Sync) syncTask(t *Task) {

	if t.tooSoon() {
		log.WithField("task", t.Name).Info("task fired too soon, skipping")
		return
	}

	log.WithFields(log.Fields{
		"task":   t.Name,
		"source": t.Source.Registry,
		"target": t.Target.Registry}).Info("syncing task")
	t.failed = false

	for _, m := range t.Mappings {

		log.WithFields(log.Fields{"from": m.From, "to": m.To}).Info("mapping")

		if err := t.Source.RefreshAuth(); err != nil {
			log.Error(err)
			t.fail(true)
			continue
		}
		if err := t.Target.RefreshAuth(); err != nil {
			log.Error(err)
			t.fail(true)
			continue
		}

		refs, err := t.mappingRefs(m)
		if err != nil {
			log.Error(err)
			t.fail(true)
			continue
		}

		for _, ref := range refs {

			src := ref[0]
			trgt := ref[1]

			if err := t.ensureTargetExists(trgt); err != nil {
				log.Error(err)
				t.fail(true)
				break
			}

			if err := s.relay.Sync(&relays.SyncOptions{
				SrcRef:            src,
				SrcAuth:           t.Source.GetAuth(),
				SrcSkipTLSVerify:  t.Source.SkipTLSVerify,
				TrgtRef:           trgt,
				TrgtAuth:          t.Target.GetAuth(),
				TrgtSkipTLSVerify: t.Target.SkipTLSVerify,
				Tags:              m.tagSet,
				Platform:          m.Platform,
				Verbose:           t.Verbose}); err != nil {
				log.Error(err)
				t.fail(true)
			}
		}
	}

	t.lastTick = time.Now()
}
