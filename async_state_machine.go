// MIT License: https://github.com/hhassoubi/go-statechart/blob/master/LICENSE
// Copyright (c) 2023 Hicham Hassoubi

package statechart

import "sync"

type AsyncStateMachine[C any] struct {
	impl         stateMachineImpl[C]
	eventQueue   chan Event
	dispatcherWG sync.WaitGroup
}

func MakeAsyncStateMachine[C any](userContext_ *C) AsyncStateMachine[C] {
	return AsyncStateMachine[C]{impl: stateMachineImpl[C]{userContext: userContext_}}
}

func (sm *AsyncStateMachine[C]) AddState(state State[C]) StateId {
	return sm.impl.addState(state)
}

func (sm *AsyncStateMachine[C]) AddSubState(state State[C], parentId StateId) StateId {
	return sm.impl.addSubState(state, parentId)
}

func (sm *AsyncStateMachine[C]) Initialize(initStateId StateId) {
	sm.impl.initialize(initStateId)
	sm.dispatcherWG.Add(1)
	sm.eventQueue = make(chan Event, 10)
	go sm.eventDispatcher()
}

func (sm *AsyncStateMachine[C]) DispatchEvent(event Event) {
	sm.eventQueue <- event
}

func (sm *AsyncStateMachine[C]) Close() {
	close(sm.eventQueue)
	sm.dispatcherWG.Wait()
}

func (sm *AsyncStateMachine[C]) SetDebugLogger(logger func(msg string, keysAndValues ...interface{})) {
	sm.impl.DebugLogger = logger
}

func (sm *AsyncStateMachine[C]) eventDispatcher() {
	for event := range sm.eventQueue {
		sm.impl.dispatchEvent(event)
	}
	sm.dispatcherWG.Done()
}
