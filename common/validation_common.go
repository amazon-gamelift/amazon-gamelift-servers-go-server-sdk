/*
* Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
* SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"regexp"
)

// Generic validation for string input fields.
func ValidateString(fieldName string, input string, regex *regexp.Regexp, minLength int, maxLength int, required bool, overrideErrorMessage string) error {
	if len(input) == 0 {
		if required {
			return NewGameLiftError(ValidationException, "", fmt.Sprintf("%s is required.", fieldName))
		}
	} else {
		if len(input) < minLength || (maxLength != MaxStringLengthNoLimit && len(input) > maxLength) {
			if maxLength == MaxStringLengthNoLimit {
				return NewGameLiftError(ValidationException, "", fmt.Sprintf("%s is invalid. Length must be at least %d characters.", fieldName, minLength))
			}
			return NewGameLiftError(ValidationException, "", fmt.Sprintf("%s is invalid. Length must be between %d and %d characters.", fieldName,
				minLength, maxLength))
		}
		if regex != nil && !regex.MatchString(input) {
			// override for regex error message
			if overrideErrorMessage != "" {
				return NewGameLiftError(ValidationException, "", overrideErrorMessage)
			}
			return NewGameLiftError(ValidationException, "", fmt.Sprintf("%s is invalid. Must match the pattern: %s.", fieldName, regex.String()))
		}
	}
	return nil
}