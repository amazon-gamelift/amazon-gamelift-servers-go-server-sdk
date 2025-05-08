/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"testing"
)

func TestNewGameLiftError(t *testing.T) {
	for errType, desc := range errorMessages {
		err, ok := NewGameLiftError(errType, "", "").(*GameLiftError)
		if !ok {
			t.Fatal("Incorrect error type from the function NewGameLiftError")
		}
		if err.getNameOrDefaultForErrorType() != desc.name {
			t.Fatalf("Incorrect error name, expect: \"%s\" but get: \"%s\"",
				desc.name,
				err.getNameOrDefaultForErrorType(),
			)
		}
		if err.getMessageOrDefaultForErrorType() != desc.message {
			t.Fatalf("Incorrect error message, expect: \"%s\" but get: \"%s\"",
				desc.message,
				err.getMessageOrDefaultForErrorType(),
			)
		}

		err, ok = NewGameLiftError(errType, "Test Name", "Test Message").(*GameLiftError)
		if !ok {
			t.Fatal("Incorrect error type from the function NewGameLiftError")
		}

		if err.getNameOrDefaultForErrorType() != "Test Name" {
			t.Fatalf("Incorrect error name, expect: \"Test Name\",but get: \"%s\"",
				err.getNameOrDefaultForErrorType(),
			)
		}
		if err.getMessageOrDefaultForErrorType() != "Test Message" {
			t.Fatalf("Incorrect error message, expect: \"Test Message\", but get: \"%s\"",
				err.getMessageOrDefaultForErrorType(),
			)
		}
		parsedErrorType := GetErrorTypeFromMessage(err.Error())
		if parsedErrorType != errType {
			t.Fatalf("Incorrect error type, expect: \"%d\", but get: \"%d\"",
				errType,
				parsedErrorType,
			)
		}


	}
}
