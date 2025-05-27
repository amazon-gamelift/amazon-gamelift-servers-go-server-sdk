/*
* Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
* SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"github.com/golang/mock/gomock"
	"strings"
	"testing"
)

func AssertEqual(t *testing.T, expected interface{}, actual interface{}) {
	if expected != actual {
		t.Fatalf("Expected %v but got %v", expected, actual)
	}
}

func AssertContains(t *testing.T, s string, substr string) {
	if !strings.Contains(s, substr) {
		t.Fatalf("Expected %v to contain %v", s, substr)
	}
}

// MockIsPresentAnd - returns an intermediate gomock.Matcher that passes when the provided matcher passes
// while also having the actual value exist (is not empty). This matcher is not necessarily useful by itself
// and should be used to build more useful matchers. See MockEquals for a useful example.
func MockIsPresentAnd(matcher gomock.Matcher) gomock.Matcher {
	return gomock.All(
		gomock.Not(gomock.Len(0)),
		matcher)
}

// MockEquals - returns a gomock.Matcher that passes when the actual value matches the expected value
// and the value is actually present.
//
// This matcher is an improved version of gomock.Eq() because it guarantees that the test didn't mistakenly
// provide an empty value.
func MockEquals(requiredValue string) gomock.Matcher {
	return MockIsPresentAnd(gomock.Eq(requiredValue))
}

// MockNoneOf - returns a gomock.Matcher that passes if the actual value doesn't match any of the
// provided unexpected values but also confirm that the actual value is still present.
//
// Useful when confirming a unique value was generated instead of using input arguments.
func MockNoneOf(unexpectedValues ...string) gomock.Matcher {
	var matchers []gomock.Matcher
	for _, value := range unexpectedValues {
		matchers = append(matchers, gomock.Not(gomock.Eq(value)))
	}
	return MockIsPresentAnd(gomock.All(matchers...))
}

// MockStringMapContainsExpectedValue - returns a gomock.Matcher for a map[string]string value. This matcher passes if
// there exists a (key, value) pair in the actual where the value contains the expected value.
func MockStringMapContainsExpectedValue(expectedValue string) gomock.Matcher {
	return &StringMapContainsExpectedValueMatcher{ExpectedValue: expectedValue}
}

type StringMapContainsExpectedValueMatcher struct {
	ExpectedValue string
}

func (m *StringMapContainsExpectedValueMatcher) Matches(x interface{}) bool {
	if x == nil {
		return false
	}
	mapObj, ok := x.(map[string]string)
	if !ok {
		return false
	}
	for _, value := range mapObj {
		if strings.Contains(value, m.ExpectedValue) {
			return true
		}
	}
	return false
}

func (m *StringMapContainsExpectedValueMatcher) String() string {
	return "map contains value " + m.ExpectedValue
}
