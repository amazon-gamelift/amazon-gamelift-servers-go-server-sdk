/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/message"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/request"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model/result"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server/internal/security"
)

var localRnd *rand.Rand

// store in a variable in order to unit test
var exitFunc = os.Exit

const ActivateServerProcessRequestTimeoutInSeconds = time.Duration(6) * time.Second

func init() {
	//nolint:gosec // Use a weak random generator is enough in this case
	localRnd = rand.New(rand.NewSource(time.Now().Unix()))
}

type iGameLiftServerState interface {
	processReady(*ProcessParameters) error
	processEnding() error
	activateGameSession() error
	updatePlayerSessionCreationPolicy(*model.PlayerSessionCreationPolicy) error
	getGameSessionID() (string, error)
	getTerminationTime() (int64, error)
	acceptPlayerSession(playerSessionID string) error
	removePlayerSession(playerSessionID string) error
	describePlayerSessions(*request.DescribePlayerSessionsRequest) (result.DescribePlayerSessionsResult, error)
	startMatchBackfill(*request.StartMatchBackfillRequest) (result.StartMatchBackfillResult, error)
	stopMatchBackfill(*request.StopMatchBackfillRequest) error
	getComputeCertificate() (result.GetComputeCertificateResult, error)
	getFleetRoleCredentials(*request.GetFleetRoleCredentialsRequest) (result.GetFleetRoleCredentialsResult, error)
	destroy() error
}

type gameLiftServerState struct {
	wsGameLift internal.IGameLiftManager
	parameters *ProcessParameters

	processID string
	hostID    string
	fleetID   string

	gameSessionID   string
	terminationTime int64

	isReadyProcess common.AtomicBool
	onManagedEC2   bool

	fleetRoleResultCache map[string]result.GetFleetRoleCredentialsResult
	mtx                  sync.Mutex

	defaultJitterIntervalMs int64
	healthCheckInterval     time.Duration
	healthCheckTimeout      time.Duration
	serviceCallTimeout      time.Duration

	shutdown chan bool
}

func (state *gameLiftServerState) init(params ServerParameters, wsGameLift internal.IGameLiftManager) error {
	state.fleetRoleResultCache = make(map[string]result.GetFleetRoleCredentialsResult)
	// processID should be initialized by caller
	params.HostID = common.GetEnvStringOrDefault(common.EnvironmentKeyHostID, params.HostID)
	params.FleetID = common.GetEnvStringOrDefault(common.EnvironmentKeyFleetID, params.FleetID)
	params.WebSocketURL = common.GetEnvStringOrDefault(common.EnvironmentKeyWebsocketURL, params.WebSocketURL)
	params.AuthToken = common.GetEnvStringOrDefault(common.EnvironmentKeyAuthToken, params.AuthToken)
	params.AwsRegion = common.GetEnvStringOrDefault(common.EnvironmentKeyAwsRegion, params.AwsRegion)
	params.AccessKey = common.GetEnvStringOrDefault(common.EnvironmentKeyAccessKey, params.AccessKey)
	params.SecretKey = common.GetEnvStringOrDefault(common.EnvironmentKeySecretKey, params.SecretKey)
	params.SessionToken = common.GetEnvStringOrDefault(common.EnvironmentKeySessionToken, params.SessionToken)

	computeType := common.GetEnvStringOrDefault(common.EnvironmentKeyComputeType, "")

	// AuthToken takes priority as the authorization strategy
	if params.AuthToken != "" {
		params.AwsRegion = ""
		params.AccessKey = ""
		params.SecretKey = ""
		params.SessionToken = ""
	}

	err := ValidateServerParameters(params, computeType)
	if err != nil {
		return err
	}

	state.processID = params.ProcessID
	state.hostID = params.HostID
	state.fleetID = params.FleetID
	state.onManagedEC2 = true
	state.defaultJitterIntervalMs = common.GetEnvDurationOrDefault(
		common.HealthcheckMaxJitter,
		common.HealthcheckMaxJitterDefault,
		lg,
	).Milliseconds()
	state.healthCheckInterval = common.GetEnvDurationOrDefault(
		common.HealthcheckInterval,
		common.HealthcheckIntervalDefault,
		lg,
	)
	state.healthCheckTimeout = common.GetEnvDurationOrDefault(
		common.HealthcheckTimeout,
		common.HealthcheckTimeoutDefault,
		lg,
	)
	state.serviceCallTimeout = common.GetEnvDurationOrDefault(
		common.ServiceCallTimeout,
		common.ServiceCallTimeoutDefault,
		lg,
	)

	var sigV4QueryParameters map[string]string
	if params.AuthToken == "" {
		if computeType == common.ComputeTypeContainer {
			awsCredentials, err := wsGameLift.FetchCredentials(computeType)
			if err != nil {
				return err
			}
			params.AccessKey = awsCredentials.AccessKey
			params.SecretKey = awsCredentials.SecretKey
			params.SessionToken = awsCredentials.SessionToken

			metadata, err := wsGameLift.FetchMetadata(computeType)
			if err != nil {
				return err
			}

			state.hostID = metadata.GetHostId()
		}
		sigV4QueryParameters = getSigV4QueryParameters(params.AwsRegion, params.AccessKey, params.SecretKey, params.SessionToken)
	}

	state.wsGameLift = wsGameLift
	err = state.wsGameLift.Connect(
		params.WebSocketURL,
		state.processID,
		state.hostID,
		state.fleetID,
		params.AuthToken,
		sigV4QueryParameters,
	)
	if err != nil {
		return common.NewGameLiftError(common.LocalConnectionFailed, "", err.Error())
	}
	return nil
}

func getSigV4QueryParameters(awsRegion, accessKey, secretKey, sessionToken string) map[string]string {
	awsCredentials := security.AwsCredentials{AccessKey: accessKey, SecretKey: secretKey, SessionToken: sessionToken}
	queryParamsToSign := map[string]string{
		common.ComputeIDKey: state.hostID,
		common.FleetIDKey:   state.fleetID,
		common.PidKey:       state.processID,
	}

	sigV4Parameters := security.SigV4Parameters{
		AwsRegion:      awsRegion,
		AwsCredentials: awsCredentials,
		QueryParams:    queryParamsToSign,
		RequestTime:    time.Now().UTC(),
	}

	sigV4QueryParameters, err := security.GenerateSigV4QueryParameters(sigV4Parameters)
	if err != nil {
		log.Fatalf("Error generating SigV4 query string: %v\n", err)
	}
	return sigV4QueryParameters
}

func (state *gameLiftServerState) processReady(params *ProcessParameters) error {
	if params == nil {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	err := ValidateProcessParameters(*params)
	if err != nil {
		return err
	}
	var res message.ResponseMessage
	state.parameters = params
	req := request.NewActivateServerProcess(
		common.SdkVersion,
		common.SdkLanguage,
		params.Port,
	)
	req.LogPaths = params.LogParameters.LogPaths

	// Add the tool name and version to the request if the environment variables are set
	sdkToolName := common.GetEnvStringOrDefault(common.EnvironmentKeySDKToolName, "")
	if sdkToolName != "" {
		req.SdkToolName = sdkToolName
	}
	toolVersion := common.GetEnvStringOrDefault(common.EnvironmentKeySDKToolVersion, "")
	if toolVersion != "" {
		req.SdkToolVersion = toolVersion
	}

	// Wait for response from ActivateServerProcess() request
	err = state.wsGameLift.HandleRequest(req, &res, ActivateServerProcessRequestTimeoutInSeconds)

	if err != nil {
		return common.NewGameLiftError(common.ProcessNotReady, "", err.Error())
	}
	state.isReadyProcess.Store(true)
	state.shutdown = make(chan bool)
	go state.startHealthCheck(state.shutdown)
	return nil
}

func (state *gameLiftServerState) processEnding() error {
	err := state.wsGameLift.HandleRequest(request.NewTerminateServerProcess(), nil, state.serviceCallTimeout)

	if err != nil {
		return common.NewGameLiftError(common.ProcessEndingFailed, "", err.Error())
	}
	state.stopServerProcess()

	return nil
}

func (state *gameLiftServerState) activateGameSession() error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	req := request.NewActivateGameSession(state.gameSessionID)
	err := state.wsGameLift.HandleRequest(req, nil, state.serviceCallTimeout)
	return err
}

func (state *gameLiftServerState) updatePlayerSessionCreationPolicy(policy *model.PlayerSessionCreationPolicy) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	if policy == nil {
		return common.NewGameLiftError(common.BadRequestException, "", "PlayerSessionCreationPolicy is required.")
	}
	err := ValidatePlayerSessionCreationPolicy(*policy)
	if err != nil {
		return err
	}
	req := request.NewUpdatePlayerSessionCreationPolicy(state.gameSessionID, *policy)
	err = state.wsGameLift.HandleRequest(req, nil, state.serviceCallTimeout)
	return err
}

func (state *gameLiftServerState) getGameSessionID() (string, error) {
	return state.gameSessionID, nil
}

// getTerminationTime - returns number of seconds that have elapsed since Unix epoch time begins (00:00:00 UTC Jan 1 1970).
func (state *gameLiftServerState) getTerminationTime() (int64, error) {
	if state.terminationTime == 0 {
		return 0, common.NewGameLiftError(common.TerminationTimeNotSet, "", "")
	}
	return state.terminationTime, nil
}

func (state *gameLiftServerState) acceptPlayerSession(playerSessionID string) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	err := ValidatePlayerSessionId(playerSessionID)
	if err != nil {
		return err
	}
	req := request.NewAcceptPlayerSession(state.gameSessionID, playerSessionID)
	err = state.wsGameLift.HandleRequest(req, nil, state.serviceCallTimeout)
	return err
}

func (state *gameLiftServerState) removePlayerSession(playerSessionID string) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if state.gameSessionID == "" {
		return common.NewGameLiftError(common.GamesessionIDNotSet, "", "")
	}
	err := ValidatePlayerSessionId(playerSessionID)
	if err != nil {
		return err
	}
	req := request.NewRemovePlayerSession(state.gameSessionID, playerSessionID)
	err = state.wsGameLift.HandleRequest(req, nil, state.serviceCallTimeout)
	return err
}

func (state *gameLiftServerState) describePlayerSessions(req *request.DescribePlayerSessionsRequest) (result.DescribePlayerSessionsResult, error) {
	var playerSessionResult result.DescribePlayerSessionsResult
	if !state.isReadyProcess.Load() {
		return playerSessionResult, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if req == nil {
		return playerSessionResult, common.NewGameLiftError(common.BadRequestException, "", "DescribePlayerSessionsRequest is required.")
	}
	err := ValidateDescribePlayerSessionsRequest(*req)
	if err != nil {
		return playerSessionResult, err
	}
	err = state.wsGameLift.HandleRequest(req, &playerSessionResult, state.serviceCallTimeout)
	return playerSessionResult, err
}

func (state *gameLiftServerState) startMatchBackfill(req *request.StartMatchBackfillRequest) (result.StartMatchBackfillResult, error) {
	var startMatchBackfillResult result.StartMatchBackfillResult
	if !state.isReadyProcess.Load() {
		return startMatchBackfillResult, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if req == nil {
		return startMatchBackfillResult, common.NewGameLiftError(common.BadRequestException, "", "StartMatchBackfillRequest is required.")
	}
	err := ValidateStartMatchBackfillRequest(*req)
	if err != nil {
		return startMatchBackfillResult, err
	}
	err = state.wsGameLift.HandleRequest(req, &startMatchBackfillResult, state.serviceCallTimeout)
	return startMatchBackfillResult, err
}

func (state *gameLiftServerState) stopMatchBackfill(req *request.StopMatchBackfillRequest) error {
	if !state.isReadyProcess.Load() {
		return common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	if req == nil {
		return common.NewGameLiftError(common.BadRequestException, "", "StopMatchBackfillRequest is required.")
	}
	err := ValidateStopMatchBackfillRequest(*req)
	if err != nil {
		return err
	}
	err = state.wsGameLift.HandleRequest(req, nil, state.serviceCallTimeout)
	return err
}

func (state *gameLiftServerState) getComputeCertificate() (result.GetComputeCertificateResult, error) {
	lg.Debugf("Calling GetComputeCertificate")
	var res result.GetComputeCertificateResult
	if !state.isReadyProcess.Load() {
		return res, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}
	err := state.wsGameLift.HandleRequest(request.NewGetComputeCertificate(), &res, state.serviceCallTimeout)
	return res, err
}

func (state *gameLiftServerState) getRoleCredentialsFromCache(roleArn string) (result.GetFleetRoleCredentialsResult, bool) {
	state.mtx.Lock()
	defer state.mtx.Unlock()
	if previousResult, ok := state.fleetRoleResultCache[roleArn]; ok {
		timeToLive := time.Duration(previousResult.Expiration-time.Now().UnixMilli()) * time.Millisecond
		if timeToLive > common.InstanceRoleCredentialTTL {
			return previousResult, true
		}
		delete(state.fleetRoleResultCache, roleArn)
	}
	return result.GetFleetRoleCredentialsResult{}, false
}

func (state *gameLiftServerState) getFleetRoleCredentials(
	req *request.GetFleetRoleCredentialsRequest,
) (result.GetFleetRoleCredentialsResult, error) {
	lg.Debugf("Calling GetFleetRoleCredentials")
	if req == nil {
		return result.GetFleetRoleCredentialsResult{},
			common.NewGameLiftError(common.BadRequestException, "", "GetFleetRoleCredentialsRequest is required.")
	}

	if !state.onManagedEC2 {
		return result.GetFleetRoleCredentialsResult{},
			common.NewGameLiftError(common.BadRequestException, "", "Fleet role credentials only available for servers hosted on managed fleet.")
	}

	res, ok := state.getRoleCredentialsFromCache(req.RoleArn)
	if ok {
		return res, nil
	}
	// If role session name was not provided, default to fleetId-hostId
	if req.RoleSessionName == "" {
		req.RoleSessionName = fmt.Sprintf("%s-%s", state.fleetID, state.hostID)
		if len(req.RoleSessionName) > common.RoleSessionNameMaxLength {
			req.RoleSessionName = req.RoleSessionName[:common.RoleSessionNameMaxLength]
		}
	}
	err := ValidateGetFleetRoleCredentialsRequest(*req)
	if err != nil {
		return res, err
	}
	if !state.isReadyProcess.Load() {
		return res, common.NewGameLiftError(common.ProcessNotReady, "", "")
	}

	err = state.wsGameLift.HandleRequest(req, &res, state.serviceCallTimeout)
	if err != nil {
		return res, err
	}
	if res.AccessKeyID == "" {
		state.onManagedEC2 = false
		return res, common.NewGameLiftError(common.BadRequestException, "", "Fleet role credentials only available for servers hosted on managed fleet.")
	}

	state.mtx.Lock()
	defer state.mtx.Unlock()
	state.fleetRoleResultCache[req.RoleArn] = res

	return res, err
}

func (state *gameLiftServerState) destroy() error {
	state.stopServerProcess()
	if state.wsGameLift == nil {
		return nil
	}
	err := state.wsGameLift.Disconnect()
	state.wsGameLift = nil
	return err
}

func (state *gameLiftServerState) stopServerProcess() {
	if state.isReadyProcess.CompareAndSwap(true, false) {
		if isChannelOpen(state.shutdown) && state.shutdown != nil {
			close(state.shutdown)
		}
	}
}

func (state *gameLiftServerState) startHealthCheck(done <-chan bool) {
	lg.Debugf("HealthCheck thread started.")
	for state.isReadyProcess.Load() {
		timeout := time.After(state.getNextHealthCheckIntervalSeconds())
		go state.heartbeatServerProcess(done)
		select {
		case <-timeout:
			continue
		case <-done:
			return
		}
	}
}

func (state *gameLiftServerState) heartbeatServerProcess(done <-chan bool) {
	res := make(chan bool)
	go func(res chan<- bool) {
		if state.parameters != nil && state.parameters.OnHealthCheck != nil {
			lg.Debugf("Reporting health using the OnHealthCheck callback.")
			res <- state.parameters.OnHealthCheck()
		} else {
			close(res)
		}
	}(res)
	timeout := time.After(state.healthCheckTimeout)
	status := false
	select {
	case <-timeout:
		lg.Debugf("Timed out waiting for health response from the server process. Reporting as unhealthy.")
		status = false
	case status = <-res:
		lg.Debugf("Received health response from the server process: %v", status)
	case <-done:
		return
	}
	var response message.Message
	err := state.wsGameLift.HandleRequest(
		request.NewHeartbeatServerProcess(status),
		&response,
		state.serviceCallTimeout,
	)
	if err != nil {
		lg.Warnf("Could not send health status: %s", err)
	}
}

// getNextHealthCheckIntervalSeconds - return a healthCheck interval +/- a random value
// between [- defaultJitterIntervalMs, defaultJitterIntervalMs].
//
//nolint:gosec // weak math random generator is enough in this case
func (state *gameLiftServerState) getNextHealthCheckIntervalSeconds() time.Duration {
	jitterMs := 2*localRnd.Int63n(state.defaultJitterIntervalMs) - state.defaultJitterIntervalMs
	return state.healthCheckInterval - time.Duration(jitterMs)*time.Millisecond
}

// OnStartGameSession handler for message.CreateGameSessionMessage (already started in a separate goroutine).
func (state *gameLiftServerState) OnStartGameSession(session *model.GameSession) {
	if session == nil {
		lg.Warnf("OnStartGameSession was called with nil game session")
		return
	}
	// Inject data that already exists on the server
	session.FleetID = state.fleetID
	lg.Debugf("server got the startGameSession signal. GameSession : %s", session.GameSessionID)
	if !state.isReadyProcess.Load() {
		lg.Debugf("Got a game session on inactive process. Ignoring.")
		return
	}
	state.gameSessionID = session.GameSessionID
	if state.parameters != nil && state.parameters.OnStartGameSession != nil {
		state.parameters.OnStartGameSession(*session)
	}
}

// OnUpdateGameSession - handler for message.UpdateGameSessionMessage (already started in a separate goroutine).
func (state *gameLiftServerState) OnUpdateGameSession(
	gameSession *model.GameSession,
	updateReason *model.UpdateReason,
	backfillTicketID string,
) {
	if gameSession == nil {
		lg.Warnf("OnUpdateGameSession was called with nil game session")
		return
	}
	lg.Debugf("ServerState got the updateGameSession signal. GameSession : %s", gameSession.GameSessionID)
	if !state.isReadyProcess.Load() {
		lg.Warnf("Got an updated game session on inactive process.")
		return
	}
	if updateReason == nil {
		lg.Warnf("OnUpdateGameSession was called with nil update reason")
	}
	if state.parameters != nil && state.parameters.OnUpdateGameSession != nil {
		state.parameters.OnUpdateGameSession(
			model.UpdateGameSession{
				GameSession:      *gameSession,
				UpdateReason:     updateReason,
				BackfillTicketID: backfillTicketID,
			},
		)
	}
}

// OnTerminateProcess - handler for message.TerminateProcessMessage (already started in a separate goroutine).
func (state *gameLiftServerState) OnTerminateProcess(terminationTime int64) {
	// terminationTime is milliseconds that have elapsed since Unix epoch time begins (00:00:00 UTC Jan 1 1970).
	state.terminationTime = terminationTime / 1000
	lg.Debugf("ServerState got the terminateProcess signal. termination time : %d", state.terminationTime)
	if state.parameters != nil && state.parameters.OnProcessTerminate != nil {
		state.parameters.OnProcessTerminate()
	} else {
		lg.Debugf("OnProcessTerminate handler is not defined. Calling ProcessEnding() and Destroy()")
		processEndingErr := state.processEnding()
		destroyErr := state.destroy()
		if processEndingErr == nil && destroyErr == nil {
			exitFunc(0)
		} else {
			if processEndingErr != nil {
				lg.Errorf("ProcessEnding failed: %s", processEndingErr)
			}
			if destroyErr != nil {
				lg.Errorf("Destroy failed: %s", destroyErr)
			}
			exitFunc(-1)
		}
	}
}

// OnRefreshConnection - callback function that the Amazon GameLift Servers service invokes when
// the server process need to refresh current websocket connection.
func (state *gameLiftServerState) OnRefreshConnection(refreshConnectionEndpoint, authToken string) {
	err := state.wsGameLift.Connect(
		refreshConnectionEndpoint,
		state.processID,
		state.hostID,
		state.fleetID,
		authToken,
		nil,
	)
	if err != nil {
		lg.Errorf("Failed to refresh websocket connection. The sever SDK will try again each minute "+
			"until the refresh succeeds, or the websocket is forcibly closed: %s", err)
	}
}

func isChannelOpen(ch <-chan bool) bool {
	select {
	case <-ch:
		return false
	default:
	}
	return true
}
