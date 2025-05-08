/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"strconv"
	"strings"
)

// attributeType - unexported (private) data type, has a several predefined values see below.
type attributeType int

// Possible values for attribute types
const (
	None attributeType = iota
	String
	Double
	StringList
	StringDoubleMap
)

var attributeTypesStr = []string{"NONE", "STRING", "DOUBLE", "STRING_LIST", "STRING_DOUBLE_MAP"}

func (a *attributeType) String() string {
	n := int(*a)
	if n >= len(attributeTypesStr) {
		n = 0
	}
	return attributeTypesStr[n]
}

func (a *attributeType) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(a.String())), nil
}

func (a *attributeType) UnmarshalJSON(data []byte) error {
	origin, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	for i := range attributeTypesStr {
		if strings.EqualFold(attributeTypesStr[i], origin) {
			*a = attributeType(i)
			return nil
		}
	}
	*a = None
	return nil
}

// AttributeValue is the object lets you specify an attribute value using any of the valid data types:
//   - string,
//   - number,
//   - string array,
//   - data map.
//
// Each AttributeValue object can use only one of the available properties.
type AttributeValue struct {
	// The attribute data type.
	AttrType *attributeType `json:"AttrType"`
	// For number values, expressed as double.
	N float64 `json:"N,omitempty"`
	// For single string values. Maximum string length is 100 characters.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	S string `json:"S,omitempty"`
	// For a list of up to 100 strings. Maximum length for each string is 100 characters.
	// Duplicate values are not recognized; all occurrences of the repeated value after the first of a repeated value are ignored.
	// Length Constraints: Minimum length of 1. Maximum length of 1024.
	SL []string `json:"SL,omitempty"`
	// For a map of up to 10 data type:value pairs. Maximum length for each string value is 100 characters.
	// Key Length Constraints: Minimum length of 1. Maximum length of 1024.
	SDM map[string]float64 `json:"SDM,omitempty"`
}

// GetAttrType - return current attributeType.
//
//nolint:revive // Return an unexposed type is enough to use a predefined values [ String, Double, StringList, StringDoubleMap ]
func (a AttributeValue) GetAttrType() attributeType {
	return *a.AttrType
}

// MakeAttributeValue - create an AttributeValue by argument arg
//
//nolint:gosimple,gocritic // No need to check type assertion for arg in switch-case by type
func MakeAttributeValue(arg any) AttributeValue {
	switch arg.(type) {
	case float64:
		attrType := Double
		return AttributeValue{
			AttrType: &attrType,
			N:        arg.(float64),
		}
	case string:
		attrType := String
		return AttributeValue{
			AttrType: &attrType,
			S:        arg.(string),
		}
	case []string:
		attrType := StringList
		return AttributeValue{
			AttrType: &attrType,
			SL:       arg.([]string),
		}
	case map[string]float64:
		attrType := StringDoubleMap
		return AttributeValue{
			AttrType: &attrType,
			SDM:      arg.(map[string]float64),
		}
	case []interface{}:
		sl := arg.([]interface{})
		val := make([]string, 0, len(sl))
		for i := range sl {
			if str, ok := sl[i].(string); ok {
				val = append(val, str)
			}
		}
		attrType := StringList
		return AttributeValue{
			AttrType: &attrType,
			SL:       val,
		}
	case map[string]interface{}:
		sdm := arg.(map[string]interface{})
		val := make(map[string]float64, len(sdm))
		for key := range sdm {
			if f, ok := sdm[key].(float64); ok {
				val[key] = f
			}
		}
		attrType := StringDoubleMap
		return AttributeValue{
			AttrType: &attrType,
			SDM:      val,
		}
	}
	attrType := None
	return AttributeValue{AttrType: &attrType}
}
