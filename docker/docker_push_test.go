package dockerlocal

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wercker/wercker/auth"
	"github.com/wercker/wercker/core"
	"github.com/wercker/wercker/util"
)

type PushSuite struct {
	*util.TestSuite
}

func TestPushSuite(t *testing.T) {
	suiteTester := &PushSuite{&util.TestSuite{}}
	suite.Run(t, suiteTester)
}

//TestEmptyPush tests if you juse did something like this
// - internal/docker-push
// it should fill in a tag of the git branch and commit
// check to see if its pushing up to the right registry or not
func (s *PushSuite) TestEmptyPush() {
	config := &core.StepConfig{
		ID:   "internal/docker-push",
		Data: map[string]string{},
	}
	options := &core.PipelineOptions{
		GitOptions: &core.GitOptions{
			GitBranch: "master",
			GitCommit: "s4k2r0d6a9b",
		},
		ApplicationID:            "1000001",
		ApplicationName:          "myproject",
		ApplicationOwnerName:     "wercker",
		WerckerContainerRegistry: &url.URL{Scheme: "https", Host: "wcr.io", Path: "/v2/"},
		GlobalOptions: &core.GlobalOptions{
			AuthToken: "su69persec420uret0k3n",
		},
	}
	step, _ := NewDockerPushStep(config, options, nil)
	step.InitEnv(nil)
	repositoryName := step.authenticator.Repository(step.repository)
	s.Equal("wcr.io/wercker/myproject", repositoryName)
	tags := step.buildTags()
	s.Equal([]string{"latest", "master-s4k2r0d6a9b"}, tags)
}

func (s *PushSuite) TestInferRegistry() {
	testWerckerRegistry, _ := url.Parse("https://test.wcr.io/v2")
	repoTests := []struct {
		registry           string
		repository         string
		expectedRegistry   string
		expectedRepository string
	}{
		{"", "appowner/appname", "", "appowner/appname"},
		{"", "", testWerckerRegistry.String() + "/", testWerckerRegistry.Host + "/appowner/appname"},
		{"", "someregistry.com/appowner/appname", "https://someregistry.com/v2/", "someregistry.com/appowner/appname"},
		{"", "appOWNER/appname", "", "appowner/appname"},
	}

	for _, tt := range repoTests {
		options := &core.PipelineOptions{
			ApplicationOwnerName:     "appowner",
			ApplicationName:          "appname",
			WerckerContainerRegistry: testWerckerRegistry,
		}
		repo, opts := InferRegistry(tt.repository, dockerauth.CheckAccessOptions{
			Registry: tt.registry,
		}, options)
		s.Equal(tt.expectedRegistry, opts.Registry, "%q, wants %q", opts.Registry, tt.expectedRegistry)
		s.Equal(tt.expectedRepository, repo, "%q, wants %q", repo, tt.expectedRepository)
	}
}
