//   Copyright 2016 Wercker Holding BV
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
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wercker/wercker/util"
)

type ArtifactSuite struct {
	*util.TestSuite
}

func TestArtifactSuite(t *testing.T) {
	suiteTester := &ArtifactSuite{&util.TestSuite{}}
	suite.Run(t, suiteTester)
}

func (s *ArtifactSuite) TestDockerFileCollectorSingle() {
	client := DockerOrSkip(s.T())

	container, err := TempBusybox(client)
	s.Nil(err)
	defer container.Remove()

	dfc := NewDockerFileCollector(client, container.ID)

	archive, errs := dfc.Collect("/etc/alpine-release")
	var b bytes.Buffer

	select {
	case err := <-archive.SingleBytes("alpine-release", &b):
		s.Nil(err)
	case err := <-errs:
		s.Nil(err)
		s.T().FailNow()
	}

	s.Equal("3.1.4\n", b.String())
}

func (s *ArtifactSuite) TestDockerFileCollectorSingleNotFound() {
	client := DockerOrSkip(s.T())

	container, err := TempBusybox(client)
	s.Nil(err)
	defer container.Remove()

	dfc := NewDockerFileCollector(client, container.ID)

	// Fail first from docker client
	archive, errs := dfc.Collect("/notfound/file")
	var b bytes.Buffer
	select {
	case <-archive.SingleBytes("file", &b):
		s.T().FailNow()
	case err := <-errs:
		s.Equal(err, util.ErrEmptyTarball)
	}

	// Or from archive
	archive, errs = dfc.Collect("/etc/issue")
	var b2 bytes.Buffer
	select {
	case err := <-archive.SingleBytes("notfound", &b2):
		s.Equal(err, util.ErrEmptyTarball)
	case <-errs:
		s.T().FailNow()
	}
}

func (s *ArtifactSuite) TestDockerFileCollectorMulti() {
	client := DockerOrSkip(s.T())

	container, err := TempBusybox(client)
	s.Nil(err)
	defer container.Remove()

	dfc := NewDockerFileCollector(client, container.ID)

	archive, errs := dfc.Collect("/etc/apk")
	var b bytes.Buffer

	select {
	case err := <-archive.SingleBytes("apk/arch", &b):
		s.Nil(err)
	case <-errs:
		s.T().FailNow()
	}

	check := "x86_64\n"
	s.Equal(check, b.String())
}

func (s *ArtifactSuite) TestDockerFileCollectorMultiEmptyTarball() {
	client := DockerOrSkip(s.T())

	container, err := TempBusybox(client)
	s.Nil(err)
	defer container.Remove()

	dfc := NewDockerFileCollector(client, container.ID)

	archive, errs := dfc.Collect("/var/tmp")

	tmp, err := ioutil.TempDir("", "test-")
	s.Nil(err)
	defer os.RemoveAll(tmp)

	select {
	case err := <-archive.Multi("tmp", tmp, maxArtifactSize):
		s.Equal(err, util.ErrEmptyTarball)
	case <-errs:
		s.FailNow()
	}
}

func (s *ArtifactSuite) TestDockerFileCollectorMultiNotFound() {
	client := DockerOrSkip(s.T())

	container, err := TempBusybox(client)
	s.Nil(err)
	defer container.Remove()

	dfc := NewDockerFileCollector(client, container.ID)

	archive, errs := dfc.Collect("/notfound")

	tmp, err := ioutil.TempDir("", "test-")
	s.Nil(err)
	defer os.RemoveAll(tmp)

	select {
	case <-archive.Multi("default", tmp, maxArtifactSize):
		s.FailNow()
	case err := <-errs:
		s.Equal(err, util.ErrEmptyTarball)
	}
}
