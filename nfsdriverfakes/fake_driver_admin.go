// This file was generated by counterfeiter
package nfsdriverfakes

import (
	"sync"

	"code.cloudfoundry.org/nfsv3driver/driveradmin"
	"code.cloudfoundry.org/voldriver"
)

type FakeDriverAdmin struct {
	EvacuateStub        func(env voldriver.Env) driveradmin.ErrorResponse
	evacuateMutex       sync.RWMutex
	evacuateArgsForCall []struct {
		env voldriver.Env
	}
	evacuateReturns struct {
		result1 driveradmin.ErrorResponse
	}
	PingStub        func(env voldriver.Env) driveradmin.ErrorResponse
	pingMutex       sync.RWMutex
	pingArgsForCall []struct {
		env voldriver.Env
	}
	pingReturns struct {
		result1 driveradmin.ErrorResponse
	}
}

func (fake *FakeDriverAdmin) Evacuate(env voldriver.Env) driveradmin.ErrorResponse {
	fake.evacuateMutex.Lock()
	fake.evacuateArgsForCall = append(fake.evacuateArgsForCall, struct {
		env voldriver.Env
	}{env})
	fake.evacuateMutex.Unlock()
	if fake.EvacuateStub != nil {
		return fake.EvacuateStub(env)
	} else {
		return fake.evacuateReturns.result1
	}
}

func (fake *FakeDriverAdmin) EvacuateCallCount() int {
	fake.evacuateMutex.RLock()
	defer fake.evacuateMutex.RUnlock()
	return len(fake.evacuateArgsForCall)
}

func (fake *FakeDriverAdmin) EvacuateArgsForCall(i int) voldriver.Env {
	fake.evacuateMutex.RLock()
	defer fake.evacuateMutex.RUnlock()
	return fake.evacuateArgsForCall[i].env
}

func (fake *FakeDriverAdmin) EvacuateReturns(result1 driveradmin.ErrorResponse) {
	fake.EvacuateStub = nil
	fake.evacuateReturns = struct {
		result1 driveradmin.ErrorResponse
	}{result1}
}

func (fake *FakeDriverAdmin) Ping(env voldriver.Env) driveradmin.ErrorResponse {
	fake.pingMutex.Lock()
	fake.pingArgsForCall = append(fake.pingArgsForCall, struct {
		env voldriver.Env
	}{env})
	fake.pingMutex.Unlock()
	if fake.PingStub != nil {
		return fake.PingStub(env)
	} else {
		return fake.pingReturns.result1
	}
}

func (fake *FakeDriverAdmin) PingCallCount() int {
	fake.pingMutex.RLock()
	defer fake.pingMutex.RUnlock()
	return len(fake.pingArgsForCall)
}

func (fake *FakeDriverAdmin) PingArgsForCall(i int) voldriver.Env {
	fake.pingMutex.RLock()
	defer fake.pingMutex.RUnlock()
	return fake.pingArgsForCall[i].env
}

func (fake *FakeDriverAdmin) PingReturns(result1 driveradmin.ErrorResponse) {
	fake.PingStub = nil
	fake.pingReturns = struct {
		result1 driveradmin.ErrorResponse
	}{result1}
}

var _ driveradmin.DriverAdmin = new(FakeDriverAdmin)