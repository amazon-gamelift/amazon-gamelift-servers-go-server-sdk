/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

var attrValue = []attributeType{None, String, Double, StringList, StringDoubleMap}

var matchMakerData = MatchmakerData{
	MatchID:                     "0123456789",
	MatchmakingConfigurationArn: "TestMatchMakingConfigurationArn",
	AutoBackfillTicketID:        "TestAutoBackfillTicketID",
	BackfillMode:                BackFillModeAutomatic,
	Players: []Player{
		{
			PlayerID: "TestPlayerID_1",
			Team:     "attacker",
			PlayerAttributes: map[string]AttributeValue{
				"none": {
					AttrType: &attrValue[0],
				},
				"gamemod": {
					AttrType: &attrValue[1],
					S:        "Deathmatch",
				},
				"kds": {
					AttrType: &attrValue[2],
					N:        21.65,
				},
				"achievements": {
					AttrType: &attrValue[3],
					SL:       []string{"Double kill", "Triple kill", "Ultra kill", "Rampage"},
				},
				"skill": {
					AttrType: &attrValue[4],
					SDM:      map[string]float64{"Body": 10.0, "Mind": 12.0, "Heart": 15.0, "Soul": 33.0},
				},
			},
		},
		{
			PlayerID: "TestPlayerID_2",
			Team:     "defender",
			PlayerAttributes: map[string]AttributeValue{
				"none": {
					AttrType: &attrValue[0],
				},
				"gamemod": {
					AttrType: &attrValue[1],
					S:        "Deathmatch",
				},
				"kds": {
					AttrType: &attrValue[2],
					N:        10.82,
				},
				"achievements": {
					AttrType: &attrValue[3],
					SL:       []string{"Team Shield", "The Rock", "Double Kill", "Healer"},
				},
				"skill": {
					AttrType: &attrValue[4],
					SDM:      map[string]float64{"Body": 11.0, "Mind": 12.0, "Heart": 11.0, "Soul": 40.0},
				},
			},
		},
		{
			PlayerID: "TestPlayerID_3",
			Team:     "defender",
			PlayerAttributes: map[string]AttributeValue{
				"none": {
					AttrType: &attrValue[0],
				},
				"gamemod": {
					AttrType: &attrValue[1],
					S:        "Health",
				},
				"kds": {
					AttrType: &attrValue[2],
					N:        33.5,
				},
				"achievements": {
					AttrType: &attrValue[3],
					SL:       []string{"Defence", "Double Defence", "Ultra defence", "Rampage"},
				},
				"skill": {
					AttrType: &attrValue[4],
					SDM:      map[string]float64{"Body": 10.0, "Mind": 22.0, "Heart": 35.0, "Soul": 103.0},
				},
			},
		},
	},
}

var mDataOriginal = matchmakerDataOriginal{
	MatchID:                     "0123456789",
	MatchmakingConfigurationArn: "TestMatchMakingConfigurationArn",
	AutoBackfillTicketID:        "TestAutoBackfillTicketID",
	BackfillMode:                "AUTOMATIC",
	Teams: []teamMatchmakerData{
		{
			Name: "attacker",
			Players: []playerMatchmakerData{
				{
					PlayerID: "TestPlayerID_1",
					AttributeValue: map[string]attributeMatchMakerData{
						"none": {
							AttributeType: "NONE",
						},
						"gamemod": {
							AttributeType:  "STRING",
							ValueAttribute: "Deathmatch",
						},
						"kds": {
							AttributeType:  "DOUBLE",
							ValueAttribute: 21.65,
						},
						"achievements": {
							AttributeType:  "STRING_LIST",
							ValueAttribute: []string{"Double kill", "Triple kill", "Ultra kill", "Rampage"},
						},
						"skill": {
							AttributeType:  "STRING_DOUBLE_MAP",
							ValueAttribute: map[string]float64{"Body": 10.0, "Mind": 12.0, "Heart": 15.0, "Soul": 33.0},
						},
					},
				},
			},
		},
		{
			Name: "defender",
			Players: []playerMatchmakerData{
				{
					PlayerID: "TestPlayerID_2",
					AttributeValue: map[string]attributeMatchMakerData{
						"none": {
							AttributeType: "NONE",
						},
						"gamemod": {
							AttributeType:  "STRING",
							ValueAttribute: "Deathmatch",
						},
						"kds": {
							AttributeType:  "DOUBLE",
							ValueAttribute: 10.82,
						},
						"achievements": {
							AttributeType:  "STRING_LIST",
							ValueAttribute: []string{"Team Shield", "The Rock", "Double Kill", "Healer"},
						},
						"skill": {
							AttributeType:  "STRING_DOUBLE_MAP",
							ValueAttribute: map[string]float64{"Body": 11.0, "Mind": 12.0, "Heart": 11.0, "Soul": 40.0},
						},
					},
				},
				{
					PlayerID: "TestPlayerID_3",
					AttributeValue: map[string]attributeMatchMakerData{
						"none": {
							AttributeType: "NONE",
						},
						"gamemod": {
							AttributeType:  "STRING",
							ValueAttribute: "Health",
						},
						"kds": {
							AttributeType:  "DOUBLE",
							ValueAttribute: 33.5,
						},
						"achievements": {
							AttributeType:  "STRING_LIST",
							ValueAttribute: []string{"Defence", "Double Defence", "Ultra defence", "Rampage"},
						},
						"skill": {
							AttributeType:  "STRING_DOUBLE_MAP",
							ValueAttribute: map[string]float64{"Body": 10.0, "Mind": 22.0, "Heart": 35.0, "Soul": 103.0},
						},
					},
				},
			},
		},
	},
}

func getTeam(teams []teamMatchmakerData, name string) (*teamMatchmakerData, error) {
	for i := range teams {
		if teams[i].Name == name {
			return &teams[i], nil
		}
	}
	return nil, fmt.Errorf("undefine team with name %s", name)
}

func getPlayer(players []playerMatchmakerData, id string) (*playerMatchmakerData, error) {
	for i := range players {
		if players[i].PlayerID == id {
			return &players[i], nil
		}
	}
	return nil, fmt.Errorf("undefine player with id %s", id)
}

func TestMatchmakerData_MarshalJSON(t *testing.T) {
	marshaledData, err := matchMakerData.MarshalJSON()
	if err != nil {
		t.Errorf("json marshal matchMakedData error: %s", err.Error())
		return
	}

	var matchmakerData matchmakerDataOriginal
	if err := json.Unmarshal(marshaledData, &matchmakerData); err != nil {
		t.Errorf("json unmarshaling to matchmakerDataOriginal fail: %s", err)
		return
	}
	expectMatchmakerData := mDataOriginal
	teamsGot := matchmakerData.Teams
	teamsOriginal := expectMatchmakerData.Teams
	if len(teamsOriginal) != len(teamsGot) {
		t.Fatalf("Teams length are not equal, expect: %d, but get %d", len(teamsOriginal), len(teamsGot))
	}
	// Compare matchmaker data without team at first
	expectMatchmakerData.Teams = nil
	matchmakerData.Teams = nil
	if !reflect.DeepEqual(expectMatchmakerData, matchmakerData) {
		t.Errorf("\nexpect  %v \nbut get %v", expectMatchmakerData, matchmakerData)
		return
	}
	// Compare teams between original and marshaled data
	for _, originTeam := range teamsOriginal {
		gotTeam, err := getTeam(teamsGot, originTeam.Name)
		if err != nil {
			t.Fatalf("error when try get team %s", err)
		}
		if len(gotTeam.Players) != len(originTeam.Players) {
			t.Fatalf("Players length in team %s are not equal, expect: %d, but get %d",
				originTeam.Name,
				len(originTeam.Players),
				len(gotTeam.Players),
			)
		}
		// Compare players in each team between original and marshaled data
		for _, originPlayer := range originTeam.Players {
			player, err := getPlayer(gotTeam.Players, originPlayer.PlayerID)
			if err != nil {
				t.Fatalf("error when try get player %s", err)
			}
			if len(player.AttributeValue) != len(originPlayer.AttributeValue) {
				t.Fatalf("incorrect length of attribute values expect: %d but get %d",
					len(originPlayer.AttributeValue),
					len(player.AttributeValue),
				)
			}
			// Compare attributes for each player on each team between original and marshaled data
			for key, attr := range originPlayer.AttributeValue {
				gotAttr, ok := player.AttributeValue[key]
				if !ok {
					t.Fatalf("undefine attribute value for %s", key)
				}
				if attr.AttributeType != gotAttr.AttributeType {
					t.Fatalf("expect: \"%s\", but get: \"%s\"", attr.AttributeType, gotAttr.AttributeType)
				}
			}
		}
	}
}

func TestMatchmakerData_UnmarshalJSON(t *testing.T) {
	marshaledData, err := json.Marshal(mDataOriginal)
	if err != nil {
		t.Errorf("json marshal matchmakerDataOriginal error: %s", err.Error())
		return
	}

	var unmarshalJsonOutput MatchmakerData
	err = unmarshalJsonOutput.UnmarshalJSON(marshaledData)
	if err != nil {
		t.Fatalf("json unmarshal matchMakedData error: %s", err.Error())
	}

	if !reflect.DeepEqual(unmarshalJsonOutput, matchMakerData) {
		t.Fatalf("\nexpect  %v \nbut get %v", matchMakerData, unmarshalJsonOutput)
	}
}
