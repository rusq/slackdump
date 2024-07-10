// Code generated by MockGen. DO NOT EDIT.
// Source: future.go
//
// Generated by this command:
//
//	mockgen -destination=mock_dumper_test.go -source=future.go -package=export dumper
//
// Package export is a generated GoMock package.
package export

import (
	context "context"
	reflect "reflect"
	time "time"

	slack "github.com/rusq/slack"
	slackdump "github.com/rusq/slackdump/v3"
	types "github.com/rusq/slackdump/v3/types"
	gomock "go.uber.org/mock/gomock"
)

// Mockdumper is a mock of dumper interface.
type Mockdumper struct {
	ctrl     *gomock.Controller
	recorder *MockdumperMockRecorder
}

// MockdumperMockRecorder is the mock recorder for Mockdumper.
type MockdumperMockRecorder struct {
	mock *Mockdumper
}

// NewMockdumper creates a new mock instance.
func NewMockdumper(ctrl *gomock.Controller) *Mockdumper {
	mock := &Mockdumper{ctrl: ctrl}
	mock.recorder = &MockdumperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockdumper) EXPECT() *MockdumperMockRecorder {
	return m.recorder
}

// Client mocks base method.
func (m *Mockdumper) Client() *slack.Client {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Client")
	ret0, _ := ret[0].(*slack.Client)
	return ret0
}

// Client indicates an expected call of Client.
func (mr *MockdumperMockRecorder) Client() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Client", reflect.TypeOf((*Mockdumper)(nil).Client))
}

// CurrentUserID mocks base method.
func (m *Mockdumper) CurrentUserID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CurrentUserID")
	ret0, _ := ret[0].(string)
	return ret0
}

// CurrentUserID indicates an expected call of CurrentUserID.
func (mr *MockdumperMockRecorder) CurrentUserID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CurrentUserID", reflect.TypeOf((*Mockdumper)(nil).CurrentUserID))
}

// DumpRaw mocks base method.
func (m *Mockdumper) DumpRaw(ctx context.Context, link string, oldest, latest time.Time, processFn ...slackdump.ProcessFunc) (*types.Conversation, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, link, oldest, latest}
	for _, a := range processFn {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DumpRaw", varargs...)
	ret0, _ := ret[0].(*types.Conversation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DumpRaw indicates an expected call of DumpRaw.
func (mr *MockdumperMockRecorder) DumpRaw(ctx, link, oldest, latest any, processFn ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, link, oldest, latest}, processFn...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DumpRaw", reflect.TypeOf((*Mockdumper)(nil).DumpRaw), varargs...)
}

// GetUsers mocks base method.
func (m *Mockdumper) GetUsers(ctx context.Context) (types.Users, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUsers", ctx)
	ret0, _ := ret[0].(types.Users)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUsers indicates an expected call of GetUsers.
func (mr *MockdumperMockRecorder) GetUsers(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsers", reflect.TypeOf((*Mockdumper)(nil).GetUsers), ctx)
}

// StreamChannels mocks base method.
func (m *Mockdumper) StreamChannels(ctx context.Context, chanTypes []string, cb func(slack.Channel) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StreamChannels", ctx, chanTypes, cb)
	ret0, _ := ret[0].(error)
	return ret0
}

// StreamChannels indicates an expected call of StreamChannels.
func (mr *MockdumperMockRecorder) StreamChannels(ctx, chanTypes, cb any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamChannels", reflect.TypeOf((*Mockdumper)(nil).StreamChannels), ctx, chanTypes, cb)
}
