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

package rdd

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/docker/docker/client"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpcmw "github.com/mwitkow/go-grpc-middleware"
	"github.com/wercker/pkg/log"
	rddpb "github.com/wercker/wercker/rdd/rddpb"
	"google.golang.org/grpc"
	"gopkg.in/urfave/cli.v1"
)

//RDD - encapsulate all RDD access
type RDD struct {
	rddServiceEndpoint  string
	rddProvisionTimeout int64
	runID               string
	ctx                 context.Context
	rddClient           rddpb.RddClient
	rddDetails          *rddDetails
}

type rddDetails struct {
	rddURI                string
	rddProvisionRequestID string
}

//Init - initialize a RDD construct
func Init(ctx context.Context, rddServiceEndpoint string, rddProvisionTimeout int64, runID string) (*RDD, error) {
	log.Debug("Connecting to rdd service")

	rddInterceptors := []grpc.UnaryClientInterceptor{
		grpc_prometheus.UnaryClientInterceptor,
	}

	rddConn, err := grpc.Dial(rddServiceEndpoint, grpc.WithInsecure(), grpc.WithUnaryInterceptor(grpcmw.ChainUnaryClient(rddInterceptors...)))
	if err != nil {
		errMsg := fmt.Sprintf("Failed to dial rdd service at %s for runID %s, Error: %s", rddServiceEndpoint, runID, err.Error())
		log.WithField("rddServiceEndpoint", rddServiceEndpoint).
			WithError(err).
			Error(errMsg)
		return nil, cli.NewExitError(errMsg, 1)
	}

	rddClient := rddpb.NewRddClient(rddConn)

	rdd := &RDD{rddServiceEndpoint: rddServiceEndpoint,
		rddProvisionTimeout: rddProvisionTimeout,
		runID:               runID,
		ctx:                 ctx,
		rddClient:           rddClient}

	return rdd, nil
}

//Get - Invokes RDD Service to get remote docker daemon URL
func (rdd *RDD) Get() (string, error) {
	rddProvRequest := &rddpb.RDDProvisionRequest{RunID: rdd.runID}
	rddProvResponse, err := rdd.rddClient.Provision(rdd.ctx, rddProvRequest)

	if err != nil {
		errMsg := fmt.Sprintf("Error invoking Provision() from rdd service at %s for runID %s, Error: %s", rdd.rddServiceEndpoint, rdd.runID, err.Error())
		log.Error(errMsg)
		return "", cli.NewExitError(errMsg, 1)
	}

	rddResponseID := rddProvResponse.GetId()
	if rddResponseID == "" {
		errMsg := fmt.Sprintf("Invalid response by Provision() from rdd service at %s for runID %s, ResponseID is empty.", rdd.rddServiceEndpoint, rdd.runID)
		log.Error(errMsg)
		return "", cli.NewExitError(errMsg, 1)
	}

	timeoutThresholdInMinutes, err := time.ParseDuration(fmt.Sprintf("%dm", rdd.rddProvisionTimeout))
	if err != nil {
		log.Error("Error parsing timeout value from input rdd-provision-timeout of %d, Error: %s. Default value of 5m will be used.", rdd.rddProvisionTimeout, err.Error())
		timeoutThresholdInMinutes = 5 * time.Minute
	}
	timeout := time.After(timeoutThresholdInMinutes)
	tick := time.Tick(5 * time.Second)

	for {
		select {

		case <-timeout:
			errMsg := fmt.Sprintf("RDD provisioning timed out from rdd service at %s for runID %s after %d minutes.", rdd.rddServiceEndpoint, rdd.runID, 5)
			log.Error(errMsg)
			return "", cli.NewExitError(errMsg, 1)

		case <-tick:
			rddStatusRequest := &rddpb.RDDStatusRequest{Id: rddResponseID}
			rddStatusResponse, err := rdd.rddClient.GetStatus(rdd.ctx, rddStatusRequest)
			if err != nil {
				errMsg := fmt.Sprintf("Error invoking GetStatus() from rdd service at %s for runID %s, Error: %s. Retrying...", rdd.rddServiceEndpoint, rdd.runID, err.Error())
				log.Error(errMsg)
				continue
			}
			currentRDDState := rddStatusResponse.GetState()
			if currentRDDState == rddpb.DaemonState_error {
				errMsg := fmt.Sprintf("Error provisioning RDD from rdd service at %s for runID %s. Aborting.", rdd.rddServiceEndpoint, rdd.runID)
				log.Error(errMsg)
				return "", cli.NewExitError(errMsg, 1)
			}
			if currentRDDState == rddpb.DaemonState_provisioned {
				rddURI := rddStatusResponse.URL
				if rddURI == "" {
					errMsg := fmt.Sprintf("Invalid RDD uri returned from rdd service at %s for runID %s. Aborting.", rdd.rddServiceEndpoint, rdd.runID)
					log.Error(errMsg)
					return "", cli.NewExitError(errMsg, 1)
				}
				rdd.rddDetails = &rddDetails{rddProvisionRequestID: rddResponseID, rddURI: rddURI}
				err := rdd.verify()
				if err != nil {
					return "", err
				}
				return rddURI, nil

			} else {
				log.Info(fmt.Sprintf("runID: %s, RDD Service URI: %s, RDD Provisioning status: %s", rdd.runID, rdd.rddServiceEndpoint, rddStatusResponse.GetState().String()))
			}
		}
	}
}

//Delete - removes a RDD
func (rdd *RDD) Delete() {
	rddDeProvRequest := &rddpb.RDDDeprovisionRequest{Id: rdd.rddDetails.rddProvisionRequestID}
	_, err := rdd.rddClient.Deprovision(rdd.ctx, rddDeProvRequest)
	if err != nil {
		errMsg := fmt.Sprintf("Error invoking Deprovision() from rdd service at %s for runID %s, Error: %s. Ignoring", rdd.rddServiceEndpoint, rdd.runID, err.Error())
		log.Error(errMsg)
	}
}

func (rdd *RDD) verify() error {
	rddURI := rdd.rddDetails.rddURI
	dockerClient, err := client.NewClientWithOpts(client.WithHost(rddURI))
	if err != nil {
		return fmt.Errorf(`Unable to create a docker client with RDD URI: %s, Error: %s
		`, rddURI, err.Error())
	}
	version, err := dockerClient.ServerVersion(rdd.ctx)
	if err != nil {
		if reflect.TypeOf(err).String() == "client.errConnectionFailed" {
			return fmt.Errorf(`RDD URL %s does not point to a working Docker environment or wercker can't connect to the Docker endpoint, Error: 
			%s`, rddURI, err.Error())
		}
		return err
	}
	if version.Version == "" {
		return fmt.Errorf(`Unidentifiable docker version at RDD URI: %s
			`, rddURI)
	}
	log.Info(fmt.Sprintf("Successfully connected to RDD at %s, Docker version: %s, Docker API version: %s", rddURI, version.Version, version.APIVersion))
	return nil
}
