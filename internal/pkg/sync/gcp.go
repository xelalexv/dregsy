package sync

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

const url = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"

//
type Response struct {
	AccessToken string         `json:"access_token"`
	ExpiresIn   *time.Duration `json:"expires_in"`
	TokenType   string         `json:"token_type"`
}

//
func (l *Location) isGCR() bool {
	url := strings.Split(l.Registry, ".")
	gcr := url[len(url)-2] == "gcr" && url[len(url)-1] == "io"
	return gcr
}

//
func isGCE() bool {
	resp, err := http.Head(url)
	if err != nil {
		return false
	}

	if _, ok := resp.Header["Metadata-Flavor"]; ok {
		return resp.Header.Get("Metadata-Flavor") == "Google"
	}

	return false
}

//
func tokenFromCreds() (string, time.Time, error) {

	b, err := ioutil.ReadFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	if err != nil {
		return "", time.Time{}, err
	}
	var c = struct {
		Email      string `json:"client_email"`
		PrivateKey string `json:"private_key"`
	}{}

	json.Unmarshal(b, &c)

	config := &jwt.Config{
		Email:      c.Email,
		PrivateKey: []byte(c.PrivateKey),
		Scopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_write",
		},
		TokenURL: google.JWTTokenURL,
	}

	token, err := config.TokenSource(oauth2.NoContext).Token()
	if err != nil {
		return "", time.Time{}, err
	}

	return token.AccessToken, token.Expiry, nil
}

//
func tokenFromMetadata() (string, time.Time, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", time.Time{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Metadata-Flavor", "Google")

	client := &http.Client{}

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	var tokenResponse Response

	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	if err != nil {
		return "", time.Time{}, err
	}

	expiryTime := start.Add(time.Second * *tokenResponse.ExpiresIn)

	return tokenResponse.AccessToken, expiryTime, nil
}
