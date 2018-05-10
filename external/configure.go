// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package external

import (
	"fmt"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	context "golang.org/x/net/context"
)

// Get the Docker client
func (cp *RunnerParams) getDockerClient() error {
	context.Background()
	cli, err := docker.NewClient(cp.DockerEndpoint)
	if err != nil {
		cp.Logger.Fatal(fmt.Sprintf("unable to create the Docker client: %s", err))
		return err
	}
	cp.client = cli
	return nil
}

// Describe the local image and return the Image structure
func (cp *RunnerParams) getLocalImage() (*docker.Image, error) {

	opts := docker.ListImagesOptions{
		All: true,
	}

	// Find the image containing 'wercker/wercker-runner:external-runner"
	images, err := cp.client.ListImages(opts)
	if err != nil {
		return nil, err
	}

	// Dynamically figure out the image name based on a known static string embedded in
	// the repository tag. This allows different repository prefixs and version information
	// in the tail end of the tag. When more than one instance is found then take the
	// most recent image.

	var imageName string
	var latest int64 = 0
	for _, image := range images {
		for _, slice := range image.RepoTags {
			if strings.Contains(slice, "wercker/wercker-runner:external-runner") {
				if latest < image.Created {
					latest = image.Created
					imageName = slice
					break
				}
			}
		}
	}
	if imageName == "" {
		return nil, nil
	}
	cp.ImageName = imageName

	image, err := cp.client.InspectImage(cp.ImageName)
	if err != nil {
		return nil, err
	}
	return image, err
}

// Check the external runner images between local and remote repositories.
// If local exists but remote does not then do nothing
// If local exists and is the same as the remote then do nothing
// If local is older than remote then give user the option to download the remote
// If neither exists then fail immediately
func (cp *RunnerParams) CheckRegistryImages() error {

	err := cp.getDockerClient()
	if err != nil {
		cp.Logger.Fatal(err)
	}

	// Get the local image for the runner
	localImage, err := cp.getLocalImage()

	imageList, err := cp.getRemoteImages()
	for _, remoteImage := range imageList {
		cp.Logger.Infoln(fmt.Sprintf("%s %s", remoteImage.tag, remoteImage.timestamp))
	}

	if err != nil {
		cp.Logger.Fatal(err)
	}
	if localImage == nil {
		cp.Logger.Fatal("No docker external runner image exists in the local repository.")
	}
	message := fmt.Sprintf("Docker image %s is up-to-date, created: %s", cp.ImageName, localImage.Created)
	cp.Logger.Print(message)
	return nil
}
