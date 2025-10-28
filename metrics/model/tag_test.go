/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"strings"
	"testing"
)

func TestValidateTagKey(t *testing.T) {
	tests := []struct {
		key     string
		wantErr bool
	}{
		{"environment", false},
		{"server_type", false},
		{"server-type", false},
		{"server.type", false},
		{"server/type", false},
		{"server123", false},
		{"a", false},
		{strings.Repeat("a", MaxTagLength), false},

		{"", true},
		{"   ", true},
		{strings.Repeat("a", MaxTagLength+1), true},
		{"123server", true},
		{"_server", true},
		{"-server", true},
		{"server:type", true},
		{"server type", true},
		{"server@type", true},
	}

	for _, tt := range tests {
		err := ValidateTagKey(tt.key)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateTagKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
		}
	}
}

func TestValidateTagValue(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"production", false},
		{"", false},
		{"game_server", false},
		{"game:server", false},
		{"version123", false},
		{"app:version_1.2-3", false},
		{strings.Repeat("a", MaxTagLength), false},

		{strings.Repeat("a", MaxTagLength+1), true},
		{"production server", true},
		{"server@domain", true},
		{"server[1]", true},
		{"server\ttype", true},
	}

	for _, tt := range tests {
		err := ValidateTagValue(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateTagValue(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestTagCharacterValidation(t *testing.T) {
	keyValid := []rune{'a', 'Z', '0', '9', '_', '-', '.', '/'}
	keyInvalid := []rune{':', ' ', '@', '[', '\t'}

	for _, r := range keyValid {
		if !isValidTagKeyCharacter(r) {
			t.Errorf("isValidTagKeyCharacter(%c) should be true", r)
		}
	}

	for _, r := range keyInvalid {
		if isValidTagKeyCharacter(r) {
			t.Errorf("isValidTagKeyCharacter(%c) should be false", r)
		}
	}

	valueValid := []rune{'a', 'Z', '0', '9', '_', '-', '.', '/', ':'}
	valueInvalid := []rune{' ', '@', '[', '\t', '\n'}

	for _, r := range valueValid {
		if !isValidTagValueCharacter(r) {
			t.Errorf("isValidTagValueCharacter(%c) should be true", r)
		}
	}

	for _, r := range valueInvalid {
		if isValidTagValueCharacter(r) {
			t.Errorf("isValidTagValueCharacter(%c) should be false", r)
		}
	}
}
