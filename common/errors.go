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
	// AlreadyInitialized - The server SDK has already been initialized with Initialize().
	AlreadyInitialized GameLiftErrorType = iota
	// FleetMismatch - The target fleet does not match the fleet of a gameSession or playerSession.
	FleetMismatch
	// GameLiftClientNotInitialized - The server SDK has not been initialized.
	GameLiftClientNotInitialized
	// GameLiftServerNotInitialized - The server SDK has not been initialized.
	GameLiftServerNotInitialized
	// GameSessionEndedFailed - The server SDK could not contact the service to report the game session ended.
	GameSessionEndedFailed
	// GameSessionNotReady - The game session was not activated.
	GameSessionNotReady
	// GameSessionReadyFailed - The server SDK could not contact the service to report the game session is ready.
	GameSessionReadyFailed
	// GamesessionIDNotSet - No game sessions are bound to this process.
	GamesessionIDNotSet
	// InitializationMismatch - A client method was called after Server::Initialize(), or vice versa.
	InitializationMismatch
	// NotInitialized - The server SDK has not been initialized with Initialize().
	NotInitialized
	// NoTargetAliasIDSet - A target aliasId has not been set.
	NoTargetAliasIDSet
	// NoTargetFleetSet - A target fleet has not been set.
	NoTargetFleetSet
	// ProcessEndingFailed - The server SDK could not contact the service to report the process is ending.
	ProcessEndingFailed
	// ProcessNotActive - The server process is not yet active, not bound to a GameSession, and cannot accept or process PlayerSessions.
	ProcessNotActive
	// ProcessNotReady - The server process is not yet ready to be activated.
	ProcessNotReady
	// ProcessReadyFailed - The server SDK could not contact the service to report the process is ready.
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
	// UnauthorizedException - Client request error due to invalid or missing authorization.
	UnauthorizedException
	// ForbiddenException - Client requesting to use resources/operations they aren't allow to access.
	ForbiddenException
	// NotFoundException - Request needs a backend resource that the client failed to install.
	NotFoundException
	// ConflictException - Client request error caused by stale data.
	ConflictException
	// TooManyRequestsException - Client called request too often and is being throttled.
	TooManyRequestsException
	// InternalServiceException - internal service error.
	InternalServiceException
	// ValidationException - Client-side error when invalid parameters are passed.
	ValidationException
	// WebsocketConnectFailure - Failure to connect to Amazon GameLift Servers websocket.
	WebsocketConnectFailure
	// WebsocketRetriableSendMessageFailure - Retriable failure to send message to the Amazon GameLift Servers WebSocket.
	WebsocketRetriableSendMessageFailure
	// WebsocketSendMessageFailure - Failure to send message to the Amazon GameLift Servers WebSocket.
	WebsocketSendMessageFailure
	// WebsocketClosingError - An error may occur when try close a websocket.
	WebsocketClosingError
	// UnknownException - Unknown error. Used for testing purposes, never thrown.
	UnknownException
)

type errorDescription struct {
	name    string
	message string
}

// errorMessages read-only map that contains all Amazon GameLift Servers errors.
var errorMessages = map[GameLiftErrorType]errorDescription{
	AlreadyInitialized: {
		name:    "Already Initialized",
		message: "Server SDK has already been initialized. You must call Destroy() before reinitializing the server SDK.",
	},
	FleetMismatch: {
		name: "Fleet mismatch.",
		message: "The Target fleet does not match the request fleet. " +
			"Make sure GameSessions and PlayerSessions belong to your target fleet.",
	},
	GameLiftClientNotInitialized: {
		name:    "Sever SDK not initialized.",
		message: "You must call InitSDK() before making calls to the server SDK for Amazon GameLift Servers.",
	},
	GameLiftServerNotInitialized: {
		name:    "Server SDK not initialized.",
		message: "You must call InitSDK() before making calls to the server SDK for Amazon GameLift Servers.",
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
		name:    "Server SDK not initialized.",
		message: "You must call InitSDK() before making calls to the server SDK for Amazon GameLift Servers.",
	},
	NotInitialized: {
		name:    "Server SDK not initialized.",
		message: "You must call InitSDK() before making calls to the server SDK for Amazon GameLift Servers.",
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
		message: "The server SDK call to ProcessEnding() failed.",
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
		message: "The server SDK call to ProcessEnding() failed.",
	},
	SdkVersionDetectionFailed: {
		name:    "Could not detect SDK version.",
		message: "Could not detect SDK version.",
	},
	ServiceCallFailed: {
		name:    "Service call failed.",
		message: "The call to an AWS service has failed. See the root cause error for more information.",
	},
	UnexpectedPlayerSession: {
		name: "Unexpected player session.",
		message: "The player session was not expected by the server. " +
			"Clients wishing to connect to a server must obtain a PlayerSessionID from Amazon GameLift Servers " +
			"by creating a player session on the desired game session.",
	},
	LocalConnectionFailed: {
		name:    "Local connection failed.",
		message: "Connection to the game server could not be established.",
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
	UnauthorizedException: {
		name:    "Unauthorized exception.",
		message: "User provided invalid or missing authorization to access a resource/operation.",
	},
	ForbiddenException: {
		name:    "Forbidden exception.",
		message: "User is attempting to access resources/operations that they are not allowed to access.",
	},
	NotFoundException: {
		name:    "Not found exception.",
		message: "A necessary resource was missing when attempting to process the request.",
	},
	ConflictException: {
		name:    "Conflict exception.",
		message: "Request conflicts with the current state of the target resource.",
	},
	TooManyRequestsException: {
		name:    "Throttling exception.",
		message: "Too many requests; please increase throttle limit if needed.",
	},
	InternalServiceException: {
		name:    "Internal service exception.",
		message: "Internal service exception.",
	},
	ValidationException: {
		name:    "Validation exception.",
		message: "Validation exception.",
	},
	WebsocketConnectFailure: {
		name:    "WebSocket Connection Failed",
		message: "Connection to the Amazon GameLift Servers websocket has failed",
	},
	WebsocketRetriableSendMessageFailure: {
		name:    "WebSocket Send Message Failed",
		message: "Sending Message to the Amazon GameLift Servers websocket has failed",
	},
	WebsocketSendMessageFailure: {
		name:    "WebSocket Send Message Failed",
		message: "Sending Message to the Amazon GameLift Servers websocket has failed",
	},
	WebsocketClosingError: {
		name:    "WebSocket close error",
		message: "An error has occurred in closing the connection",
	},
	UnknownException: {
		name:    "Unknown exception.",
		message: "Unknown exception.",
	},
}

// GameLiftError - Represents an error in a call to the server SDK for Amazon GameLift Servers.
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
	// Catch valid 4xx (client errors) status codes returned by websocket lambda
	// All other 4xx codes fallback to "bad request exception"
	if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		switch statusCode {
		case 400:
			return BadRequestException
		case 401:
			return UnauthorizedException
		case 403:
			return ForbiddenException
		case 404:
			return NotFoundException
		case 409:
			return ConflictException
		case 429:
			return TooManyRequestsException
		}
		return BadRequestException
	}

	// The websocket can return other error types, in this case classify it as an internal service exception
	return InternalServiceException
}

func GetErrorTypeFromMessage(errorMessage string) GameLiftErrorType {
	// Parse GameLiftError: ErrorType={%d} from the message
	errorType := UnknownException
	_, err := fmt.Sscanf(errorMessage, "[GameLiftError: ErrorType={%d}", &errorType)
	if err != nil {
		return UnknownException
	}
	return errorType
}
