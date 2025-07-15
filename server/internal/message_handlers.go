/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package internal

import "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/model"

// IGameLiftMessageHandler - async messages handlers from game server process or APIG.
//
//go:generate mockgen -destination ../internal/mock/handlers.go -package=mock . IGameLiftMessageHandler
type IGameLiftMessageHandler interface {
	OnStartGameSession(gameSession *model.GameSession)
	OnUpdateGameSession(
		gameSession *model.GameSession,
		updateReason *model.UpdateReason,
		backfillTicketID string,
	)
	OnTerminateProcess(terminationTime int64)
	OnRefreshConnection(refreshConnectionEndpoint, authToken string)
}
