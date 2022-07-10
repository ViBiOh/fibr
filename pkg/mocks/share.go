// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ViBiOh/fibr/pkg/provider (interfaces: ShareManager)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	provider "github.com/ViBiOh/fibr/pkg/provider"
	gomock "github.com/golang/mock/gomock"
)

// Share is a mock of ShareManager interface.
type Share struct {
	ctrl     *gomock.Controller
	recorder *ShareMockRecorder
}

// ShareMockRecorder is the mock recorder for Share.
type ShareMockRecorder struct {
	mock *Share
}

// NewShare creates a new mock instance.
func NewShare(ctrl *gomock.Controller) *Share {
	mock := &Share{ctrl: ctrl}
	mock.recorder = &ShareMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Share) EXPECT() *ShareMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *Share) Create(arg0 context.Context, arg1 string, arg2, arg3 bool, arg4 string, arg5 bool, arg6 time.Duration) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *ShareMockRecorder) Create(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*Share)(nil).Create), arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// Delete mocks base method.
func (m *Share) Delete(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *ShareMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*Share)(nil).Delete), arg0, arg1)
}

// Get mocks base method.
func (m *Share) Get(arg0 string) provider.Share {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(provider.Share)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *ShareMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*Share)(nil).Get), arg0)
}

// List mocks base method.
func (m *Share) List() []provider.Share {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].([]provider.Share)
	return ret0
}

// List indicates an expected call of List.
func (mr *ShareMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*Share)(nil).List))
}
