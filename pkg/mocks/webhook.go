// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ViBiOh/fibr/pkg/provider (interfaces: WebhookManager)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	provider "github.com/ViBiOh/fibr/pkg/provider"
	gomock "github.com/golang/mock/gomock"
)

// Webhook is a mock of WebhookManager interface.
type Webhook struct {
	ctrl     *gomock.Controller
	recorder *WebhookMockRecorder
}

// WebhookMockRecorder is the mock recorder for Webhook.
type WebhookMockRecorder struct {
	mock *Webhook
}

// NewWebhook creates a new mock instance.
func NewWebhook(ctrl *gomock.Controller) *Webhook {
	mock := &Webhook{ctrl: ctrl}
	mock.recorder = &WebhookMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Webhook) EXPECT() *WebhookMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *Webhook) Create(arg0 string, arg1 bool, arg2 string, arg3 []provider.EventType) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *WebhookMockRecorder) Create(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*Webhook)(nil).Create), arg0, arg1, arg2, arg3)
}

// Delete mocks base method.
func (m *Webhook) Delete(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *WebhookMockRecorder) Delete(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*Webhook)(nil).Delete), arg0)
}

// Enabled mocks base method.
func (m *Webhook) Enabled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enabled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Enabled indicates an expected call of Enabled.
func (mr *WebhookMockRecorder) Enabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enabled", reflect.TypeOf((*Webhook)(nil).Enabled))
}

// List mocks base method.
func (m *Webhook) List() map[string]provider.Webhook {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].(map[string]provider.Webhook)
	return ret0
}

// List indicates an expected call of List.
func (mr *WebhookMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*Webhook)(nil).List))
}
