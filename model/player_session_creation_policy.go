/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package model

import "strconv"

// PlayerSessionCreationPolicy - used in UpdatePlayerSessionCreationPolicy
type PlayerSessionCreationPolicy int

// Possible PlayerSessionCreationPolicy values
const (
	NotSet PlayerSessionCreationPolicy = iota
	DenyAll
	AcceptAll
)

var playerSessionPolicyStr = []string{"NOT_SET", "DENY_ALL", "ACCEPT_ALL"}

func (p *PlayerSessionCreationPolicy) String() string {
	n := int(*p)
	if n >= len(playerSessionPolicyStr) {
		n = 0
	}
	return playerSessionPolicyStr[n]
}

func (p *PlayerSessionCreationPolicy) ToPlayerSessionPolicy(s string) {
	for i := range playerSessionPolicyStr {
		if playerSessionPolicyStr[i] == s {
			*p = PlayerSessionCreationPolicy(i)
			return
		}
	}
	*p = NotSet
}

func (p *PlayerSessionCreationPolicy) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(p.String())), nil
}

func (p *PlayerSessionCreationPolicy) UnmarshalJSON(data []byte) error {
	origin, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	p.ToPlayerSessionPolicy(origin)
	return nil
}
