// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ViBiOh/absto/pkg/model (interfaces: Storage)

// Package mocks is a generated GoMock package.
package mocks

import (
	io "io"
	reflect "reflect"
	time "time"

	model "github.com/ViBiOh/absto/pkg/model"
	gomock "github.com/golang/mock/gomock"
)

// Storage is a mock of Storage interface.
type Storage struct {
	ctrl     *gomock.Controller
	recorder *StorageMockRecorder
}

// StorageMockRecorder is the mock recorder for Storage.
type StorageMockRecorder struct {
	mock *Storage
}

// NewStorage creates a new mock instance.
func NewStorage(ctrl *gomock.Controller) *Storage {
	mock := &Storage{ctrl: ctrl}
	mock.recorder = &StorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Storage) EXPECT() *StorageMockRecorder {
	return m.recorder
}

// CreateDir mocks base method.
func (m *Storage) CreateDir(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDir", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateDir indicates an expected call of CreateDir.
func (mr *StorageMockRecorder) CreateDir(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDir", reflect.TypeOf((*Storage)(nil).CreateDir), arg0)
}

// Enabled mocks base method.
func (m *Storage) Enabled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enabled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Enabled indicates an expected call of Enabled.
func (mr *StorageMockRecorder) Enabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enabled", reflect.TypeOf((*Storage)(nil).Enabled))
}

// Info mocks base method.
func (m *Storage) Info(arg0 string) (model.Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Info", arg0)
	ret0, _ := ret[0].(model.Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Info indicates an expected call of Info.
func (mr *StorageMockRecorder) Info(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*Storage)(nil).Info), arg0)
}

// List mocks base method.
func (m *Storage) List(arg0 string) ([]model.Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0)
	ret0, _ := ret[0].([]model.Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *StorageMockRecorder) List(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*Storage)(nil).List), arg0)
}

// Name mocks base method.
func (m *Storage) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *StorageMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*Storage)(nil).Name))
}

// Path mocks base method.
func (m *Storage) Path(arg0 string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Path", arg0)
	ret0, _ := ret[0].(string)
	return ret0
}

// Path indicates an expected call of Path.
func (mr *StorageMockRecorder) Path(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Path", reflect.TypeOf((*Storage)(nil).Path), arg0)
}

// ReadFrom mocks base method.
func (m *Storage) ReadFrom(arg0 string) (io.ReadSeekCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFrom", arg0)
	ret0, _ := ret[0].(io.ReadSeekCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFrom indicates an expected call of ReadFrom.
func (mr *StorageMockRecorder) ReadFrom(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFrom", reflect.TypeOf((*Storage)(nil).ReadFrom), arg0)
}

// Remove mocks base method.
func (m *Storage) Remove(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove.
func (mr *StorageMockRecorder) Remove(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*Storage)(nil).Remove), arg0)
}

// Rename mocks base method.
func (m *Storage) Rename(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rename", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Rename indicates an expected call of Rename.
func (mr *StorageMockRecorder) Rename(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rename", reflect.TypeOf((*Storage)(nil).Rename), arg0, arg1)
}

// UpdateDate mocks base method.
func (m *Storage) UpdateDate(arg0 string, arg1 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDate", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateDate indicates an expected call of UpdateDate.
func (mr *StorageMockRecorder) UpdateDate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDate", reflect.TypeOf((*Storage)(nil).UpdateDate), arg0, arg1)
}

// Walk mocks base method.
func (m *Storage) Walk(arg0 string, arg1 func(model.Item) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Walk", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Walk indicates an expected call of Walk.
func (mr *StorageMockRecorder) Walk(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Walk", reflect.TypeOf((*Storage)(nil).Walk), arg0, arg1)
}

// WithIgnoreFn mocks base method.
func (m *Storage) WithIgnoreFn(arg0 func(model.Item) bool) model.Storage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithIgnoreFn", arg0)
	ret0, _ := ret[0].(model.Storage)
	return ret0
}

// WithIgnoreFn indicates an expected call of WithIgnoreFn.
func (mr *StorageMockRecorder) WithIgnoreFn(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithIgnoreFn", reflect.TypeOf((*Storage)(nil).WithIgnoreFn), arg0)
}

// WriteTo mocks base method.
func (m *Storage) WriteTo(arg0 string, arg1 io.Reader) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteTo", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteTo indicates an expected call of WriteTo.
func (mr *StorageMockRecorder) WriteTo(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteTo", reflect.TypeOf((*Storage)(nil).WriteTo), arg0, arg1)
}
