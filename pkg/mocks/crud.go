// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ViBiOh/fibr/pkg/provider (interfaces: Crud)

// Package mocks is a generated GoMock package.
package mocks

import (
	multipart "mime/multipart"
	http "net/http"
	reflect "reflect"

	provider "github.com/ViBiOh/fibr/pkg/provider"
	renderer "github.com/ViBiOh/httputils/v4/pkg/renderer"
	gomock "github.com/golang/mock/gomock"
)

// Crud is a mock of Crud interface.
type Crud struct {
	ctrl     *gomock.Controller
	recorder *CrudMockRecorder
}

// CrudMockRecorder is the mock recorder for Crud.
type CrudMockRecorder struct {
	mock *Crud
}

// NewCrud creates a new mock instance.
func NewCrud(ctrl *gomock.Controller) *Crud {
	mock := &Crud{ctrl: ctrl}
	mock.recorder = &CrudMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Crud) EXPECT() *CrudMockRecorder {
	return m.recorder
}

// Browser mocks base method.
func (m *Crud) Browser(arg0 http.ResponseWriter, arg1 provider.Request, arg2 provider.StorageItem, arg3 renderer.Message) (string, int, map[string]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Browser", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(map[string]interface{})
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Browser indicates an expected call of Browser.
func (mr *CrudMockRecorder) Browser(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Browser", reflect.TypeOf((*Crud)(nil).Browser), arg0, arg1, arg2, arg3)
}

// Create mocks base method.
func (m *Crud) Create(arg0 http.ResponseWriter, arg1 *http.Request, arg2 provider.Request) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Create", arg0, arg1, arg2)
}

// Create indicates an expected call of Create.
func (mr *CrudMockRecorder) Create(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*Crud)(nil).Create), arg0, arg1, arg2)
}

// Delete mocks base method.
func (m *Crud) Delete(arg0 http.ResponseWriter, arg1 *http.Request, arg2 provider.Request) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Delete", arg0, arg1, arg2)
}

// Delete indicates an expected call of Delete.
func (mr *CrudMockRecorder) Delete(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*Crud)(nil).Delete), arg0, arg1, arg2)
}

// Get mocks base method.
func (m *Crud) Get(arg0 http.ResponseWriter, arg1 *http.Request, arg2 provider.Request) (string, int, map[string]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(map[string]interface{})
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Get indicates an expected call of Get.
func (mr *CrudMockRecorder) Get(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*Crud)(nil).Get), arg0, arg1, arg2)
}

// List mocks base method.
func (m *Crud) List(arg0 http.ResponseWriter, arg1 provider.Request, arg2 renderer.Message) (string, int, map[string]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(map[string]interface{})
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// List indicates an expected call of List.
func (mr *CrudMockRecorder) List(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*Crud)(nil).List), arg0, arg1, arg2)
}

// Post mocks base method.
func (m *Crud) Post(arg0 http.ResponseWriter, arg1 *http.Request, arg2 provider.Request) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Post", arg0, arg1, arg2)
}

// Post indicates an expected call of Post.
func (mr *CrudMockRecorder) Post(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Post", reflect.TypeOf((*Crud)(nil).Post), arg0, arg1, arg2)
}

// Rename mocks base method.
func (m *Crud) Rename(arg0 http.ResponseWriter, arg1 *http.Request, arg2 provider.Request) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Rename", arg0, arg1, arg2)
}

// Rename indicates an expected call of Rename.
func (mr *CrudMockRecorder) Rename(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rename", reflect.TypeOf((*Crud)(nil).Rename), arg0, arg1, arg2)
}

// Start mocks base method.
func (m *Crud) Start(arg0 <-chan struct{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start", arg0)
}

// Start indicates an expected call of Start.
func (mr *CrudMockRecorder) Start(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*Crud)(nil).Start), arg0)
}

// Upload mocks base method.
func (m *Crud) Upload(arg0 http.ResponseWriter, arg1 *http.Request, arg2 provider.Request, arg3 map[string]string, arg4 *multipart.Part) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Upload", arg0, arg1, arg2, arg3, arg4)
}

// Upload indicates an expected call of Upload.
func (mr *CrudMockRecorder) Upload(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upload", reflect.TypeOf((*Crud)(nil).Upload), arg0, arg1, arg2, arg3, arg4)
}