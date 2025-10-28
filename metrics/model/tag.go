/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import (
	"strings"
	"unicode"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/common"
)

// MaxTagLength is the maximum character length of a tag.
const (
	MaxTagLength = 200
)

// ValidateTagKey validates a tag key according to DataDog specifications.
// Keys must start with a letter and cannot contain colons.
func ValidateTagKey(key string) error {
	if len(strings.TrimSpace(key)) == 0 {
		return common.NewGameLiftError(common.ValidationException, "", "tag key cannot be empty")
	}
	if len(key) > MaxTagLength {
		return common.NewGameLiftError(common.ValidationException, "", "tag key exceeds maximum allowed length")
	}
	if !unicode.IsLetter(rune(key[0])) {
		return common.NewGameLiftError(common.ValidationException, "", "tag key must start with a letter")
	}
	for i, r := range key {
		if i == 0 {
			continue
		}
		if !isValidTagKeyCharacter(r) {
			return common.NewGameLiftError(common.ValidationException, "", "tag key contains invalid characters. Only alphanumerics, underscores, minuses, periods, and slashes are allowed (no colons)")
		}
	}
	return nil
}

// ValidateTagValue validates a tag value according to DataDog specifications.
// Values can contain colons and most other characters.
func ValidateTagValue(value string) error {
	if len(value) > MaxTagLength {
		return common.NewGameLiftError(common.ValidationException, "", "tag value exceeds maximum allowed length")
	}
	// Values can be empty in DataDog tags.
	// Values have more relaxed character requirements and can contain colons.
	for _, r := range value {
		if !isValidTagValueCharacter(r) {
			return common.NewGameLiftError(common.ValidationException, "", "tag value contains invalid characters")
		}
	}
	return nil
}

// isValidTagKeyCharacter returns true if the rune is allowed in tag keys (no colons allowed).
func isValidTagKeyCharacter(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsDigit(r) ||
		r == '_' ||
		r == '-' ||
		r == '.' ||
		r == '/'
}

// isValidTagValueCharacter returns true if the rune is allowed in tag values (colons allowed).
func isValidTagValueCharacter(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsDigit(r) ||
		r == '_' ||
		r == '-' ||
		r == ':' ||
		r == '.' ||
		r == '/'
}
