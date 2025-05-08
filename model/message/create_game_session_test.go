/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package message

import (
	"encoding/json"
	"testing"
)

func TestNewGameSession(t *testing.T) {
	const gamesessionJSON = `{
		"Action": "CreateGameSession",
		"MaximumPlayerSessionCount":1000,
		"Port": 1122,
		"IpAddress": "test.com",
		"GameSessionId": "game_session_id",
		"GameSessionName": "game_session_name",
		"GameSessionData": "game_session_data",
		"MatchmakerData": "{}",
		"GameProperties": {
			"property_1": "value_1",
			"property_2": "value_2",
			"property_3": "value_3",
			"property_4": "value_4"
		}
	}`
	var createGameSessionMsg CreateGameSessionMessage
	if err := json.Unmarshal([]byte(gamesessionJSON), &createGameSessionMsg); err != nil {
		t.Fatalf("Error when try parse createGameSessionMessage: %s", err.Error())
	}
	if createGameSessionMsg.Action != "CreateGameSession" {
		t.Fatalf("Incorrect Action: expect %s, but get %s", "CreateGameSession", createGameSessionMsg.Action)
	}
	gameSession := NewGameSession(&createGameSessionMsg)
	if gameSession.MaximumPlayerSessionCount != 1000 || gameSession.Port != 1122 {
		t.Fatalf("Incorrect parse integer values from the json: %s, to the object %v", gamesessionJSON, gameSession)
	}
	if gameSession.IPAddress != "test.com" ||
		gameSession.GameSessionID != "game_session_id" ||
		gameSession.Name != "game_session_name" ||
		gameSession.GameSessionData != "game_session_data" ||
		gameSession.MatchmakerData != "{}" {
		t.Fatalf("Incorrect parse string values from the json: %s, to the object %v", gamesessionJSON, gameSession)
	}
	for prop, value := range gameSession.GameProperties {
		val, ok := createGameSessionMsg.GameProperties[prop]
		if !ok {
			t.Fatalf("Undefined %s property in CreateGameSessionMessage", prop)
		}
		if val != value {
			t.Fatalf("Incorrect parse value in property %s, expect: %s, but get: %s", prop, val, value)
		}
	}
}
