// Code generated by MockGen. DO NOT EDIT.
// Source: interfaces.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	http "net/http"
	reflect "reflect"
	time "time"

	model "github.com/ViBiOh/absto/pkg/model"
	ident "github.com/ViBiOh/auth/v2/pkg/ident"
	model0 "github.com/ViBiOh/auth/v2/pkg/model"
	model1 "github.com/ViBiOh/exas/pkg/model"
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
func (m *Crud) Get(arg0 http.ResponseWriter, arg1 *http.Request, arg2 provider.Request) (renderer.Page, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1, arg2)
	ret0, _ := ret[0].(renderer.Page)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *CrudMockRecorder) Get(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*Crud)(nil).Get), arg0, arg1, arg2)
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

// Auth is a mock of Auth interface.
type Auth struct {
	ctrl     *gomock.Controller
	recorder *AuthMockRecorder
}

// AuthMockRecorder is the mock recorder for Auth.
type AuthMockRecorder struct {
	mock *Auth
}

// NewAuth creates a new mock instance.
func NewAuth(ctrl *gomock.Controller) *Auth {
	mock := &Auth{ctrl: ctrl}
	mock.recorder = &AuthMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Auth) EXPECT() *AuthMockRecorder {
	return m.recorder
}

// IsAuthenticated mocks base method.
func (m *Auth) IsAuthenticated(arg0 *http.Request) (ident.Provider, model0.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAuthenticated", arg0)
	ret0, _ := ret[0].(ident.Provider)
	ret1, _ := ret[1].(model0.User)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// IsAuthenticated indicates an expected call of IsAuthenticated.
func (mr *AuthMockRecorder) IsAuthenticated(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAuthenticated", reflect.TypeOf((*Auth)(nil).IsAuthenticated), arg0)
}

// IsAuthorized mocks base method.
func (m *Auth) IsAuthorized(arg0 context.Context, arg1 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAuthorized", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsAuthorized indicates an expected call of IsAuthorized.
func (mr *AuthMockRecorder) IsAuthorized(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAuthorized", reflect.TypeOf((*Auth)(nil).IsAuthorized), arg0, arg1)
}

// ShareManager is a mock of ShareManager interface.
type ShareManager struct {
	ctrl     *gomock.Controller
	recorder *ShareManagerMockRecorder
}

// ShareManagerMockRecorder is the mock recorder for ShareManager.
type ShareManagerMockRecorder struct {
	mock *ShareManager
}

// NewShareManager creates a new mock instance.
func NewShareManager(ctrl *gomock.Controller) *ShareManager {
	mock := &ShareManager{ctrl: ctrl}
	mock.recorder = &ShareManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *ShareManager) EXPECT() *ShareManagerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *ShareManager) Create(arg0 context.Context, arg1 string, arg2, arg3 bool, arg4 string, arg5 bool, arg6 time.Duration) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *ShareManagerMockRecorder) Create(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*ShareManager)(nil).Create), arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// Delete mocks base method.
func (m *ShareManager) Delete(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *ShareManagerMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*ShareManager)(nil).Delete), arg0, arg1)
}

// Get mocks base method.
func (m *ShareManager) Get(arg0 string) provider.Share {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(provider.Share)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *ShareManagerMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*ShareManager)(nil).Get), arg0)
}

// List mocks base method.
func (m *ShareManager) List() []provider.Share {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].([]provider.Share)
	return ret0
}

// List indicates an expected call of List.
func (mr *ShareManagerMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*ShareManager)(nil).List))
}

// WebhookManager is a mock of WebhookManager interface.
type WebhookManager struct {
	ctrl     *gomock.Controller
	recorder *WebhookManagerMockRecorder
}

// WebhookManagerMockRecorder is the mock recorder for WebhookManager.
type WebhookManagerMockRecorder struct {
	mock *WebhookManager
}

// NewWebhookManager creates a new mock instance.
func NewWebhookManager(ctrl *gomock.Controller) *WebhookManager {
	mock := &WebhookManager{ctrl: ctrl}
	mock.recorder = &WebhookManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *WebhookManager) EXPECT() *WebhookManagerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *WebhookManager) Create(arg0 context.Context, arg1 string, arg2 bool, arg3 provider.WebhookKind, arg4 string, arg5 []provider.EventType) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *WebhookManagerMockRecorder) Create(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*WebhookManager)(nil).Create), arg0, arg1, arg2, arg3, arg4, arg5)
}

// Delete mocks base method.
func (m *WebhookManager) Delete(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *WebhookManagerMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*WebhookManager)(nil).Delete), arg0, arg1)
}

// List mocks base method.
func (m *WebhookManager) List() []provider.Webhook {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].([]provider.Webhook)
	return ret0
}

// List indicates an expected call of List.
func (mr *WebhookManagerMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*WebhookManager)(nil).List))
}

// ExifManager is a mock of ExifManager interface.
type ExifManager struct {
	ctrl     *gomock.Controller
	recorder *ExifManagerMockRecorder
}

// ExifManagerMockRecorder is the mock recorder for ExifManager.
type ExifManagerMockRecorder struct {
	mock *ExifManager
}

// NewExifManager creates a new mock instance.
func NewExifManager(ctrl *gomock.Controller) *ExifManager {
	mock := &ExifManager{ctrl: ctrl}
	mock.recorder = &ExifManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *ExifManager) EXPECT() *ExifManagerMockRecorder {
	return m.recorder
}

// GetAggregateFor mocks base method.
func (m *ExifManager) GetAggregateFor(ctx context.Context, item model.Item) (provider.Aggregate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAggregateFor", ctx, item)
	ret0, _ := ret[0].(provider.Aggregate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAggregateFor indicates an expected call of GetAggregateFor.
func (mr *ExifManagerMockRecorder) GetAggregateFor(ctx, item interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAggregateFor", reflect.TypeOf((*ExifManager)(nil).GetAggregateFor), ctx, item)
}

// GetExifFor mocks base method.
func (m *ExifManager) GetExifFor(ctx context.Context, item model.Item) (model1.Exif, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExifFor", ctx, item)
	ret0, _ := ret[0].(model1.Exif)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExifFor indicates an expected call of GetExifFor.
func (mr *ExifManagerMockRecorder) GetExifFor(ctx, item interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExifFor", reflect.TypeOf((*ExifManager)(nil).GetExifFor), ctx, item)
}

// ListDir mocks base method.
func (m *ExifManager) ListDir(ctx context.Context, item model.Item) ([]model.Item, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDir", ctx, item)
	ret0, _ := ret[0].([]model.Item)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDir indicates an expected call of ListDir.
func (mr *ExifManagerMockRecorder) ListDir(ctx, item interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDir", reflect.TypeOf((*ExifManager)(nil).ListDir), ctx, item)
}

// SaveAggregateFor mocks base method.
func (m *ExifManager) SaveAggregateFor(ctx context.Context, item model.Item, aggregate provider.Aggregate) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveAggregateFor", ctx, item, aggregate)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveAggregateFor indicates an expected call of SaveAggregateFor.
func (mr *ExifManagerMockRecorder) SaveAggregateFor(ctx, item, aggregate interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveAggregateFor", reflect.TypeOf((*ExifManager)(nil).SaveAggregateFor), ctx, item, aggregate)
}

// SaveExifFor mocks base method.
func (m *ExifManager) SaveExifFor(ctx context.Context, item model.Item, exif model1.Exif) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveExifFor", ctx, item, exif)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveExifFor indicates an expected call of SaveExifFor.
func (mr *ExifManagerMockRecorder) SaveExifFor(ctx, item, exif interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveExifFor", reflect.TypeOf((*ExifManager)(nil).SaveExifFor), ctx, item, exif)
}
