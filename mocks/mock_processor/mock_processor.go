// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rusq/slackdump/v3/processor (interfaces: Conversations,Users,Channels,ChannelInformer,Filer)
//
// Generated by this command:
//
//	mockgen -destination ../mocks/mock_processor/mock_processor.go github.com/rusq/slackdump/v3/processor Conversations,Users,Channels,ChannelInformer,Filer
//

// Package mock_processor is a generated GoMock package.
package mock_processor

import (
	context "context"
	reflect "reflect"

	slack "github.com/rusq/slack"
	gomock "go.uber.org/mock/gomock"
)

// MockConversations is a mock of Conversations interface.
type MockConversations struct {
	ctrl     *gomock.Controller
	recorder *MockConversationsMockRecorder
}

// MockConversationsMockRecorder is the mock recorder for MockConversations.
type MockConversationsMockRecorder struct {
	mock *MockConversations
}

// NewMockConversations creates a new mock instance.
func NewMockConversations(ctrl *gomock.Controller) *MockConversations {
	mock := &MockConversations{ctrl: ctrl}
	mock.recorder = &MockConversationsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConversations) EXPECT() *MockConversationsMockRecorder {
	return m.recorder
}

// ChannelInfo mocks base method.
func (m *MockConversations) ChannelInfo(arg0 context.Context, arg1 *slack.Channel, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelInfo", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelInfo indicates an expected call of ChannelInfo.
func (mr *MockConversationsMockRecorder) ChannelInfo(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelInfo", reflect.TypeOf((*MockConversations)(nil).ChannelInfo), arg0, arg1, arg2)
}

// ChannelUsers mocks base method.
func (m *MockConversations) ChannelUsers(arg0 context.Context, arg1, arg2 string, arg3 []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelUsers", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelUsers indicates an expected call of ChannelUsers.
func (mr *MockConversationsMockRecorder) ChannelUsers(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelUsers", reflect.TypeOf((*MockConversations)(nil).ChannelUsers), arg0, arg1, arg2, arg3)
}

// Close mocks base method.
func (m *MockConversations) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockConversationsMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockConversations)(nil).Close))
}

// Files mocks base method.
func (m *MockConversations) Files(arg0 context.Context, arg1 *slack.Channel, arg2 slack.Message, arg3 []slack.File) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Files", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// Files indicates an expected call of Files.
func (mr *MockConversationsMockRecorder) Files(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Files", reflect.TypeOf((*MockConversations)(nil).Files), arg0, arg1, arg2, arg3)
}

// Messages mocks base method.
func (m *MockConversations) Messages(arg0 context.Context, arg1 string, arg2 int, arg3 bool, arg4 []slack.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Messages", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// Messages indicates an expected call of Messages.
func (mr *MockConversationsMockRecorder) Messages(arg0, arg1, arg2, arg3, arg4 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Messages", reflect.TypeOf((*MockConversations)(nil).Messages), arg0, arg1, arg2, arg3, arg4)
}

// ThreadMessages mocks base method.
func (m *MockConversations) ThreadMessages(arg0 context.Context, arg1 string, arg2 slack.Message, arg3, arg4 bool, arg5 []slack.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ThreadMessages", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(error)
	return ret0
}

// ThreadMessages indicates an expected call of ThreadMessages.
func (mr *MockConversationsMockRecorder) ThreadMessages(arg0, arg1, arg2, arg3, arg4, arg5 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ThreadMessages", reflect.TypeOf((*MockConversations)(nil).ThreadMessages), arg0, arg1, arg2, arg3, arg4, arg5)
}

// MockUsers is a mock of Users interface.
type MockUsers struct {
	ctrl     *gomock.Controller
	recorder *MockUsersMockRecorder
}

// MockUsersMockRecorder is the mock recorder for MockUsers.
type MockUsersMockRecorder struct {
	mock *MockUsers
}

// NewMockUsers creates a new mock instance.
func NewMockUsers(ctrl *gomock.Controller) *MockUsers {
	mock := &MockUsers{ctrl: ctrl}
	mock.recorder = &MockUsersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUsers) EXPECT() *MockUsersMockRecorder {
	return m.recorder
}

// Users mocks base method.
func (m *MockUsers) Users(arg0 context.Context, arg1 []slack.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Users", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Users indicates an expected call of Users.
func (mr *MockUsersMockRecorder) Users(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Users", reflect.TypeOf((*MockUsers)(nil).Users), arg0, arg1)
}

// MockChannels is a mock of Channels interface.
type MockChannels struct {
	ctrl     *gomock.Controller
	recorder *MockChannelsMockRecorder
}

// MockChannelsMockRecorder is the mock recorder for MockChannels.
type MockChannelsMockRecorder struct {
	mock *MockChannels
}

// NewMockChannels creates a new mock instance.
func NewMockChannels(ctrl *gomock.Controller) *MockChannels {
	mock := &MockChannels{ctrl: ctrl}
	mock.recorder = &MockChannelsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChannels) EXPECT() *MockChannelsMockRecorder {
	return m.recorder
}

// Channels mocks base method.
func (m *MockChannels) Channels(arg0 context.Context, arg1 []slack.Channel) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Channels", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Channels indicates an expected call of Channels.
func (mr *MockChannelsMockRecorder) Channels(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Channels", reflect.TypeOf((*MockChannels)(nil).Channels), arg0, arg1)
}

// MockChannelInformer is a mock of ChannelInformer interface.
type MockChannelInformer struct {
	ctrl     *gomock.Controller
	recorder *MockChannelInformerMockRecorder
}

// MockChannelInformerMockRecorder is the mock recorder for MockChannelInformer.
type MockChannelInformerMockRecorder struct {
	mock *MockChannelInformer
}

// NewMockChannelInformer creates a new mock instance.
func NewMockChannelInformer(ctrl *gomock.Controller) *MockChannelInformer {
	mock := &MockChannelInformer{ctrl: ctrl}
	mock.recorder = &MockChannelInformerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChannelInformer) EXPECT() *MockChannelInformerMockRecorder {
	return m.recorder
}

// ChannelInfo mocks base method.
func (m *MockChannelInformer) ChannelInfo(arg0 context.Context, arg1 *slack.Channel, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelInfo", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelInfo indicates an expected call of ChannelInfo.
func (mr *MockChannelInformerMockRecorder) ChannelInfo(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelInfo", reflect.TypeOf((*MockChannelInformer)(nil).ChannelInfo), arg0, arg1, arg2)
}

// ChannelUsers mocks base method.
func (m *MockChannelInformer) ChannelUsers(arg0 context.Context, arg1, arg2 string, arg3 []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelUsers", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelUsers indicates an expected call of ChannelUsers.
func (mr *MockChannelInformerMockRecorder) ChannelUsers(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelUsers", reflect.TypeOf((*MockChannelInformer)(nil).ChannelUsers), arg0, arg1, arg2, arg3)
}

// MockFiler is a mock of Filer interface.
type MockFiler struct {
	ctrl     *gomock.Controller
	recorder *MockFilerMockRecorder
}

// MockFilerMockRecorder is the mock recorder for MockFiler.
type MockFilerMockRecorder struct {
	mock *MockFiler
}

// NewMockFiler creates a new mock instance.
func NewMockFiler(ctrl *gomock.Controller) *MockFiler {
	mock := &MockFiler{ctrl: ctrl}
	mock.recorder = &MockFilerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFiler) EXPECT() *MockFilerMockRecorder {
	return m.recorder
}

// Files mocks base method.
func (m *MockFiler) Files(arg0 context.Context, arg1 *slack.Channel, arg2 slack.Message, arg3 []slack.File) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Files", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// Files indicates an expected call of Files.
func (mr *MockFilerMockRecorder) Files(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Files", reflect.TypeOf((*MockFiler)(nil).Files), arg0, arg1, arg2, arg3)
}
