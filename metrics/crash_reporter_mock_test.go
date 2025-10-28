/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package metrics

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCrashReporter is a mock of CrashReporter interface.
type MockCrashReporter struct {
	ctrl     *gomock.Controller
	recorder *MockCrashReporterMockRecorder
}

// MockCrashReporterMockRecorder is the mock recorder for MockCrashReporter.
type MockCrashReporterMockRecorder struct {
	mock *MockCrashReporter
}

// NewMockCrashReporter creates a new mock instance.
func NewMockCrashReporter(ctrl *gomock.Controller) *MockCrashReporter {
	mock := &MockCrashReporter{ctrl: ctrl}
	mock.recorder = &MockCrashReporterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCrashReporter) EXPECT() *MockCrashReporterMockRecorder {
	return m.recorder
}

// DeregisterProcess mocks base method.
func (m *MockCrashReporter) DeregisterProcess() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeregisterProcess")
	ret0, _ := ret[0].(error)
	return ret0
}

// DeregisterProcess indicates an expected call of DeregisterProcess.
func (mr *MockCrashReporterMockRecorder) DeregisterProcess() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeregisterProcess", reflect.TypeOf((*MockCrashReporter)(nil).DeregisterProcess))
}

// RegisterProcess mocks base method.
func (m *MockCrashReporter) RegisterProcess() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterProcess")
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterProcess indicates an expected call of RegisterProcess.
func (mr *MockCrashReporterMockRecorder) RegisterProcess() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterProcess", reflect.TypeOf((*MockCrashReporter)(nil).RegisterProcess))
}

// TagGameSession mocks base method.
func (m *MockCrashReporter) TagGameSession(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TagGameSession", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// TagGameSession indicates an expected call of TagGameSession.
func (mr *MockCrashReporterMockRecorder) TagGameSession(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TagGameSession", reflect.TypeOf((*MockCrashReporter)(nil).TagGameSession), arg0)
}
