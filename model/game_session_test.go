/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"encoding/json"
	"testing"
)

func TestGameSession_WithStatus(t *testing.T) {
	var gameSession GameSession
	statuses := []gameSessionStatus{
		GameActive,
		GameNotSet,
		GameActivating,
		GameTerminated,
		GameTerminating,
	}
	for i := range statuses {
		gameSession = gameSession.WithStatus(statuses[i])
		data, err := json.Marshal(gameSession)
		if err != nil {
			t.Errorf("json marshal GameSession error: %s", err.Error())
			return
		}
		gameSession = GameSession{}
		err = json.Unmarshal(data, &gameSession)
		if err != nil {
			t.Errorf("json unmarshal GameSession error %s", err.Error())
			return
		}
		if gameSession.GetStatus() != statuses[i] {
			t.Errorf("statuses are not equal: expect %v but get %v", statuses[i], gameSession.GetStatus())
		}
	}
}
