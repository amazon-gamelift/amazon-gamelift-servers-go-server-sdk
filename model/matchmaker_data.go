/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"encoding/json"
)

type MatchmakerData struct {
	// The unique identifiers for this group of profiles that match.
	// Length Constraints: Minimum length of 1. Maximum length of 255.
	MatchID string
	// Unique identifier, in the form of an ARN, for the matchmaker to use for this request.
	MatchmakingConfigurationArn string
	// A set of data representing all players who are currently in the game session.
	Players []Player
	// A unique identifier for a matchmaking ticket. If no ticket ID is specified here,
	// Amazon GameLift will generate one in the form of a UUID.
	// Length Constraints: Maximum length of 128.
	AutoBackfillTicketID string
}

type attributeMatchMakerData struct {
	AttributeType  string      `json:"attributeType"`
	ValueAttribute interface{} `json:"valueAttribute"`
}

type playerMatchmakerData struct {
	// A unique identifier for a player
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	PlayerID string `json:"playerId"`
	// A collection of key:value pairs containing player information for use in matchmaking.
	// Player attribute keys must match the playerAttributes used in a matchmaking rule set.
	// Example: "PlayerAttributes": {"skill": {"N": "23"}, "gameMode": {"S": "deathmatch"}}.
	// You can provide up to 10 PlayerAttributes.
	// Type: String to AttributeValue object map
	// Key Length Constraints: Minimum length of 1. Maximum length of 1024.
	AttributeValue map[string]attributeMatchMakerData `json:"attributes"`
}

type teamMatchmakerData struct {
	// Name of the team that the player is assigned to in a match.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	Name string `json:"name"`
	// A set of data representing players matchmaker data.
	Players []playerMatchmakerData `json:"players"`
}

type matchmakerDataOriginal struct {
	// The unique identifiers for this group of profiles that match.
	// Length Constraints: Minimum length of 1. Maximum length of 255.
	MatchID string `json:"matchId"`
	// Unique identifier, in the form of an ARN, for the matchmaker to use for this request.
	MatchmakingConfigurationArn string `json:"matchmakingConfigurationArn"`
	// A set of data representing teams matchmaker data.
	Teams []teamMatchmakerData `json:"teams"`
	// A unique identifier for a matchmaking ticket. If no ticket ID is specified here,
	// Amazon GameLift will generate one in the form of a UUID.
	// Length Constraints: Maximum length of 128.
	AutoBackfillTicketID string `json:"autoBackfillTicketId"`
}

func (m *MatchmakerData) MarshalJSON() ([]byte, error) {
	var origin matchmakerDataOriginal
	origin.fromMatchmakerData(m)

	return json.Marshal(origin)
}

func (m *MatchmakerData) UnmarshalJSON(data []byte) error {
	var matchmaker matchmakerDataOriginal

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &matchmaker); err != nil {
		return err
	}
	*m = matchmaker.toMatchmakerData()

	return nil
}

func fromAttributesValue(a AttributeValue) attributeMatchMakerData {
	switch *a.AttrType {
	case String:
		return attributeMatchMakerData{
			AttributeType:  "STRING",
			ValueAttribute: a.S,
		}
	case Double:
		return attributeMatchMakerData{
			AttributeType:  "DOUBLE",
			ValueAttribute: a.N,
		}
	case StringList:
		return attributeMatchMakerData{
			AttributeType:  "STRING_LIST",
			ValueAttribute: a.SL,
		}
	case StringDoubleMap:
		return attributeMatchMakerData{
			AttributeType:  "STRING_DOUBLE_MAP",
			ValueAttribute: a.SDM,
		}
	}
	return attributeMatchMakerData{
		AttributeType: "NONE",
	}
}

func (mo *matchmakerDataOriginal) fromMatchmakerData(m *MatchmakerData) {
	mo.MatchID = m.MatchID
	mo.MatchmakingConfigurationArn = m.MatchmakingConfigurationArn
	mo.AutoBackfillTicketID = m.AutoBackfillTicketID
	teams := make(map[string][]Player)
	for i := range m.Players {
		if _, ok := teams[m.Players[i].Team]; !ok {
			teams[m.Players[i].Team] = make([]Player, 0, 1)
		}
		teams[m.Players[i].Team] = append(teams[m.Players[i].Team], m.Players[i])
	}
	for name, players := range teams {
		var pMatchmakerData []playerMatchmakerData
		for _, singlePlayer := range players {
			var attributes = make(map[string]attributeMatchMakerData)
			for k, v := range singlePlayer.PlayerAttributes {
				attributes[k] = fromAttributesValue(v)
			}
			pMatchmakerData = append(pMatchmakerData, playerMatchmakerData{
				PlayerID:       singlePlayer.PlayerID,
				AttributeValue: attributes,
			})
		}
		mo.Teams = append(mo.Teams, teamMatchmakerData{
			Name:    name,
			Players: pMatchmakerData,
		})
	}
}

func (mo *matchmakerDataOriginal) toMatchmakerData() MatchmakerData {
	var data = MatchmakerData{
		MatchID:                     mo.MatchID,
		MatchmakingConfigurationArn: mo.MatchmakingConfigurationArn,
		AutoBackfillTicketID:        mo.AutoBackfillTicketID,
	}
	for _, team := range mo.Teams {
		for _, player := range team.Players {
			var playerAttributes = make(map[string]AttributeValue)
			for k, v := range player.AttributeValue {
				playerAttributes[k] = MakeAttributeValue(v.ValueAttribute)
			}
			data.Players = append(data.Players, Player{
				Team:             team.Name,
				PlayerID:         player.PlayerID,
				PlayerAttributes: playerAttributes,
			})
		}
	}
	return data
}
