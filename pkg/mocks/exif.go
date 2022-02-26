// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ViBiOh/fibr/pkg/provider (interfaces: ExifManager)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	model "github.com/ViBiOh/absto/pkg/model"
	model0 "github.com/ViBiOh/exas/pkg/model"
	provider "github.com/ViBiOh/fibr/pkg/provider"
	gomock "github.com/golang/mock/gomock"
)

// Exif is a mock of ExifManager interface.
type Exif struct {
	ctrl     *gomock.Controller
	recorder *ExifMockRecorder
}

// ExifMockRecorder is the mock recorder for Exif.
type ExifMockRecorder struct {
	mock *Exif
}

// NewExif creates a new mock instance.
func NewExif(ctrl *gomock.Controller) *Exif {
	mock := &Exif{ctrl: ctrl}
	mock.recorder = &ExifMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Exif) EXPECT() *ExifMockRecorder {
	return m.recorder
}

// GetAggregateFor mocks base method.
func (m *Exif) GetAggregateFor(arg0 context.Context, arg1 model.Item) (provider.Aggregate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAggregateFor", arg0, arg1)
	ret0, _ := ret[0].(provider.Aggregate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAggregateFor indicates an expected call of GetAggregateFor.
func (mr *ExifMockRecorder) GetAggregateFor(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAggregateFor", reflect.TypeOf((*Exif)(nil).GetAggregateFor), arg0, arg1)
}

// GetExifFor mocks base method.
func (m *Exif) GetExifFor(arg0 context.Context, arg1 model.Item) (model0.Exif, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExifFor", arg0, arg1)
	ret0, _ := ret[0].(model0.Exif)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExifFor indicates an expected call of GetExifFor.
func (mr *ExifMockRecorder) GetExifFor(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExifFor", reflect.TypeOf((*Exif)(nil).GetExifFor), arg0, arg1)
}
