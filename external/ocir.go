// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package external

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	os "os"
	"time"

	"github.com/fsouza/go-dockerclient"
)

// LatestImage for output
type LatestImage struct {
	ImageName string
	Created   time.Time
}

// Request token for authenticated request
type requestToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
}

// CurrentImage item
type CurrentImage struct {
	URL   string `json:"url"`
	Start string `json:"start"`
	Limit int    `json:"limit"`
}

// RemoteImage item
type RemoteImage struct {
	Tag       string `json:"tag"`
	Digest    string `json:"digest"`
	Timestamp string `json:"timestamp"`
}

// List wrapper for response payload
type listWrapper struct {
	Current CurrentImage  `json:"current"`
	Imgs    []RemoteImage `json:"imgs"`
}

func (cp *RunnerParams) getRemoteImage() (*LatestImage, error) {

	resultToken, err := cp.getBearerToken()

	url := "https://iad.ocir.io/20180419/docker/images/odx-pipelines/wercker/wercker-runner"

	var client http.Client

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+resultToken)
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	var latestImageName string
	var latestImageTime time.Time

	basis := "iad.ocir.io/odx-pipelines/wercker/wercker-runner"

	if resp.StatusCode == 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		bodyString := string(bodyBytes)
		theWrapper := listWrapper{}
		json.Unmarshal([]byte(bodyString), &theWrapper)

		for _, imageItem := range theWrapper.Imgs {
			tm, err := time.Parse(time.RFC3339, imageItem.Timestamp)
			if err != nil {
				cp.Logger.Error(err)
				continue
			}

			if tm.After(latestImageTime) {
				latestImageTime = tm
				latestImageName = fmt.Sprintf("%s:%s", basis, imageItem.Tag)
			}
		}
	}
	return &LatestImage{
		ImageName: latestImageName,
		Created:   latestImageTime,
	}, nil
}

func (cp *RunnerParams) getBearerToken() (string, error) {

	username := os.Getenv("WERCKER_OCIR_USERNAME")
	password := os.Getenv("WERCKER_OCIR_PASSWORD")

	if username == "" || password == "" {
		return "", nil
	}

	auth := username + ":" + password
	tokenAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	url := "https://iad.ocir.io/20180419/docker/token"

	var client http.Client

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Basic"+tokenAuth)
	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var resultToken string

	if resp.StatusCode == 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		bodyString := string(bodyBytes)
		theToken := requestToken{}
		json.Unmarshal([]byte(bodyString), &theToken)
		resultToken = theToken.Token
	}
	return resultToken, nil
}

// Pull the newer image from OCIR. The older image is left so if there
// is a problem with the newer image it can be removed from the local
// repository as a manual rollback.
func (cp *RunnerParams) pullNewerImage(imageName string) error {

	username := os.Getenv("WERCKER_OCIR_USERNAME")
	password := os.Getenv("WERCKER_OCIR_PASSWORD")

	opts := docker.PullImageOptions{
		Repository: "iad.ocir.io",
		Registry:   "odx-pipelines",
		Tag:        imageName,
	}
	auth := docker.AuthConfiguration{
		Username: username,
		Password: password,
	}
	err := cp.client.PullImage(opts, auth)

	if err != nil {
		message := fmt.Sprintf("Failed to update external runner image: %s", err)
		cp.Logger.Error(message)
	} else {
		message := fmt.Sprintf("Pulled newer external runner image: %s", imageName)
		cp.Logger.Info(message)
	}
	return err
}
