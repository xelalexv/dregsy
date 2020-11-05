package skopeo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/xelalexv/dregsy/internal/pkg/log"
)

const defaultSkopeoBinary = "skopeo"
const defaultCertsBaseDir = "/etc/skopeo/certs.d"

var skopeoBinary string
var certsBaseDir string

//
func init() {
	skopeoBinary = defaultSkopeoBinary
	certsBaseDir = defaultCertsBaseDir
}

//
type creds struct {
	Username string
	Password string
}

//
type tagList struct {
	Repository string   `json:"Repository"`
	Tags       []string `json:"Tags"`
}

//
func listAllTags(ref, creds, certDir string, skipTLSVerify bool) (
	[]string, error) {

	cmd := []string{
		"list-tags",
	}

	if skipTLSVerify {
		cmd = append(cmd, "--tls-verify=false")
	}

	if creds != "" {
		cmd = append(cmd, fmt.Sprintf("--creds=%s", creds))
	}

	if certDir != "" {
		cmd = append(cmd, fmt.Sprintf("--cert-dir=%s", certDir))
	}

	cmd = append(cmd, "docker://"+ref)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	if err := runSkopeo(bufOut, bufErr, true, cmd...); err != nil {
		return nil,
			fmt.Errorf("error listing image tags: %s, %v", bufErr.String(), err)
	}

	list, err := decodeTagList(bufOut.Bytes())

	if err != nil {
		return nil, err
	}
	return list.Tags, nil
}

//
func chooseOutStream(out io.Writer, verbose, isErrorStream bool) io.Writer {
	if verbose {
		if out != nil {
			return out
		}
		if log.ToTerminal {
			if isErrorStream {
				return os.Stderr
			}
			return os.Stdout
		}
	}
	return ioutil.Discard
}

//
func runSkopeo(outWr, errWr io.Writer, verbose bool, args ...string) error {

	cmd := exec.Command(skopeoBinary, args...)

	cmd.Stdout = chooseOutStream(outWr, verbose, false)
	cmd.Stderr = chooseOutStream(errWr, verbose, true)

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

//
func decodeTagList(tl []byte) (*tagList, error) {
	var ret tagList
	if err := json.Unmarshal(tl, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

//
func decodeJSONAuth(authBase64 string) string {

	if authBase64 == "" {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(authBase64)
	if log.Error(err) {
		return ""
	}

	var ret creds
	if err := json.Unmarshal([]byte(decoded), &ret); log.Error(err) {
		return ""
	}

	return fmt.Sprintf("%s:%s", ret.Username, ret.Password)
}

//
func withoutPort(repo string) string {
	ix := strings.Index(repo, ":")
	if ix == -1 {
		return repo
	}
	return repo[:ix]
}
