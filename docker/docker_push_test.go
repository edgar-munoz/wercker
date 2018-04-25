package dockerlocal

import (
	"encoding/json"
	"net/url"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/suite"
	"github.com/wercker/docker-check-access"
	"github.com/wercker/wercker/auth"
	"github.com/wercker/wercker/core"
	"github.com/wercker/wercker/util"
)

const (
	RepoUnauthorized         = "fail_me/unauthorized"
	ErrorMessageUnauthorized = "unauthorized: incorrect username or password"
	RepoUnconfirmedPush      = "fail_me/unconfirmed"
	ErrorMessageUnconfirmed  = NoPushConfirmationInStatus
	RepoSuccessful           = "pass_me/successful"
	RepoSuccessfulImageSHA   = "9987d147c777f2fff2ec17d557304b20da65bc9e270f945623ab04de59ca4f2c"
	RepoSuccessfulImageSize  = 121
	RepoSuccessfulImageTag   = "stage"
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

func (s *PushSuite) TestInferRegistryAndRepository() {
	testWerckerRegistry, _ := url.Parse("https://test.wcr.io/v2")
	repoTests := []struct {
		registry           string
		repository         string
		expectedRegistry   string
		expectedRepository string
	}{
		{"", "appowner/appname", "", "appowner/appname"},
		{"", "", testWerckerRegistry.String(), testWerckerRegistry.Host + "/appowner/appname"},
		{"", "someregistry.com/appowner/appname", "https://someregistry.com/v2/", "someregistry.com/appowner/appname"},
		{"", "appOWNER/appname", "", "appowner/appname"},
		{"https://someregistry.com", "appowner/appname", "https://someregistry.com", "someregistry.com/appowner/appname"},
		{"https://someregistry.com/v1", "appowner/appname", "https://someregistry.com/v1", "someregistry.com/appowner/appname"},
		{"https://someregistry.com/v2", "appowner/appname", "https://someregistry.com/v2", "someregistry.com/appowner/appname"},
		{"https://someregistry.com", "someotherregistry.com/appowner/appname", "https://someotherregistry.com/v2/", "someotherregistry.com/appowner/appname"},
		{"https://someregistry.com", "appowner/appname", "https://someregistry.com", "someregistry.com/appowner/appname"},
	}

	for _, tt := range repoTests {
		options := &core.PipelineOptions{
			ApplicationOwnerName:     "appowner",
			ApplicationName:          "appname",
			WerckerContainerRegistry: testWerckerRegistry,
		}
		opts := dockerauth.CheckAccessOptions{
			Registry: tt.registry,
		}
		repo, registry, _ := InferRegistryAndRepository(tt.repository, opts.Registry, options)
		opts.Registry = registry
		s.Equal(tt.expectedRegistry, opts.Registry, "%q, wants %q", opts.Registry, tt.expectedRegistry)
		s.Equal(tt.expectedRepository, repo, "%q, wants %q", repo, tt.expectedRepository)
	}

}

//TestTagAndPushCorretStatusReportingForUnauthorizedFailedPush - Tests a scenario when
// push will fail due to an unauthorized access to a repo
func (s *PushSuite) TestTagAndPushCorretStatusReportingForUnauthorizedFailedPush() {
	stepData := make(map[string]string)
	stepData["username"] = "user"
	stepData["password"] = "pass"
	stepData["repository"] = RepoUnauthorized
	stepData["registry"] = "https://quay.io"
	stepData["tag"] = "test"

	exitCode, error := executePushStep(stepData)
	s.NotEqual(exitCode, 0)
	s.NotNil(error)
	s.Contains(error.Error(), ErrorMessageUnauthorized)
}

//TestTagAndPushCorretStatusReportingForUnconfirmedFailedPush - Tests a scenario when
// push will not return any failure message as such and also will not be successful!
func (s *PushSuite) TestTagAndPushCorretStatusReportingForUnconfirmedFailedPush() {
	stepData := make(map[string]string)
	stepData["username"] = "user"
	stepData["password"] = "pass"
	stepData["repository"] = RepoUnconfirmedPush
	stepData["registry"] = "https://quay.io"
	stepData["tag"] = "test"

	exitCode, error := executePushStep(stepData)
	s.NotEqual(exitCode, 0)
	s.NotNil(error)
	s.Contains(error.Error(), ErrorMessageUnconfirmed)
}

//TestTagAndPushCorretStatusReportingForSuccessfulPush - Tests the scenario when a push is
// successful and tagAndPush will only return success if the status message from docker will
// contain digest and tag of pushed container
func (s *PushSuite) TestTagAndPushCorretStatusReportingForSuccessfulPush() {
	stepData := make(map[string]string)
	stepData["username"] = "user"
	stepData["password"] = "pass"
	stepData["repository"] = RepoSuccessful
	stepData["registry"] = "https://quay.io"
	stepData["tag"] = RepoSuccessfulImageTag

	exitCode, error := executePushStep(stepData)
	s.Equal(exitCode, 0)
	s.Nil(error)
}

//executePushStep - Prepares stepcConfig for docker-push step from input stepData
// and invokes tagAndPush
func executePushStep(stepData map[string]string) (int, error) {
	config := &core.StepConfig{
		ID:   "internal/docker-push",
		Data: stepData,
	}
	options := &core.PipelineOptions{}
	step, _ := NewDockerPushStep(config, options, nil)
	step.configure(&util.Environment{})
	step.dockerOptions = &Options{}
	step.authenticator = &auth.DockerAuth{}
	step.logger = util.NewLogger().WithFields(util.LogFields{
		"Logger": "Test",
	})
	mockEmittor := core.NewNormalizedEmitter()
	mockDockerClient := &DockerClient{}
	return step.tagAndPush("test", mockEmittor, mockDockerClient)
}

//RemoveImage - Mocks DockerClient.TagImage
func (c *DockerClient) TagImage(name string, opts docker.TagImageOptions) error {
	return nil
}

//RemoveImage - Mocks DockerClient.RemoveImage
func (c *DockerClient) RemoveImage(name string) error {
	return nil
}

//PushImage - Mocks DockerClient.PushImage - writes status messages to OutputStream based on repository name
func (c *DockerClient) PushImage(opts docker.PushImageOptions, auth docker.AuthConfiguration) error {
	status := &PushStatus{}
	if opts.Name == RepoUnauthorized {
		status.Error = ErrorMessageUnauthorized
		status.ErrorDetail = &PushStatusErrorDetail{Message: ErrorMessageUnauthorized}
	} else if opts.Name == RepoUnconfirmedPush {
		status.Status = "Waiting"
		status.ID = "61c06e07759a"
		status.ProgressDetail = &PushStatusProgressDetail{}
	} else if opts.Name == RepoSuccessful {
		status.Aux = &PushStatusAux{Digest: RepoSuccessfulImageSHA, Size: RepoSuccessfulImageSize, Tag: RepoSuccessfulImageTag}
	}
	jsonData, _ := json.Marshal(status)
	opts.OutputStream.Write(jsonData)
	return nil
}

//TestInferRegistryAndRepositoryInvalidInputs validates that poper errors
// are being returned by InferRegistryAndRepository menthod when invalid
// inputs are provided for repository and registry
func (s *PushSuite) TestInferRegistryAndRepositoryInvalidInputs() {
	testWerckerRegistry, _ := url.Parse("https://test.wcr.io/v2")
	repoTests := []struct {
		registry           string
		repository         string
		expectedRegistry   string
		expectedRepository string
		errorMessage       string
	}{
		{"invalidregistry", "appowner/appname", "", "", "not a valid registry URL"},
		{"https://someregistry.com", "appowner//appname", "", "", "not a valid repository"},
		{"https://someregistry.com", "https://someregistry.com/appowner/appname", "", "", "not a valid repository"},
	}

	for _, tt := range repoTests {
		options := &core.PipelineOptions{
			ApplicationOwnerName:     "appowner",
			ApplicationName:          "appname",
			WerckerContainerRegistry: testWerckerRegistry,
		}
		opts := dockerauth.CheckAccessOptions{
			Registry: tt.registry,
		}
		repo, registry, err := InferRegistryAndRepository(tt.repository, opts.Registry, options)
		opts.Registry = registry
		s.Error(err)
		s.Contains(err.Error(), tt.errorMessage)
		s.Equal(tt.expectedRegistry, opts.Registry, "%q, wants %q", opts.Registry, tt.expectedRegistry)
		s.Equal(tt.expectedRepository, repo, "%q, wants %q", repo, tt.expectedRepository)
	}

}
