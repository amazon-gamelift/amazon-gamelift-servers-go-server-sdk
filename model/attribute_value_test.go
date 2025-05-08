/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"bytes"
	"encoding/json"
	"math"
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
