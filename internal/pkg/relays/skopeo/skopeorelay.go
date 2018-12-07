package skopeo

import (
	"bytes"
	"fmt"
	"io"

	"github.com/xelalexv/dregsy/internal/pkg/log"
	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
)

const RelayID = "skopeo"

//
type RelayConfig struct {
	Binary   string `yaml:"binary"`
	CertsDir string `yaml:"certs-dir"`
}

//
type SkopeoRelay struct {
	wrOut io.Writer
}

//
func NewSkopeoRelay(conf *RelayConfig, out io.Writer) *SkopeoRelay {

	relay := &SkopeoRelay{}

	if out != nil {
		relay.wrOut = out
	}
	if conf != nil {
		if conf.Binary != "" {
			skopeoBinary = conf.Binary
		}
		if conf.CertsDir != "" {
			certsBaseDir = conf.CertsDir
		}
	}

	return relay
}

//
func (r *SkopeoRelay) Prepare() error {
	bufOut := new(bytes.Buffer)
	if err := runSkopeo(bufOut, nil, true, "--version"); err != nil {
		return fmt.Errorf("cannot execute skopeo: %v", err)
	}
	log.Println()
	log.Info(bufOut.String())
	log.Info("%s relay ready", RelayID)
	return nil
}

//
func (r *SkopeoRelay) Dispose() {
}

//
func (r *SkopeoRelay) Sync(srcRef, srcAuth string, srcSkipTLSVerify bool,
	destRef, destAuth string, destSkipTLSVerify bool,
	tags []string, verbose bool) error {

	srcCreds := decodeJSONAuth(srcAuth)
	destCreds := decodeJSONAuth(destAuth)

	cmd := []string{
		"--insecure-policy",
		"copy",
	}

	if srcSkipTLSVerify {
		cmd = append(cmd, "--src-tls-verify=false")
	}
	if destSkipTLSVerify {
		cmd = append(cmd, "--dest-tls-verify=false")
	}

	srcCertDir := ""
	repo, _, _ := docker.SplitRef(srcRef)
	if repo != "" {
		srcCertDir = fmt.Sprintf("%s/%s", certsBaseDir, withoutPort(repo))
		cmd = append(cmd, fmt.Sprintf("--src-cert-dir=%s", srcCertDir))
	}
	repo, _, _ = docker.SplitRef(destRef)
	if repo != "" {
		cmd = append(cmd, fmt.Sprintf(
			"--dest-cert-dir=%s/%s", certsBaseDir, withoutPort(repo)))
	}

	if srcCreds != "" {
		cmd = append(cmd, fmt.Sprintf("--src-creds=%s", srcCreds))
	}
	if destCreds != "" {
		cmd = append(cmd, fmt.Sprintf("--dest-creds=%s", destCreds))
	}

	var err error

	if len(tags) == 0 {
		tags, err = listAllTags(srcRef, srcCreds, srcCertDir, srcSkipTLSVerify)
		if err != nil {
			return err
		}
	}

	for _, tag := range tags {
		log.Println()
		log.Info("syncing tag '%s':", tag)
		log.Error(
			runSkopeo(r.wrOut, r.wrOut, verbose,
				append(cmd,
					fmt.Sprintf("docker://%s:%s", srcRef, tag),
					fmt.Sprintf("docker://%s:%s", destRef, tag))...))
	}

	return nil
}
