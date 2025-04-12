// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rusq/slackdump/v3/internal/chunk (interfaces: Encoder)
//
// Generated by this command:
//
//	mockgen -destination=mock_chunk/mock_encoder.go . Encoder
//

// Package mock_chunk is a generated GoMock package.
package mock_chunk

import (
	context "context"
	reflect "reflect"

	chunk "github.com/rusq/slackdump/v3/internal/chunk"
	gomock "go.uber.org/mock/gomock"
)

// MockEncoder is a mock of Encoder interface.
type MockEncoder struct {
	ctrl     *gomock.Controller
	recorder *MockEncoderMockRecorder
	isgomock struct{}
}

// MockEncoderMockRecorder is the mock recorder for MockEncoder.
type MockEncoderMockRecorder struct {
	mock *MockEncoder
}

// NewMockEncoder creates a new mock instance.
func NewMockEncoder(ctrl *gomock.Controller) *MockEncoder {
	mock := &MockEncoder{ctrl: ctrl}
	mock.recorder = &MockEncoderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEncoder) EXPECT() *MockEncoderMockRecorder {
	return m.recorder
}

// Encode mocks base method.
func (m *MockEncoder) Encode(ctx context.Context, chunk *chunk.Chunk) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Encode", ctx, chunk)
	ret0, _ := ret[0].(error)
	return ret0
}

// Encode indicates an expected call of Encode.
func (mr *MockEncoderMockRecorder) Encode(ctx, chunk any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Encode", reflect.TypeOf((*MockEncoder)(nil).Encode), ctx, chunk)
}
