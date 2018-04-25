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
	"path"

	"github.com/docker/docker/client"
)

// NewOfficialDockerClient uses the official docker client to create a Client struct
// which can be used to perform operations against a docker server
func NewOfficialDockerClient(options *Options) (*client.Client, error) {
	var dockerClient *client.Client
	var err error
	if options.TLSVerify == "1" {
		// We're using TLS, let's locate our certs and such
		// boot2docker puts its certs at...
		dockerCertPath := options.CertPath
		// TODO: maybe fast-fail if these don't exist?
		cert := path.Join(dockerCertPath, fmt.Sprintf("cert.pem"))
		ca := path.Join(dockerCertPath, fmt.Sprintf("ca.pem"))
		key := path.Join(dockerCertPath, fmt.Sprintf("key.pem"))
		dockerClient, err = client.NewClientWithOpts(client.WithHost(options.Host), client.WithTLSClientConfig(ca, cert, key), client.WithVersion("1.24"))
	} else {
		dockerClient, err = client.NewClientWithOpts(client.WithHost(options.Host), client.WithVersion("1.24"))
	}
	if err != nil {
		return nil, err
	}
	return dockerClient, nil
}
