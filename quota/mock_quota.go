// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/google/trillian/quota (interfaces: Manager)

package quota

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
)

// Mock of Manager interface
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *_MockManagerRecorder
}

// Recorder for MockManager (not exported)
type _MockManagerRecorder struct {
	mock *MockManager
}

func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &_MockManagerRecorder{mock}
	return mock
}

func (_m *MockManager) EXPECT() *_MockManagerRecorder {
	return _m.recorder
}

func (_m *MockManager) GetTokens(_param0 context.Context, _param1 int, _param2 []Spec) error {
	ret := _m.ctrl.Call(_m, "GetTokens", _param0, _param1, _param2)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockManagerRecorder) GetTokens(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetTokens", arg0, arg1, arg2)
}

func (_m *MockManager) GetUser(_param0 context.Context, _param1 interface{}) string {
	ret := _m.ctrl.Call(_m, "GetUser", _param0, _param1)
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockManagerRecorder) GetUser(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetUser", arg0, arg1)
}

func (_m *MockManager) PeekTokens(_param0 context.Context, _param1 []Spec) (map[Spec]int, error) {
	ret := _m.ctrl.Call(_m, "PeekTokens", _param0, _param1)
	ret0, _ := ret[0].(map[Spec]int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockManagerRecorder) PeekTokens(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "PeekTokens", arg0, arg1)
}

func (_m *MockManager) PutTokens(_param0 context.Context, _param1 int, _param2 []Spec) error {
	ret := _m.ctrl.Call(_m, "PutTokens", _param0, _param1, _param2)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockManagerRecorder) PutTokens(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "PutTokens", arg0, arg1, arg2)
}

func (_m *MockManager) ResetQuota(_param0 context.Context, _param1 []Spec) error {
	ret := _m.ctrl.Call(_m, "ResetQuota", _param0, _param1)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockManagerRecorder) ResetQuota(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ResetQuota", arg0, arg1)
}