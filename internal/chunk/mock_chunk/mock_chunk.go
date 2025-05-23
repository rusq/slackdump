// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rusq/slackdump/v3/internal/chunk (interfaces: Transformer)
//
// Generated by this command:
//
//	mockgen -destination=mock_chunk/mock_chunk.go . Transformer
//

// Package mock_chunk is a generated GoMock package.
package mock_chunk

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockTransformer is a mock of Transformer interface.
type MockTransformer struct {
	ctrl     *gomock.Controller
	recorder *MockTransformerMockRecorder
	isgomock struct{}
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
func (m *MockTransformer) Transform(ctx context.Context, channelID, threadID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transform", ctx, channelID, threadID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Transform indicates an expected call of Transform.
func (mr *MockTransformerMockRecorder) Transform(ctx, channelID, threadID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transform", reflect.TypeOf((*MockTransformer)(nil).Transform), ctx, channelID, threadID)
}
