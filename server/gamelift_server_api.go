/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package server

import (
	"aws/amazon-gamelift-go-sdk/common"
	"aws/amazon-gamelift-go-sdk/model"
	"aws/amazon-gamelift-go-sdk/model/request"
	"aws/amazon-gamelift-go-sdk/model/result"
	"aws/amazon-gamelift-go-sdk/server/internal"
	"aws/amazon-gamelift-go-sdk/server/internal/transport"
	"aws/amazon-gamelift-go-sdk/server/log"
)

var srv iGameLiftServerState
var state gameLiftServerState
var manager internal.IGameLiftManager

var lg log.ILogger = log.GetDefaultLogger()

func getServerParamsFromEnvironment() (ServerParameters, error) {
	websocketURL, err := common.GetEnvStringOrError(common.EnvironmentKeyWebsocketURL)
	if err != nil {
		return ServerParameters{}, err
	}
	processID, err := common.GetEnvStringOrError(common.EnvironmentKeyProcessID)
	if err != nil {
		return ServerParameters{}, err
	}
	hostID, err := common.GetEnvStringOrError(common.EnvironmentKeyHostID)
	if err != nil {
		return ServerParameters{}, err
	}
	fleetID, err := common.GetEnvStringOrError(common.EnvironmentKeyFleetID)
	if err != nil {
		return ServerParameters{}, err
	}
	authToken, err := common.GetEnvStringOrError(common.EnvironmentKeyAuthToken)
	if err != nil {
		return ServerParameters{}, err
	}
	return ServerParameters{
		WebSocketURL: websocketURL,
		ProcessID:    processID,
		HostID:       hostID,
		FleetID:      fleetID,
		AuthToken:    authToken,
	}, nil
}

// SetLoggerInterface - use this function to inject custom logger to the GameLift SDK.
//
// It allows you to add your own logger to the SDK from the application, see log.ILogger.
func SetLoggerInterface(l log.ILogger) {
	lg = l
}

// GetSdkVersion - returns the current version number of the SDK built into the server process.
// The returned string includes the version number only (ex. 5.0.0).
// If not successful, returns an error message see common.SdkVersionDetectionFailed.
func GetSdkVersion() (string, error) {
	return common.SdkVersion, nil
}

// InitSDK - Initializes the GameLift server SDK.
// This method should be called on launch, before any other GameLift-related initialization occurs.
// If successful, returns nil indicating that the server process is ready.
//
//	Receive: ServerParameters
//
// If successful, returns nil error indicating that the server process is ready to call.
//
//	serverParameters := ServerParameters{
//			WebSocketURL: webSocketUrl,
//			ProcessID:    processId,
//			HostID:       hostId,
//			FleetID:      fleetId,
//			AuthToken:	  authToken,
//	}
//
// InitSDK will establish a local connection with GameLift's agent to enable further communication.
//
//	err := server.InitSDK(serverParameters)
func InitSDK(params ServerParameters) error {
	var err error
	if srv != nil {
		return common.NewGameLiftError(common.AlreadyInitialized, "", "")
	}
	if manager == nil {
		wsDialer := transport.NewDialer(lg)
		wsTransport := transport.Websocket(lg, wsDialer)
		wsTransport = transport.WithRetry(wsTransport, lg)
		client := internal.GetWebsocketClient(wsTransport, lg)
		manager = internal.GetGameLiftManager(&state, client, lg)
	}
	err = state.init(&params, manager)
	srv = &state
	return err
}

// InitSDKFromEnvironment - Initializes the GameLift server SDK from system environment variables
// This method should be called on launch, before any other GameLift-related initialization occurs.
// If successful, returns nil indicating that the server process is ready.
//
// If successful, returns nil error indicating that the server process is ready to call.
//
// InitSDKFromEnvironment will establish a local connection with GameLift's agent to enable further communication.
//
//	err := server.InitSDKFromEnvironment()
func InitSDKFromEnvironment() error {
	serverParams, err := getServerParamsFromEnvironment()
	if err != nil {
		return common.NewGameLiftError(common.NotInitialized, "Could not get server parameters from system environment variables", err.Error())
	}
	return InitSDK(serverParams)
}

// ProcessReady - notifies GameLift that the server process is ready to host game sessions (receive model.GameSession).
// Call this method after successfully invoking InitSDK
// and completing setup tasks that are required before the server process can host a game session.
// This method should be called only once per process.
//
//		Receive: ProcessParameters - object communicating the following information about the server process:
//
//	  - Names of callback methods, implemented in the game server code, that the GameLift service invokes to
//	    communicate with the server process.
//	  - Port number that the server process is listening on
//	  - Path to any game session-specific files that you want GameLift to capture and store.
//
//	Returns an error if failure with an error message.
//
// Set parameters and call ProcessReady()
//
//	processParams := ProcessParameters{
//			OnStartGameSession: gameProcess.OnGameSessionHandler,
//			OnUpdateGameSession: gameProcess.OnGameSessionUpdate,
//			OnProcessTerminate: gameProcess.OnProcessTerminateHandler,
//			OnHealthCheck: gameProcess.OnHealthCheckHandler,
//			Port: port,
//		 	LogParameters: LogParameters{    // logging and error example
//		 		[]string{ "C:\\game\\logs", "C:\\game\\error"},
//			},
//		}
//
// err := server.ProcessReady(processParams);
func ProcessReady(param ProcessParameters) error {
	return srv.processReady(&param)
}

// ProcessEnding - notifies the GameLift service that the server process is shutting down.
// This method should be called after all other cleanup tasks, including shutting down all active game sessions.
// Once the method exits with a nil error, you can terminate the process with a successful exit code.
// You can also exit the process with an error code.
// If you exit with an error code,
// the fleet event will indicate the process terminated abnormally (SERVER_PROCESS_TERMINATED_UNHEALTHY).
//
//	if err := server.ProcessEnding(); err == nil {
//			os.Exit(0)
//	}
//	// otherwise, exit with error code
//	os.Exit(errorCode)
func ProcessEnding() error {
	return srv.processEnding()
}

// ActivateGameSession - notifies GameLift that the server is requesting a game session and is now ready to
// receive player connections. This action should be called as part of the ProcessParameters.OnStartGameSession callback
// function, after all game session initialization has been completed.
//
//	Returns an error if failure with an error message.
//
//	func OnStartGameSession(GameSession gameSession) {
//			// game-specific tasks when starting a new game session, such as loading map
//			// When ready to receive players
//			err := server.ActivateGameSession()
//		...
//	}
func ActivateGameSession() error {
	return srv.activateGameSession()
}

// UpdatePlayerSessionCreationPolicy - updates the current game session's ability to accept new player sessions.
// A game session can be set to either accept or deny all new player sessions.
//
//	Receive: model.PlayerSessionCreationPolicy - a string value indicating whether the game session accepts new players.
//		Valid values include:
//
//	  - model.AcceptAll – Accept all new player sessions.
//	  - model.DenyAll – Deny all new player sessions.
//
//	Returns an error if failure with an error message.
//
// err := server.UpdatePlayerSessionCreationPolicy(model.AcceptAll)
func UpdatePlayerSessionCreationPolicy(policy model.PlayerSessionCreationPolicy) error {
	return srv.updatePlayerSessionCreationPolicy(&policy)
}

// GetGameSessionID - retrieves the ID of the game session currently being hosted by the server process,
// if the server process is active.
//
// If successful, returns the game session ID and nil error
// For idle process that are not yet activated with a game session, the call returns empty string and nil error
//
// gameSessionID, err := server.GetGameSessionID()
func GetGameSessionID() (string, error) {
	return srv.getGameSessionID()
}

// GetTerminationTime - returns the timestamp in epoch seconds that a server process is scheduled to be shut down,
// if a termination time is available.
// A server process takes this action after receiving an ProcessParameters.OnProcessTerminate() callback from the GameLift service.
// GameLift may call ProcessParameters.OnProcessTerminate() for the following reasons:
//   - For poor health (the server process has reported port health or has not responded to GameLift)
//   - When terminating the instance during a scale-down event
//   - When an instance is being terminated due to a spot-instance interruption,
//     see: https://docs.aws.amazon.com/gamelift/latest/developerguide/spot-tasks.html
//
// If the process has received an ProcessParameters.OnProcessTerminate() callback, the value returned is the estimated
// termination time. If the process has not received an ProcessParameters.OnProcessTerminate() callback,
// an error message is returned. Learn more about shutting down a server process here:
// https://docs.aws.amazon.com/gamelift/latest/developerguide/gamelift-sdk-server-api.html#gamelift-sdk-server-terminate
//
// If successful, returns the termination time and nil error.
// If no termination time is available, returns an error message.
//
// terminationTime, err := server.GetTerminationTime()
func GetTerminationTime() (int64, error) {
	return srv.getTerminationTime()
}

// AcceptPlayerSession - notifies the GameLift service that a player with the specified player session ID has connected
// to the server process and needs validation. GameLift verifies that the player session ID is valid—that is,
// that the player ID has reserved a player slot in the game session. Once validated,
// GameLift changes the model.playerSessionStatus from model.PlayerReserved to model.PlayerActive
//
//	Receive: Unique ID issued by GameLift when a new player session is created.
//
//	Returns an error if failure with an error message.
//
//	func ReceiveConnectingPlayerSessionID(conn Connection, playerSessionID string) {
//			err := server.AcceptPlayerSession(playerSessionID)
//			if err != nil {
//				connection.Accept()
//			} else {
//				connection.Reject(err.Error())
//			}
//		}
func AcceptPlayerSession(playerSessionID string) error {
	return srv.acceptPlayerSession(playerSessionID)
}

// RemovePlayerSession - notifies the GameLift service that a player with the specified player session ID
// has disconnected from the server process.
// In response, GameLift changes the player slot to available, which allows it to be assigned to a new player.
//
//	Receive: Unique ID issued by GameLift when a new player session is created.
//
//	Returns an error if failure with an error message.
//
// err := server.RemovePlayerSession(playerSessionID)
func RemovePlayerSession(playerSessionID string) error {
	return srv.removePlayerSession(playerSessionID)
}

// DescribePlayerSessions - retrieves player session data, including settings, session metadata, and player data.
// Use this action to get information about:
//
//   - a single player session,
//
//   - all player sessions in a game session,
//
//   - all player sessions associated with a single player ID.
//
//     Receive: request.DescribePlayerSessionsRequest - object describing which player sessions to retrieve.
//
// If successful, a result.DescribePlayerSessionsResult object is returned
// containing a set of player session objects that fit the request parameters.
//
//	describePlayerSessionsRequest := request.NewDescribePlayerSessions() // to create this request, please use this func
//	describePlayerSessionsRequest.GameSessionID, _ = server.GetGameSessionID() //gets the IDs for the current game session
//	describePlayerSessionsRequest.Limit = 10 // return the first 10 player sessions
//	describePlayerSessionsRequest.PlayerSessionStatusFilter = "ACTIVE" // All player sessions actively connected to a specified game session
//	describePlayerSessionsResult, err := server.DescribePlayerSessions(describePlayerSessionsRequest)
func DescribePlayerSessions(req request.DescribePlayerSessionsRequest) (result.DescribePlayerSessionsResult, error) {
	return srv.describePlayerSessions(&req)
}

// StartMatchBackfill - sends a request to find new players for open slots in a game session created with FlexMatch.
//
//	See also the AWS SDK action https://docs.aws.amazon.com/gamelift/latest/apireference/API_StartMatchBackfill.html.
//	With this action, request.StartMatchBackfillRequest can be
//	initiated by a game server process that is hosting the game session. Learn more about the FlexMatch backfill feature.
//	See: "https://docs.aws.amazon.com/gamelift/latest/flexmatchguide/match-backfill.html">
//
// This action is asynchronous. If new players are successfully matched, the GameLift service delivers updated
// matchmaker data using the callback function the ProcessParameters.OnUpdateGameSession callback.
//
// A server process can have only one active match backfill request at a time.
// To send a new request, first call server.StopMatchBackfill to cancel the original request.
//
//	Receive: request.StartMatchBackfillRequest - object that communicates the following information:
//	- A ticket ID to assign to the backfill request. This information is optional; if no ID is provided, GameLift will autogenerate one.
//	- The matchmaker to send the request to. The full configuration ARN is required.
//		This value can be acquired from the game session's matchmaker data.
//	- The ID of the game session that is being backfilled.
//	- Available matchmaking data for the game session's current players.
//
//	If successful, returns a result.StartMatchBackfillResult - object with the match backfill ticket id,
//	otherwise return an error.
//
//	startBackfillRequest := request.NewStartMatchBackfill() // Please use this function to create request
//	startBackfillRequest.RequestID = "a ticket ID"          // optional
//	startBackfillRequest.MatchmakingConfigurationArn = "the matchmaker configuration ARN"
//	var matchMaker model.MatchmakerData
//	if err := matchMaker.UnmarshalJSON([]byte(gameSession.MatchmakerData)); err != nil {
//		return
//	}
//	startBackfillRequest.Players = matchMaker.Players
//	res, err := server.StartMatchBackfill(startBackfillRequest)
//
//	// Implement callback function for backfill
//	func OnUpdateGameSession(myGameSession model.GameSession){
//		// game-specific tasks to prepare for the newly matched players and update matchmaker data as needed
//	}
func StartMatchBackfill(req request.StartMatchBackfillRequest) (result.StartMatchBackfillResult, error) {
	return srv.startMatchBackfill(&req)
}

// StopMatchBackfill - cancels an active match backfill request that was created with StartMatchBackfill().
// Learn more about the FlexMatch backfill feature:
// https://docs.aws.amazon.com/gamelift/latest/flexmatchguide/match-backfill.html
//
//	Receive: request.StopMatchBackfillRequest - object identifying the matchmaking ticket to cancel:
//	- Ticket ID assigned to the backfill request being canceled
//	- Matchmaker the backfill request was sent to
//	- Game session associated with the backfill request
//
//	Returns an error if failure with an error message.
//
//	stopBackfillRequest := request.NewStopMatchBackfill()     // Please use this function to create request
//	stopBackfillRequest.TicketID = "a ticket ID"              // optional, if not provided one is autogenerated
//	stopBackfillRequest.MatchmakingConfigurationArn = "the matchmaker configuration ARN" // from the game session matchmaker data
//	err := server.StopMatchBackfill(stopBackfillRequest)
func StopMatchBackfill(req request.StopMatchBackfillRequest) error {
	return srv.stopMatchBackfill(&req)
}

// GetComputeCertificate - retrieves the path to TLS certificate used to encrypt the network connection between your
// Anywhere compute resource and GameLift. You can use the certificate path when you register
// your compute device to a GameLift Anywhere fleet. For more information see,
// https://docs.aws.amazon.com/gamelift/latest/apireference/API_RegisterCompute.html.
//
//	Returns an object containing the following:
//	- CertificatePath - The path to the TLS certificate on your compute resource.
//	- ComputeName - The hostname of your compute resource.
//
// tlsCertificate, err := server.GetComputeCertificate()
func GetComputeCertificate() (result.GetComputeCertificateResult, error) {
	return srv.getComputeCertificate()
}

// GetFleetRoleCredentials - retrieves the service role credentials you created to extend permissions to
// your other AWS services to GameLift. These credentials allow your game server to
// use your AWS resources. For more information, see
// https://docs.aws.amazon.com/gamelift/latest/developerguide/setting-up-role.html
//
// Returns result.GetFleetRoleCredentialsResult - an object that contains the following:
//   - AssumedRoleUserArn - The Amazon Resource Name (ARN) of the user that the service role belongs to.
//   - AssumedRoleID - The ID of the user that the service role belongs to.
//   - AccessKeyId - The access key ID used to authenticate and provide access to your AWS resources.
//   - SecretAccessKey - The secret access key ID used for authentication.
//   - SessionToken - A token used to identify the current active session interacting with your AWS resources.
//   - Expiration - The amount of time until your session credentials expire.
//
// request.GetFleetRoleCredentialsRequest - object identifying the Amazon Resource Name of the role you are requesting the credentials of.
//
//	// form the customer credentials request
//	getFleetRoleCredentialsRequest := request.NewGetFleetRoleCredentials() // Please use this function to create request
//	getFleetRoleCredentialsRequest.RoleArn = "arn:aws:iam::123456789012:role/service-role/exampleGameLiftAction"
//	credentials, err := server.GetFleetRoleCredentials(getFleetRoleCredentialsRequest)
func GetFleetRoleCredentials(
	req request.GetFleetRoleCredentialsRequest,
) (result.GetFleetRoleCredentialsResult, error) {
	return srv.getFleetRoleCredentials(&req)
}

// Destroy - deletes the instance of the GameLift Game Server SDK on your resource.
// This removes all state information, stops heartbeat communication with GameLift, stops game session management, and
// closes any connections. Call this after you've use server.ProcessEnding()
//
//	Returns an error if failure with an error message.
//
//	// operations to end game sessions and the server process
//	defer func() {
//		err := server.ProcessEnding();
//		server.Destroy();
//		if err != nil {
//			os.Exit(errorCode)
//		}
//	}
func Destroy() error {
	if err := srv.destroy(); err != nil {
		return err
	}
	manager = nil
	srv = nil
	return nil
}
