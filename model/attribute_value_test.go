/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestAttributeType_MarshalJSON(t *testing.T) {
	cases := map[attributeType][]byte{
		String:          []byte("\"STRING\""),
		Double:          []byte("\"DOUBLE\""),
		StringList:      []byte("\"STRING_LIST\""),
		StringDoubleMap: []byte("\"STRING_DOUBLE_MAP\""),
		None:            []byte("\"NONE\""),
	}

	for key := range cases {
		res, err := json.Marshal(&key)
		if err != nil {
			t.Errorf("json marshal attributeType error: %s", err.Error())
			return
		}
		if !bytes.Equal(res, cases[key]) {
			t.Errorf("expect %s but get %s", cases[key], res)
			return
		}
	}
}

func TestAttributeType_UnmarshalJSON(t *testing.T) {
	cases := map[attributeType][]byte{
		String:          []byte("\"STRING\""),
		Double:          []byte("\"DOUBLE\""),
		StringList:      []byte("\"STRING_LIST\""),
		StringDoubleMap: []byte("\"STRING_DOUBLE_MAP\""),
		None:            []byte("\"OBJECT\""),
	}

	for key, v := range cases {
		var val attributeType
		err := json.Unmarshal(v, &val)
		if err != nil {
			t.Errorf("json unmarshal attributeType error: %s", err.Error())
			return
		}
		if key != val {
			t.Errorf("expect %d but get %d", key, val)
			return
		}
	}
}

func TestMakeAttributeValue(t *testing.T) {
	cases := map[attributeType]any{
		String:          "Testing purpose",
		Double:          math.Pi,
		StringList:      []string{"Testing", "purpose"},
		StringDoubleMap: map[string]float64{"1": 1.0},
		None:            struct{ Val string }{Val: "Unsupported type"},
	}

	for key, val := range cases {
		attrVal := MakeAttributeValue(val)
		if attrVal.GetAttrType() != key {
			t.Errorf("invalide attribute type, expect %v but get %v", key, attrVal.GetAttrType())
			return
		}
	}
}

func TestMarshalAttributeValue(t *testing.T) {
	var allCases []map[attributeType]any
	cases1 := map[attributeType]any{
		String:          "",
		Double:          0.0,
		StringList:      []string{},
		StringDoubleMap: map[string]float64{},
		None:            struct{ Val string }{Val: "Unsupported type"},
	}
	cases2 := map[attributeType]any{
		String:          "Testing purpose",
		Double:          10.3,
		StringList:      []string{"Testing", "purpose"},
		StringDoubleMap: map[string]float64{"1": 1.0},
		None:            struct{ Val string }{Val: "Unsupported type"},
	}
	allCases = append(allCases, cases1, cases2)
	for _, cases := range allCases {
		checkmarshaledData(cases, t)
	}
}

func checkmarshaledData(cases map[attributeType]any, t *testing.T) {
	for key, val := range cases {
		attrVal := MakeAttributeValue(val)
		if attrVal.GetAttrType() != key {
			t.Errorf("invalide attribute type, expect %v but get %v", key, attrVal.GetAttrType())
			return
		}
		marshaled, err := json.Marshal(attrVal)
		if err != nil {
			t.Errorf("error marshaling attribute type %v", attrVal.GetAttrType())
			return
		}
		if attrVal.GetAttrType() == Double && getNumFieldsInMarshaledData(string(marshaled)) != 1 {
			t.Errorf("error marshaling attribute type Double, field N and only field N is supposed to be populated, populated fields count is %d", getNumFieldsInMarshaledData(string(marshaled)))
			return
		}
		if attrVal.GetAttrType() == None && getNumFieldsInMarshaledData(string(marshaled)) != 0 {
			t.Errorf("error marshaling attribute type None, field N, S, SL, or SDM is not supposed to be populated")
			return
		}
		if attrVal.GetAttrType() == String || attrVal.GetAttrType() == StringList || attrVal.GetAttrType() == StringDoubleMap {
			// if the String, SL, or SDM is empty, the corresponding S, SL, or SDM field could be omitted
			if getNumFieldsInMarshaledData(string(marshaled)) > 1 {
				t.Errorf("error marshaling attribute type %v, more than one fields of N, S, SL, or SDM are populated", attrVal.GetAttrType())
				return
			}
		}
	}
}

func getNumFieldsInMarshaledData(marshaledString string) int {
	var count = 0
	if strings.Contains(marshaledString, "\"S\":") {
		count += 1
	}
	if strings.Contains(marshaledString, "\"N\":") {
		count += 1
	}
	if strings.Contains(marshaledString, "\"SL\":") {
		count += 1
	}
	if strings.Contains(marshaledString, "\"SDM\":") {
		count += 1
	}
	return count
}
