/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import "sync/atomic"

// AtomicBool - is an atomic boolean value.
// The zero value is false.
// API compatible with atomic.Bool in golang 1.19+.
type AtomicBool struct {
	v uint32
}

// Load atomically loads and returns the value stored in x.
func (x *AtomicBool) Load() bool {
	return atomic.LoadUint32(&x.v) != 0
}

// Store atomically stores val into x.
func (x *AtomicBool) Store(val bool) {
	atomic.StoreUint32(&x.v, x.b32(val))
}

// CompareAndSwap executes the compare-and-swap operation for the boolean value x.
//
// The result of the operation indicate whether it performed the substitution.
func (x *AtomicBool) CompareAndSwap(oldValue, newValue bool) bool {
	return atomic.CompareAndSwapUint32(&x.v, x.b32(oldValue), x.b32(newValue))
}

func (x *AtomicBool) b32(val bool) uint32 {
	if val {
		return 1
	}
	return 0
}
