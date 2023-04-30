// MIT License: https://github.com/hhassoubi/go-statechart/blob/master/LICENSE
// Copyright (c) 2023 Hicham Hassoubi

package statechart

import "sync"

type StateMachine[C any] struct {
	impl          stateMachineImpl[C]
	setupMutex    sync.Mutex
	dispatchMutex sync.Mutex
}

func MakeStateMachine[C any](userContext_ *C) StateMachine[C] {
	return StateMachine[C]{impl: stateMachineImpl[C]{userContext: userContext_}}
}

func (sm *StateMachine[C]) AddState(state State[C]) StateId {
	sm.setupMutex.Lock()
	defer sm.setupMutex.Unlock()
	return sm.impl.AddState(state)
}

func (sm *StateMachine[C]) AddSubState(state State[C], parentId StateId) StateId {
	sm.setupMutex.Lock()
	defer sm.setupMutex.Unlock()
	return sm.impl.AddSubState(state, parentId)
}

func (sm *StateMachine[C]) Initialize(initStateId StateId) {
	sm.setupMutex.Lock()
	defer sm.setupMutex.Unlock()
	sm.impl.Initialize(initStateId)
}

func (sm *StateMachine[C]) DispatchEvent(event Event) {
	sm.dispatchMutex.Lock()
	defer sm.dispatchMutex.Unlock()
	sm.impl.DispatchEvent(event)
}

func (sm *StateMachine[C]) SetDebugLogger(logger func(msg string, keysAndValues ...interface{})) {
	sm.impl.DebugLogger = logger
}
