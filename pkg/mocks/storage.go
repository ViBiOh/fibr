// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ViBiOh/fibr/pkg/provider (interfaces: Storage)

// Package mocks is a generated GoMock package.
package mocks

import (
	io "io"
	reflect "reflect"
	time "time"

	provider "github.com/ViBiOh/fibr/pkg/provider"
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

// Info mocks base method.
func (m *Storage) Info(arg0 string) (provider.StorageItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Info", arg0)
	ret0, _ := ret[0].(provider.StorageItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Info indicates an expected call of Info.
func (mr *StorageMockRecorder) Info(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*Storage)(nil).Info), arg0)
}

// List mocks base method.
func (m *Storage) List(arg0 string) ([]provider.StorageItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0)
	ret0, _ := ret[0].([]provider.StorageItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *StorageMockRecorder) List(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*Storage)(nil).List), arg0)
}

// ReaderFrom mocks base method.
func (m *Storage) ReaderFrom(arg0 string) (provider.ReadSeekerCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReaderFrom", arg0)
	ret0, _ := ret[0].(provider.ReadSeekerCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReaderFrom indicates an expected call of ReaderFrom.
func (mr *StorageMockRecorder) ReaderFrom(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReaderFrom", reflect.TypeOf((*Storage)(nil).ReaderFrom), arg0)
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
func (m *Storage) Walk(arg0 string, arg1 func(provider.StorageItem, error) error) error {
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
func (m *Storage) WithIgnoreFn(arg0 func(provider.StorageItem) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WithIgnoreFn", arg0)
}

// WithIgnoreFn indicates an expected call of WithIgnoreFn.
func (mr *StorageMockRecorder) WithIgnoreFn(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithIgnoreFn", reflect.TypeOf((*Storage)(nil).WithIgnoreFn), arg0)
}

// WriterTo mocks base method.
func (m *Storage) WriterTo(arg0 string) (io.WriteCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriterTo", arg0)
	ret0, _ := ret[0].(io.WriteCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WriterTo indicates an expected call of WriterTo.
func (mr *StorageMockRecorder) WriterTo(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriterTo", reflect.TypeOf((*Storage)(nil).WriterTo), arg0)
}
