// Code generated by MockGen. DO NOT EDIT.
// Source: conversations.go
//
// Generated by this command:
//
//	mockgen -source=conversations.go -destination=dirproc_mock_test.go -package=dirproc
//
// Package dirproc is a generated GoMock package.
package dirproc

import (
	context "context"
	reflect "reflect"

	slack "github.com/rusq/slack"
	chunk "github.com/rusq/slackdump/v3/internal/chunk"
	gomock "go.uber.org/mock/gomock"
)

// MockTransformer is a mock of Transformer interface.
type MockTransformer struct {
	ctrl     *gomock.Controller
	recorder *MockTransformerMockRecorder
}

// MockTransformerMockRecorder is the mock recorder for MockTransformer.
type MockTransformerMockRecorder struct {
	mock *MockTransformer
}

// NewMockTransformer creates a new mock instance.
func NewMockTransformer(ctrl *gomock.Controller) *MockTransformer {
	mock := &MockTransformer{ctrl: ctrl}
	mock.recorder = &MockTransformerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransformer) EXPECT() *MockTransformerMockRecorder {
	return m.recorder
}

// Transform mocks base method.
func (m *MockTransformer) Transform(ctx context.Context, id chunk.FileID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transform", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Transform indicates an expected call of Transform.
func (mr *MockTransformerMockRecorder) Transform(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transform", reflect.TypeOf((*MockTransformer)(nil).Transform), ctx, id)
}

// Mocktracker is a mock of tracker interface.
type Mocktracker struct {
	ctrl     *gomock.Controller
	recorder *MocktrackerMockRecorder
}

// MocktrackerMockRecorder is the mock recorder for Mocktracker.
type MocktrackerMockRecorder struct {
	mock *Mocktracker
}

// NewMocktracker creates a new mock instance.
func NewMocktracker(ctrl *gomock.Controller) *Mocktracker {
	mock := &Mocktracker{ctrl: ctrl}
	mock.recorder = &MocktrackerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mocktracker) EXPECT() *MocktrackerMockRecorder {
	return m.recorder
}

// CloseAll mocks base method.
func (m *Mocktracker) CloseAll() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseAll")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseAll indicates an expected call of CloseAll.
func (mr *MocktrackerMockRecorder) CloseAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseAll", reflect.TypeOf((*Mocktracker)(nil).CloseAll))
}

// Recorder mocks base method.
func (m *Mocktracker) Recorder(id chunk.FileID) (datahandler, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recorder", id)
	ret0, _ := ret[0].(datahandler)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recorder indicates an expected call of Recorder.
func (mr *MocktrackerMockRecorder) Recorder(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recorder", reflect.TypeOf((*Mocktracker)(nil).Recorder), id)
}

// RefCount mocks base method.
func (m *Mocktracker) RefCount(id chunk.FileID) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefCount", id)
	ret0, _ := ret[0].(int)
	return ret0
}

// RefCount indicates an expected call of RefCount.
func (mr *MocktrackerMockRecorder) RefCount(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefCount", reflect.TypeOf((*Mocktracker)(nil).RefCount), id)
}

// Unregister mocks base method.
func (m *Mocktracker) Unregister(id chunk.FileID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Unregister", id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Unregister indicates an expected call of Unregister.
func (mr *MocktrackerMockRecorder) Unregister(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unregister", reflect.TypeOf((*Mocktracker)(nil).Unregister), id)
}

// Mockdatahandler is a mock of datahandler interface.
type Mockdatahandler struct {
	ctrl     *gomock.Controller
	recorder *MockdatahandlerMockRecorder
}

// MockdatahandlerMockRecorder is the mock recorder for Mockdatahandler.
type MockdatahandlerMockRecorder struct {
	mock *Mockdatahandler
}

// NewMockdatahandler creates a new mock instance.
func NewMockdatahandler(ctrl *gomock.Controller) *Mockdatahandler {
	mock := &Mockdatahandler{ctrl: ctrl}
	mock.recorder = &MockdatahandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockdatahandler) EXPECT() *MockdatahandlerMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *Mockdatahandler) Add(arg0 int) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// Add indicates an expected call of Add.
func (mr *MockdatahandlerMockRecorder) Add(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*Mockdatahandler)(nil).Add), arg0)
}

// ChannelInfo mocks base method.
func (m *Mockdatahandler) ChannelInfo(ctx context.Context, ci *slack.Channel, threadID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelInfo", ctx, ci, threadID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelInfo indicates an expected call of ChannelInfo.
func (mr *MockdatahandlerMockRecorder) ChannelInfo(ctx, ci, threadID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelInfo", reflect.TypeOf((*Mockdatahandler)(nil).ChannelInfo), ctx, ci, threadID)
}

// ChannelUsers mocks base method.
func (m *Mockdatahandler) ChannelUsers(ctx context.Context, channelID, threadTS string, users []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelUsers", ctx, channelID, threadTS, users)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelUsers indicates an expected call of ChannelUsers.
func (mr *MockdatahandlerMockRecorder) ChannelUsers(ctx, channelID, threadTS, users any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelUsers", reflect.TypeOf((*Mockdatahandler)(nil).ChannelUsers), ctx, channelID, threadTS, users)
}

// Close mocks base method.
func (m *Mockdatahandler) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockdatahandlerMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*Mockdatahandler)(nil).Close))
}

// Dec mocks base method.
func (m *Mockdatahandler) Dec() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Dec")
	ret0, _ := ret[0].(int)
	return ret0
}

// Dec indicates an expected call of Dec.
func (mr *MockdatahandlerMockRecorder) Dec() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Dec", reflect.TypeOf((*Mockdatahandler)(nil).Dec))
}

// Files mocks base method.
func (m *Mockdatahandler) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Files", ctx, channel, parent, ff)
	ret0, _ := ret[0].(error)
	return ret0
}

// Files indicates an expected call of Files.
func (mr *MockdatahandlerMockRecorder) Files(ctx, channel, parent, ff any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Files", reflect.TypeOf((*Mockdatahandler)(nil).Files), ctx, channel, parent, ff)
}

// Inc mocks base method.
func (m *Mockdatahandler) Inc() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Inc")
	ret0, _ := ret[0].(int)
	return ret0
}

// Inc indicates an expected call of Inc.
func (mr *MockdatahandlerMockRecorder) Inc() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Inc", reflect.TypeOf((*Mockdatahandler)(nil).Inc))
}

// Messages mocks base method.
func (m *Mockdatahandler) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Messages", ctx, channelID, numThreads, isLast, messages)
	ret0, _ := ret[0].(error)
	return ret0
}

// Messages indicates an expected call of Messages.
func (mr *MockdatahandlerMockRecorder) Messages(ctx, channelID, numThreads, isLast, messages any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Messages", reflect.TypeOf((*Mockdatahandler)(nil).Messages), ctx, channelID, numThreads, isLast, messages)
}

// N mocks base method.
func (m *Mockdatahandler) N() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "N")
	ret0, _ := ret[0].(int)
	return ret0
}

// N indicates an expected call of N.
func (mr *MockdatahandlerMockRecorder) N() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "N", reflect.TypeOf((*Mockdatahandler)(nil).N))
}

// ThreadMessages mocks base method.
func (m *Mockdatahandler) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ThreadMessages", ctx, channelID, parent, threadOnly, isLast, replies)
	ret0, _ := ret[0].(error)
	return ret0
}

// ThreadMessages indicates an expected call of ThreadMessages.
func (mr *MockdatahandlerMockRecorder) ThreadMessages(ctx, channelID, parent, threadOnly, isLast, replies any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ThreadMessages", reflect.TypeOf((*Mockdatahandler)(nil).ThreadMessages), ctx, channelID, parent, threadOnly, isLast, replies)
}

// Mockcounter is a mock of counter interface.
type Mockcounter struct {
	ctrl     *gomock.Controller
	recorder *MockcounterMockRecorder
}

// MockcounterMockRecorder is the mock recorder for Mockcounter.
type MockcounterMockRecorder struct {
	mock *Mockcounter
}

// NewMockcounter creates a new mock instance.
func NewMockcounter(ctrl *gomock.Controller) *Mockcounter {
	mock := &Mockcounter{ctrl: ctrl}
	mock.recorder = &MockcounterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockcounter) EXPECT() *MockcounterMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *Mockcounter) Add(arg0 int) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// Add indicates an expected call of Add.
func (mr *MockcounterMockRecorder) Add(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*Mockcounter)(nil).Add), arg0)
}

// Dec mocks base method.
func (m *Mockcounter) Dec() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Dec")
	ret0, _ := ret[0].(int)
	return ret0
}

// Dec indicates an expected call of Dec.
func (mr *MockcounterMockRecorder) Dec() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Dec", reflect.TypeOf((*Mockcounter)(nil).Dec))
}

// Inc mocks base method.
func (m *Mockcounter) Inc() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Inc")
	ret0, _ := ret[0].(int)
	return ret0
}

// Inc indicates an expected call of Inc.
func (mr *MockcounterMockRecorder) Inc() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Inc", reflect.TypeOf((*Mockcounter)(nil).Inc))
}

// N mocks base method.
func (m *Mockcounter) N() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "N")
	ret0, _ := ret[0].(int)
	return ret0
}

// N indicates an expected call of N.
func (mr *MockcounterMockRecorder) N() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "N", reflect.TypeOf((*Mockcounter)(nil).N))
}