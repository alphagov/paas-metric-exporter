// Code generated by counterfeiter. DO NOT EDIT.
package expirationfakes

import (
	"sync"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket/db"
	"code.cloudfoundry.org/locket/expiration"
)

type FakeLockPick struct {
	RegisterTTLStub        func(logger lager.Logger, lock *db.Lock)
	registerTTLMutex       sync.RWMutex
	registerTTLArgsForCall []struct {
		logger lager.Logger
		lock   *db.Lock
	}
	ExpirationCountsStub        func() (uint32, uint32)
	expirationCountsMutex       sync.RWMutex
	expirationCountsArgsForCall []struct{}
	expirationCountsReturns     struct {
		result1 uint32
		result2 uint32
	}
	expirationCountsReturnsOnCall map[int]struct {
		result1 uint32
		result2 uint32
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeLockPick) RegisterTTL(logger lager.Logger, lock *db.Lock) {
	fake.registerTTLMutex.Lock()
	fake.registerTTLArgsForCall = append(fake.registerTTLArgsForCall, struct {
		logger lager.Logger
		lock   *db.Lock
	}{logger, lock})
	fake.recordInvocation("RegisterTTL", []interface{}{logger, lock})
	fake.registerTTLMutex.Unlock()
	if fake.RegisterTTLStub != nil {
		fake.RegisterTTLStub(logger, lock)
	}
}

func (fake *FakeLockPick) RegisterTTLCallCount() int {
	fake.registerTTLMutex.RLock()
	defer fake.registerTTLMutex.RUnlock()
	return len(fake.registerTTLArgsForCall)
}

func (fake *FakeLockPick) RegisterTTLArgsForCall(i int) (lager.Logger, *db.Lock) {
	fake.registerTTLMutex.RLock()
	defer fake.registerTTLMutex.RUnlock()
	return fake.registerTTLArgsForCall[i].logger, fake.registerTTLArgsForCall[i].lock
}

func (fake *FakeLockPick) ExpirationCounts() (uint32, uint32) {
	fake.expirationCountsMutex.Lock()
	ret, specificReturn := fake.expirationCountsReturnsOnCall[len(fake.expirationCountsArgsForCall)]
	fake.expirationCountsArgsForCall = append(fake.expirationCountsArgsForCall, struct{}{})
	fake.recordInvocation("ExpirationCounts", []interface{}{})
	fake.expirationCountsMutex.Unlock()
	if fake.ExpirationCountsStub != nil {
		return fake.ExpirationCountsStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.expirationCountsReturns.result1, fake.expirationCountsReturns.result2
}

func (fake *FakeLockPick) ExpirationCountsCallCount() int {
	fake.expirationCountsMutex.RLock()
	defer fake.expirationCountsMutex.RUnlock()
	return len(fake.expirationCountsArgsForCall)
}

func (fake *FakeLockPick) ExpirationCountsReturns(result1 uint32, result2 uint32) {
	fake.ExpirationCountsStub = nil
	fake.expirationCountsReturns = struct {
		result1 uint32
		result2 uint32
	}{result1, result2}
}

func (fake *FakeLockPick) ExpirationCountsReturnsOnCall(i int, result1 uint32, result2 uint32) {
	fake.ExpirationCountsStub = nil
	if fake.expirationCountsReturnsOnCall == nil {
		fake.expirationCountsReturnsOnCall = make(map[int]struct {
			result1 uint32
			result2 uint32
		})
	}
	fake.expirationCountsReturnsOnCall[i] = struct {
		result1 uint32
		result2 uint32
	}{result1, result2}
}

func (fake *FakeLockPick) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.registerTTLMutex.RLock()
	defer fake.registerTTLMutex.RUnlock()
	fake.expirationCountsMutex.RLock()
	defer fake.expirationCountsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeLockPick) recordInvocation(key string, args []interface{}) {
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

var _ expiration.LockPick = new(FakeLockPick)
