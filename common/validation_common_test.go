/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"regexp"
	"strings"
	"testing"
)

const (
	VALID_STRING = "validstring"
	PATTERN = `^[a-zA-Z0-9]+$`
	INVALID_STRING = "invalidstring!"
	FIELD_NAME = "FieldName"
	MIN_LENGTH = 1
	MAX_LENGTH = 100
	REQUIRED_ERROR = "FieldName is required."
	LENGTH_ERROR = "FieldName is invalid. Length must be between 1 and 100 characters."
	LENGTH_ERROR_NO_MAX = "FieldName is invalid. Length must be at least 2 characters."
	LENGTH_ERROR_TOO_SHORT = "FieldName is invalid. Length must be between 2 and 100 characters."
	OVERRIDE_ERROR = "FieldName is invalid because of some other reason."
	FORMAT_ERROR = "FieldName is invalid. Must match the pattern: ^[a-zA-Z0-9]+$."
)

var testRegex = regexp.MustCompile(PATTERN)

func TestValidateString(t *testing.T) {
	// Test valid string
	err := ValidateString(FIELD_NAME, VALID_STRING, testRegex, MIN_LENGTH, MAX_LENGTH, true, "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidateString_EmptyAndNotRequired(t *testing.T) {
	// Test string not passed and not required
	err := ValidateString(FIELD_NAME, "", testRegex, MIN_LENGTH, MAX_LENGTH, false, "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidateString_EmptyAndRequired_ThrowsException(t *testing.T) {
	// Test string not passed and required
	err := ValidateString(FIELD_NAME, "", testRegex, MIN_LENGTH, MAX_LENGTH, true, "")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	AssertContains(t, err.Error(), REQUIRED_ERROR)
}

func TestValidateString_TooLong_ThrowsException(t *testing.T) {
	// Test invalid string
	err := ValidateString(FIELD_NAME, strings.Repeat("a", MAX_LENGTH+1), testRegex, MIN_LENGTH, MAX_LENGTH, true, "")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	AssertContains(t, err.Error(), LENGTH_ERROR)
}

func TestValidateString_TooShort_ThrowsException(t *testing.T) {
	// Test invalid string
	err := ValidateString(FIELD_NAME, "a", testRegex, 2, MAX_LENGTH, true, "")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	AssertContains(t, err.Error(), LENGTH_ERROR_TOO_SHORT)
}

func TestValidateString_TooShortNoMax_ThrowsException(t *testing.T) {
	// Test invalid string
	err := ValidateString(FIELD_NAME, "a", testRegex, 2, MaxStringLengthNoLimit, true, "")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	AssertContains(t, err.Error(), LENGTH_ERROR_NO_MAX)
}

func TestValidateString_InvalidFormat_ThrowsException(t *testing.T) {
	// Test invalid string
	err := ValidateString(FIELD_NAME, INVALID_STRING, testRegex, MIN_LENGTH, MAX_LENGTH, true, "")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	AssertContains(t, err.Error(), FORMAT_ERROR)
}

func TestValidateString_InvalidFormat_ThrowsExceptionWithCustomErrorMessage(t *testing.T) {
	// Test invalid string
	err := ValidateString(FIELD_NAME, INVALID_STRING, testRegex, MIN_LENGTH, MAX_LENGTH, true, OVERRIDE_ERROR)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	AssertContains(t, err.Error(), OVERRIDE_ERROR)
}



