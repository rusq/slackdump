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
	isgomock struct{}
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
func (m *MockConversations) ChannelInfo(ctx context.Context, ci *slack.Channel, threadID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelInfo", ctx, ci, threadID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelInfo indicates an expected call of ChannelInfo.
func (mr *MockConversationsMockRecorder) ChannelInfo(ctx, ci, threadID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelInfo", reflect.TypeOf((*MockConversations)(nil).ChannelInfo), ctx, ci, threadID)
}

// ChannelUsers mocks base method.
func (m *MockConversations) ChannelUsers(ctx context.Context, channelID, threadTS string, users []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelUsers", ctx, channelID, threadTS, users)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelUsers indicates an expected call of ChannelUsers.
func (mr *MockConversationsMockRecorder) ChannelUsers(ctx, channelID, threadTS, users any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelUsers", reflect.TypeOf((*MockConversations)(nil).ChannelUsers), ctx, channelID, threadTS, users)
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
func (m *MockConversations) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Files", ctx, channel, parent, ff)
	ret0, _ := ret[0].(error)
	return ret0
}

// Files indicates an expected call of Files.
func (mr *MockConversationsMockRecorder) Files(ctx, channel, parent, ff any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Files", reflect.TypeOf((*MockConversations)(nil).Files), ctx, channel, parent, ff)
}

// Messages mocks base method.
func (m *MockConversations) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Messages", ctx, channelID, numThreads, isLast, messages)
	ret0, _ := ret[0].(error)
	return ret0
}

// Messages indicates an expected call of Messages.
func (mr *MockConversationsMockRecorder) Messages(ctx, channelID, numThreads, isLast, messages any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Messages", reflect.TypeOf((*MockConversations)(nil).Messages), ctx, channelID, numThreads, isLast, messages)
}

// ThreadMessages mocks base method.
func (m *MockConversations) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ThreadMessages", ctx, channelID, parent, threadOnly, isLast, replies)
	ret0, _ := ret[0].(error)
	return ret0
}

// ThreadMessages indicates an expected call of ThreadMessages.
func (mr *MockConversationsMockRecorder) ThreadMessages(ctx, channelID, parent, threadOnly, isLast, replies any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ThreadMessages", reflect.TypeOf((*MockConversations)(nil).ThreadMessages), ctx, channelID, parent, threadOnly, isLast, replies)
}

// MockUsers is a mock of Users interface.
type MockUsers struct {
	ctrl     *gomock.Controller
	recorder *MockUsersMockRecorder
	isgomock struct{}
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
func (m *MockUsers) Users(ctx context.Context, users []slack.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Users", ctx, users)
	ret0, _ := ret[0].(error)
	return ret0
}

// Users indicates an expected call of Users.
func (mr *MockUsersMockRecorder) Users(ctx, users any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Users", reflect.TypeOf((*MockUsers)(nil).Users), ctx, users)
}

// MockChannels is a mock of Channels interface.
type MockChannels struct {
	ctrl     *gomock.Controller
	recorder *MockChannelsMockRecorder
	isgomock struct{}
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
func (m *MockChannels) Channels(ctx context.Context, channels []slack.Channel) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Channels", ctx, channels)
	ret0, _ := ret[0].(error)
	return ret0
}

// Channels indicates an expected call of Channels.
func (mr *MockChannelsMockRecorder) Channels(ctx, channels any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Channels", reflect.TypeOf((*MockChannels)(nil).Channels), ctx, channels)
}

// MockChannelInformer is a mock of ChannelInformer interface.
type MockChannelInformer struct {
	ctrl     *gomock.Controller
	recorder *MockChannelInformerMockRecorder
	isgomock struct{}
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
func (m *MockChannelInformer) ChannelInfo(ctx context.Context, ci *slack.Channel, threadID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelInfo", ctx, ci, threadID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelInfo indicates an expected call of ChannelInfo.
func (mr *MockChannelInformerMockRecorder) ChannelInfo(ctx, ci, threadID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelInfo", reflect.TypeOf((*MockChannelInformer)(nil).ChannelInfo), ctx, ci, threadID)
}

// ChannelUsers mocks base method.
func (m *MockChannelInformer) ChannelUsers(ctx context.Context, channelID, threadTS string, users []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChannelUsers", ctx, channelID, threadTS, users)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChannelUsers indicates an expected call of ChannelUsers.
func (mr *MockChannelInformerMockRecorder) ChannelUsers(ctx, channelID, threadTS, users any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChannelUsers", reflect.TypeOf((*MockChannelInformer)(nil).ChannelUsers), ctx, channelID, threadTS, users)
}

// MockFiler is a mock of Filer interface.
type MockFiler struct {
	ctrl     *gomock.Controller
	recorder *MockFilerMockRecorder
	isgomock struct{}
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
func (m *MockFiler) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Files", ctx, channel, parent, ff)
	ret0, _ := ret[0].(error)
	return ret0
}

// Files indicates an expected call of Files.
func (mr *MockFilerMockRecorder) Files(ctx, channel, parent, ff any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Files", reflect.TypeOf((*MockFiler)(nil).Files), ctx, channel, parent, ff)
}
