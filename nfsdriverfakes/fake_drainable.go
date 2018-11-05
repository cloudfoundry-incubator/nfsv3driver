// Code generated by counterfeiter. DO NOT EDIT.
package nfsdriverfakes

import (
	"sync"

	"code.cloudfoundry.org/dockerdriver"
	"code.cloudfoundry.org/nfsv3driver/driveradmin"
)

type FakeDrainable struct {
	DrainStub        func(env dockerdriver.Env) error
	drainMutex       sync.RWMutex
	drainArgsForCall []struct {
		env dockerdriver.Env
	}
	drainReturns struct {
		result1 error
	}
	drainReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDrainable) Drain(env dockerdriver.Env) error {
	fake.drainMutex.Lock()
	ret, specificReturn := fake.drainReturnsOnCall[len(fake.drainArgsForCall)]
	fake.drainArgsForCall = append(fake.drainArgsForCall, struct {
		env dockerdriver.Env
	}{env})
	fake.recordInvocation("Drain", []interface{}{env})
	fake.drainMutex.Unlock()
	if fake.DrainStub != nil {
		return fake.DrainStub(env)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.drainReturns.result1
}

func (fake *FakeDrainable) DrainCallCount() int {
	fake.drainMutex.RLock()
	defer fake.drainMutex.RUnlock()
	return len(fake.drainArgsForCall)
}

func (fake *FakeDrainable) DrainArgsForCall(i int) dockerdriver.Env {
	fake.drainMutex.RLock()
	defer fake.drainMutex.RUnlock()
	return fake.drainArgsForCall[i].env
}

func (fake *FakeDrainable) DrainReturns(result1 error) {
	fake.DrainStub = nil
	fake.drainReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDrainable) DrainReturnsOnCall(i int, result1 error) {
	fake.DrainStub = nil
	if fake.drainReturnsOnCall == nil {
		fake.drainReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.drainReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeDrainable) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.drainMutex.RLock()
	defer fake.drainMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeDrainable) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ driveradmin.Drainable = new(FakeDrainable)
