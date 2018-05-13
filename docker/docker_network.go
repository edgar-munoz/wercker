//   Copyright © 2018, Oracle and/or its affiliates.  All rights reserved.
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package dockerlocal

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	shortid "github.com/SKAhack/go-shortid"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/wercker/wercker/util"
)

// GetDockerNetworkName returns docker network name of docker network.
// If docker network does not exist it creates one and return its name.
func (b *DockerBox) GetDockerNetworkName() (string, error) {
	dockerNetworkName := b.dockerOptions.NetworkName
	if dockerNetworkName == "" {
		if b.options.DockerNetworkName == "" {
			preparedDockerNetworkName, err := b.prepareDockerNetworkName()
			if err != nil {
				return "", err
			}

			b.options.DockerNetworkName = preparedDockerNetworkName
			_, err = b.createDockerNetwork(b.options.DockerNetworkName)
			if err != nil {
				b.logger.Errorln("Error while creating network", err)
				return "", err
			}
		}
		return b.options.DockerNetworkName, nil
	}
	client := b.client
	_, err := client.NetworkInfo(dockerNetworkName)
	if err != nil {
		b.logger.Errorln("Network does not exist", err)
		return "", err
	}
	return dockerNetworkName, nil
}

// CleanDockerNetwork remove docker network if created for this pipeline.
func (b *DockerBox) CleanDockerNetwork() error {
	dockerNetworkName := b.dockerOptions.NetworkName
	client := b.client
	if dockerNetworkName == "" {
		dockerNetworkName = b.options.DockerNetworkName
		if dockerNetworkName != "" {
			dockerNetwork, err := client.NetworkInfo(dockerNetworkName)
			if err != nil {
				b.logger.Errorln("Unable to get network Info", err)
				return err
			}
			for k := range dockerNetwork.Containers {
				err = client.DisconnectNetwork(dockerNetwork.ID, docker.NetworkConnectionOptions{
					Container: k,
					Force:     true,
				})
				if err != nil {
					b.logger.Errorln("Error while disconnecting container from network", err)
					return err
				}
			}
			b.logger.WithFields(util.LogFields{
				"Name": dockerNetworkName,
			}).Debugln("Removing docker network ", dockerNetworkName)
			err = client.RemoveNetwork(dockerNetworkName)
			if err != nil {
				b.logger.Errorln("Error while removing docker network", err)
				return err
			}
		} else {
			b.logger.Debugln("Network does not exist")
		}
	} else {
		b.logger.Debugln("Custom netork")
	}
	return nil
}

// Create docker network
func (b *DockerBox) createDockerNetwork(dockerNetworkName string) (*docker.Network, error) {
	b.logger.Debugln("Creating docker network")
	client := b.client
	networkOptions := map[string]interface{}{
		"com.docker.network.bridge.enable_ip_masquerade": "true",
	}
	b.logger.WithFields(util.LogFields{
		"Name": dockerNetworkName,
	}).Debugln("Creating docker network :", dockerNetworkName)
	return client.CreateNetwork(docker.CreateNetworkOptions{
		Name:           dockerNetworkName,
		CheckDuplicate: true,
		Options:        networkOptions,
	})
}

// Prepares and return DockerEnvironment variables list.
// For each service In case of docker links, docker creates some environment variables and inject them to box container.
// Since docker links is replaced by docker network, these environment variables needs to be created manually.
// Below environment variables created and injected to box container.
// 01) <container name>_PORT_<port>_<protocol>_ADDR  - variable contains the IP Address.
// 02) <container name>_PORT_<port>_<protocol>_PORT - variable contains just the port number.
// 03) <container name>_PORT_<port>_<protocol>_PROTO - variable contains just the protocol.
// 04) <container name>_ENV_<name> - Docker also exposes each Docker originated environment variable from the source container as an environment variable in the target.
// 05) <container name>_PORT - variable contains the URL of the source container’s first exposed port. The ‘first’ port is defined as the exposed port with the lowest number.
// 06) <container name>_NAME - variable is set for each service specified in wercker.yml.
func (b *DockerBox) prepareSvcDockerEnvVar(env *util.Environment) ([]string, error) {
	serviceEnv := []string{}
	client := b.client
	for _, service := range b.services {
		serviceName := strings.Replace(service.GetServiceAlias(), "-", "_", -1)
		if containerID := service.GetID(); containerID != "" {
			container, err := client.InspectContainer(containerID)
			if err != nil {
				b.logger.Error("Error while inspecting container", err)
				return nil, err
			}
			ns := container.NetworkSettings
			var serviceIPAddress string
			for _, v := range ns.Networks {
				serviceIPAddress = v.IPAddress
				break
			}
			serviceEnv = append(serviceEnv, fmt.Sprintf("%s_NAME=/%s/%s", strings.ToUpper(serviceName), b.getContainerName(), serviceName))
			lowestPort := math.MaxInt32
			var protLowestPort string
			for k := range container.Config.ExposedPorts {
				exposedPort := strings.Split(string(k), "/") //exposedPort[0]=portNum and exposedPort[1]=protocal(tcp/udp)
				x, err := strconv.Atoi(exposedPort[0])
				if err != nil {
					b.logger.Error("Unable to convert string port to integer", err)
					return nil, err
				}
				if lowestPort > x {
					lowestPort = x
					protLowestPort = exposedPort[1]
				}
				dockerEnvPrefix := fmt.Sprintf("%s_PORT_%s_%s", strings.ToUpper(serviceName), exposedPort[0], strings.ToUpper(exposedPort[1]))
				serviceEnv = append(serviceEnv, fmt.Sprintf("%s=%s://%s:%s", dockerEnvPrefix, exposedPort[1], serviceIPAddress, exposedPort[0]))
				serviceEnv = append(serviceEnv, fmt.Sprintf("%s_ADDR=%s", dockerEnvPrefix, serviceIPAddress))
				serviceEnv = append(serviceEnv, fmt.Sprintf("%s_PORT=%s", dockerEnvPrefix, exposedPort[0]))
				serviceEnv = append(serviceEnv, fmt.Sprintf("%s_PROTO=%s", dockerEnvPrefix, exposedPort[1]))
			}
			if protLowestPort != "" {
				serviceEnv = append(serviceEnv, fmt.Sprintf("%s_PORT=%s://%s:%s", strings.ToUpper(serviceName), protLowestPort, serviceIPAddress, strconv.Itoa(lowestPort)))
			}
			for _, envVar := range container.Config.Env {
				serviceEnv = append(serviceEnv, fmt.Sprintf("%s_ENV_%s", strings.ToUpper(serviceName), envVar))
			}
		}
	}
	b.logger.Debug("Exposed Service Evnironment variables", serviceEnv)
	return serviceEnv, nil
}

// Generate docker network name and check if same is already in use. In case name is already in use then it regenerate it upto 3 times before throwing error.
func (b *DockerBox) prepareDockerNetworkName() (string, error) {
	generator := shortid.Generator()
	client := b.client

	for i := 0; i < 3; i++ {
		dockerNetworkName := generator.Generate()
		dockerNetworkName = "w-" + dockerNetworkName
		network, _ := client.NetworkInfo(dockerNetworkName)
		if network != nil {
			b.logger.Debugln("Network name exist, retrying...")
		} else {
			return dockerNetworkName, nil
		}
	}
	err := fmt.Errorf("Unable to prepare unique network name")
	b.logger.Errorln(err)
	return "", err
}
