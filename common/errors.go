/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"fmt"
	"net/http"
)

type GameLiftErrorType int

const (
	// AlreadyInitialized - The GameLift Server or Client has already been initialized with Initialize().
	AlreadyInitialized GameLiftErrorType = iota
	// FleetMismatch - The target fleet does not match the fleet of a gameSession or playerSession.
	FleetMismatch
	// GameLiftClientNotInitialized - The GameLift client has not been initialized.
	GameLiftClientNotInitialized
	// GameLiftServerNotInitialized - The GameLift server has not been initialized.
	GameLiftServerNotInitialized
	// GameSessionEndedFailed - The GameLift Server SDK could not contact the service to report the game session ended.
	GameSessionEndedFailed
	// GameSessionNotReady - The GameLift Server Game Session was not activated.
	GameSessionNotReady
	// GameSessionReadyFailed - The GameLift Server SDK could not contact the service to report the game session is ready.
	GameSessionReadyFailed
	// GamesessionIDNotSet - No game sessions are bound to this process.
	GamesessionIDNotSet
	// InitializationMismatch - A client method was called after Server::Initialize(), or vice versa.
	InitializationMismatch
	// NotInitialized - The GameLift Server or Client has not been initialized with Initialize().
	NotInitialized
	// NoTargetAliasIDSet - A target aliasId has not been set.
	NoTargetAliasIDSet
	// NoTargetFleetSet - A target fleet has not been set.
	NoTargetFleetSet
	// ProcessEndingFailed - The GameLift Server SDK could not contact the service to report the process is ending.
	ProcessEndingFailed
	// ProcessNotActive - The server process is not yet active, not bound to a GameSession, and cannot accept or process PlayerSessions.
	ProcessNotActive
	// ProcessNotReady - The server process is not yet ready to be activated.
	ProcessNotReady
	// ProcessReadyFailed - The GameLift Server SDK could not contact the service to report the process is ready.
	ProcessReadyFailed
	// SdkVersionDetectionFailed - SDK version detection failed.
	SdkVersionDetectionFailed
	// ServiceCallFailed - A call to an AWS service has failed.
	ServiceCallFailed
	// UnexpectedPlayerSession - An unregistered player session was encountered by the server.
	UnexpectedPlayerSession
	// LocalConnectionFailed - Connection to local agent could not be established.
	LocalConnectionFailed
	// NetworkNotInitialized - Local network was not initialized.
	NetworkNotInitialized
	// TerminationTimeNotSet - termination time has not been sent to this process.
	TerminationTimeNotSet
	// BadRequestException - An error may occur when the request does not contain a request id, or the request cannot be serialized, etc.
	BadRequestException
	// InternalServiceException - internal service error.
	InternalServiceException
	// WebsocketConnectFailure - Failure to connect to the GameLift Service WebSocket.
	WebsocketConnectFailure
	// WebsocketRetriableSendMessageFailure - Retriable failure to send message to the GameLift Service WebSocket.
	WebsocketRetriableSendMessageFailure
	// WebsocketSendMessageFailure - Failure to send message to the GameLift Service WebSocket.
	WebsocketSendMessageFailure
	// WebsocketClosingError - An error may occur when try close a websocket.
	WebsocketClosingError
)

type errorDescription struct {
	name    string
	message string
}

// errorMessages read-only map that contains all GameLift errors.
var errorMessages = map[GameLiftErrorType]errorDescription{
	AlreadyInitialized: {
		name:    "Already Initialized",
		message: "GameLift has already been initialized. You must call destroy() before reinitializing the client or server.",
	},
	FleetMismatch: {
		name: "Fleet mismatch.",
		message: "The Target fleet does not match the request fleet. " +
			"Make sure GameSessions and PlayerSessions belong to your target fleet.",
	},
	GameLiftClientNotInitialized: {
		name:    "GameLift client not initialized.",
		message: "The GameLift client has not been initialized. Please call SetTargetFleet or SetTArgetAliasId.",
	},
	GameLiftServerNotInitialized: {
		name:    "GameLift server not initialized.",
		message: "The GameLift server has not been initialized. Please call InitSDK.",
	},
	GameSessionEndedFailed: {
		name:    "Game session failed.",
		message: "The GameSessionEnded invocation failed.",
	},
	GameSessionNotReady: {
		name:    "Game session not activated.",
		message: "The Game session associated with this server was not activated.",
	},
	GameSessionReadyFailed: {
		name:    "Game session failed.",
		message: "The GameSessionReady invocation failed.",
	},
	GamesessionIDNotSet: {
		name:    "GameSession id is not set.",
		message: "No game sessions are bound to this process.",
	},
	InitializationMismatch: {
		name: "Initialization mismatch.",
		message: "The current call does not match the initialization state. " +
			"Client calls require a call to Client::Initialize(), and Server calls require Server::Initialize(). " +
			"Only one may be active at a time.",
	},
	NotInitialized: {
		name:    "Not Initialized",
		message: "GameLift has not been initialized! You must call Client::Initialize() or Server::InitSDK() before making GameLift calls.",
	},
	NoTargetAliasIDSet: {
		name:    "No target aliasId set.",
		message: "The aliasId has not been set. Clients should call SetTargetAliasId() before making calls that require an alias.",
	},
	NoTargetFleetSet: {
		name:    "No target fleet set.",
		message: "The target fleet has not been set. Clients should call SetTargetFleet() before making calls that require a fleet.",
	},
	ProcessEndingFailed: {
		name:    "Process ending failed.",
		message: "processEnding call to GameLift failed.",
	},
	ProcessNotActive: {
		name:    "Process not activated.",
		message: "The process has not yet been activated.",
	},
	ProcessNotReady: {
		name: "Process not ready.",
		message: "The process has not yet been activated by calling ProcessReady(). " +
			"Processes in standby cannot receive StartGameSession callbacks.",
	},
	ProcessReadyFailed: {
		name:    "Process ready failed.",
		message: "ProcessReady call to GameLift failed.",
	},
	SdkVersionDetectionFailed: {
		name:    "Could not detect SDK version.",
		message: "Could not detect SDK version.",
	},
	ServiceCallFailed: {
		name:    "Service call failed.",
		message: "An AWS service call has failed. See the root cause error for more information.",
	},
	UnexpectedPlayerSession: {
		name: "Unexpected player session.",
		message: "The player session was not expected by the server. " +
			"Clients wishing to connect to a server must obtain a PlayerSessionID from GameLift " +
			"by creating a player session on the desired server's game instance.",
	},
	LocalConnectionFailed: {
		name:    "Local connection failed.",
		message: "Connection to local agent could not be established.",
	},
	NetworkNotInitialized: {
		name:    "Network not initialized.",
		message: "Local network was not initialized. Have you called InitSDK()?",
	},
	TerminationTimeNotSet: {
		name:    "TerminationTime is not set.",
		message: "TerminationTime has not been sent to this process.",
	},
	BadRequestException: {
		name:    "Bad request exception.",
		message: "Bad request exception.",
	},
	InternalServiceException: {
		name:    "Internal service exception.",
		message: "Internal service exception.",
	},
	WebsocketConnectFailure: {
		name:    "WebSocket Connection Failed",
		message: "Connection to the GameLift Service WebSocket has failed",
	},
	WebsocketRetriableSendMessageFailure: {
		name:    "WebSocket Send Message Failed",
		message: "Sending Message to the GameLift Service WebSocket has failed",
	},
	WebsocketSendMessageFailure: {
		name:    "WebSocket Send Message Failed",
		message: "Sending Message to the GameLift Service WebSocket has failed",
	},
	WebsocketClosingError: {
		name:    "WebSocket close error",
		message: "An error has occurred in closing the connection",
	},
}

// GameLiftError -  represents an errors in GameLift SDK.
type GameLiftError struct {
	ErrorType GameLiftErrorType
	errorDescription
}

// NewGameLiftError - creates a new GameLiftError.
//
// Example:
//
//	err := common.NewGameLiftError(common.ProcessNotActive, "", "")
//	err := common.NewGameLiftError(common.WebsocketSendMessageFailure, "Can not send message", "Message is incorrect")
func NewGameLiftError(errorType GameLiftErrorType, name, message string) error {
	return &GameLiftError{
		ErrorType: errorType,
		errorDescription: errorDescription{
			name:    name,
			message: message,
		},
	}
}

// NewGameLiftErrorFromStatusCode - convert statusCode and errorMessage to the GameLiftError.
func NewGameLiftErrorFromStatusCode(statusCode int, errorMessage string) error {
	return NewGameLiftError(getErrorTypeForStatusCode(statusCode), "", errorMessage)
}

func (e *GameLiftError) Error() string {
	return fmt.Sprintf("[GameLiftError: ErrorType={%d}, ErrorName={%s}, ErrorMessage={%s}]",
		e.ErrorType,
		e.getNameOrDefaultForErrorType(),
		e.getMessageOrDefaultForErrorType(),
	)
}

func (e *GameLiftError) getMessageOrDefaultForErrorType() string {
	if e.message != "" {
		return e.message
	}
	if description, ok := errorMessages[e.ErrorType]; ok {
		return description.message
	}

	return "An unexpected error has occurred."
}

func (e *GameLiftError) getNameOrDefaultForErrorType() string {
	if e.name != "" {
		return e.name
	}
	if description, ok := errorMessages[e.ErrorType]; ok {
		return description.name
	}

	return "Unknown Error"
}

func getErrorTypeForStatusCode(statusCode int) GameLiftErrorType {
	// Map all 4xx requests to bad request exception. We don't have an error type for all 100 client errors
	if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		return BadRequestException
	}

	// The websocket can return other error types, in this case classify it as an internal service exception
	return InternalServiceException
}
