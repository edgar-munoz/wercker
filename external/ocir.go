// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package external

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	os "os"
)

// Request token for authenticated request
type requestToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
}

// Current image item
type currentImage struct {
	url   string
	start string
	limit int
}

// Remote image item
type RemoteImage struct {
	tag       string
	digest    string
	timestamp string
}

// List wrapper for response payload
type listWrapper struct {
	current currentImage
	imgs    []RemoteImage
}

func (cp *RunnerParams) getRemoteImages() ([]RemoteImage, error) {

	imageList := []RemoteImage{}

	resultToken, err := cp.getBearerToken()

	url := "https://iad.ocir.io/v2/odx-pipelines/wercker/wercker-runner/tags/list"

	var client http.Client

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+resultToken)
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		bodyString := string(bodyBytes)
		theWrapper := listWrapper{}
		json.Unmarshal([]byte(bodyString), &theWrapper)

		for _, imageItem := range theWrapper.imgs {
			imageList = append(imageList, imageItem)
		}
	}
	return imageList, nil
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
