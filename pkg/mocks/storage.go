// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ViBiOh/absto/pkg/model (interfaces: Storage)
//
// Generated by this command:
//
//	mockgen -destination ../mocks/storage.go -package mocks -mock_names Storage=Storage github.com/ViBiOh/absto/pkg/model Storage
//
// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	io "io"
	fs "io/fs"
	reflect "reflect"
	time "time"

	model "github.com/ViBiOh/absto/pkg/model"
	gomock "go.uber.org/mock/gomock"
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

// ConvertError mocks base method.
func (m *Storage) ConvertError(arg0 error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConvertError", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConvertError indicates an expected call of ConvertError.
func (mr *StorageMockRecorder) ConvertError(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConvertError", reflect.TypeOf((*Storage)(nil).ConvertError), arg0)
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

// List mocks base method.
func (m *Storage) List(arg0 context.Context, arg1 string) ([]model.Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].([]model.Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *StorageMockRecorder) List(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*Storage)(nil).List), arg0, arg1)
}

// Mkdir mocks base method.
func (m *Storage) Mkdir(arg0 context.Context, arg1 string, arg2 fs.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Mkdir", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Mkdir indicates an expected call of Mkdir.
func (mr *StorageMockRecorder) Mkdir(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mkdir", reflect.TypeOf((*Storage)(nil).Mkdir), arg0, arg1, arg2)
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

// OpenFile mocks base method.
func (m *Storage) OpenFile(arg0 context.Context, arg1 string, arg2 int, arg3 fs.FileMode) (model.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OpenFile", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(model.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// OpenFile indicates an expected call of OpenFile.
func (mr *StorageMockRecorder) OpenFile(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenFile", reflect.TypeOf((*Storage)(nil).OpenFile), arg0, arg1, arg2, arg3)
}

// Path mocks base method.
func (m *Storage) Path(arg0 string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Path", arg0)
	ret0, _ := ret[0].(string)
	return ret0
}

// Path indicates an expected call of Path.
func (mr *StorageMockRecorder) Path(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Path", reflect.TypeOf((*Storage)(nil).Path), arg0)
}

// ReadFrom mocks base method.
func (m *Storage) ReadFrom(arg0 context.Context, arg1 string) (model.ReadAtSeekCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFrom", arg0, arg1)
	ret0, _ := ret[0].(model.ReadAtSeekCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFrom indicates an expected call of ReadFrom.
func (mr *StorageMockRecorder) ReadFrom(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFrom", reflect.TypeOf((*Storage)(nil).ReadFrom), arg0, arg1)
}

// RemoveAll mocks base method.
func (m *Storage) RemoveAll(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveAll", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveAll indicates an expected call of RemoveAll.
func (mr *StorageMockRecorder) RemoveAll(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveAll", reflect.TypeOf((*Storage)(nil).RemoveAll), arg0, arg1)
}

// Rename mocks base method.
func (m *Storage) Rename(arg0 context.Context, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rename", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Rename indicates an expected call of Rename.
func (mr *StorageMockRecorder) Rename(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rename", reflect.TypeOf((*Storage)(nil).Rename), arg0, arg1, arg2)
}

// Stat mocks base method.
func (m *Storage) Stat(arg0 context.Context, arg1 string) (model.Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stat", arg0, arg1)
	ret0, _ := ret[0].(model.Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Stat indicates an expected call of Stat.
func (mr *StorageMockRecorder) Stat(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stat", reflect.TypeOf((*Storage)(nil).Stat), arg0, arg1)
}

// UpdateDate mocks base method.
func (m *Storage) UpdateDate(arg0 context.Context, arg1 string, arg2 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDate", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateDate indicates an expected call of UpdateDate.
func (mr *StorageMockRecorder) UpdateDate(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDate", reflect.TypeOf((*Storage)(nil).UpdateDate), arg0, arg1, arg2)
}

// Walk mocks base method.
func (m *Storage) Walk(arg0 context.Context, arg1 string, arg2 func(model.Item) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Walk", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Walk indicates an expected call of Walk.
func (mr *StorageMockRecorder) Walk(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Walk", reflect.TypeOf((*Storage)(nil).Walk), arg0, arg1, arg2)
}

// WithIgnoreFn mocks base method.
func (m *Storage) WithIgnoreFn(arg0 func(model.Item) bool) model.Storage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithIgnoreFn", arg0)
	ret0, _ := ret[0].(model.Storage)
	return ret0
}

// WithIgnoreFn indicates an expected call of WithIgnoreFn.
func (mr *StorageMockRecorder) WithIgnoreFn(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithIgnoreFn", reflect.TypeOf((*Storage)(nil).WithIgnoreFn), arg0)
}

// WriteTo mocks base method.
func (m *Storage) WriteTo(arg0 context.Context, arg1 string, arg2 io.Reader, arg3 model.WriteOpts) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteTo", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteTo indicates an expected call of WriteTo.
func (mr *StorageMockRecorder) WriteTo(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteTo", reflect.TypeOf((*Storage)(nil).WriteTo), arg0, arg1, arg2, arg3)
}
