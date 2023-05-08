// MIT License: https://github.com/hhassoubi/go-statechart/blob/master/LICENSE
// Copyright (c) 2023 Hicham Hassoubi

package statechart

import (
	"io"
	"sync"
)

type StateMachine[C any] struct {
	impl          stateMachineImpl[C]
	setupMutex    sync.Mutex
	dispatchMutex sync.Mutex
}

// Creates a state machine with a user context
func MakeStateMachine[C any](userContext_ *C) StateMachine[C] {
	return StateMachine[C]{impl: stateMachineImpl[C]{userContext: userContext_}}
}

// Adds a new State to the State Machine
// `state` the new state object to add
// returns the new stateId
func (sm *StateMachine[C]) AddState(state State[C]) StateId {
	sm.setupMutex.Lock()
	defer sm.setupMutex.Unlock()
	return sm.impl.AddState(state)
}

// Adds a Sub-State to the State Machine
// `state` the new state object to add
// `parentId` the parent (super state) ID
func (sm *StateMachine[C]) AddSubState(state State[C], parentId StateId) StateId {
	sm.setupMutex.Lock()
	defer sm.setupMutex.Unlock()
	return sm.impl.AddSubState(state, parentId)
}

// Initializes the state machine
// `initStateId` the initial starting state
func (sm *StateMachine[C]) Initialize(initStateId StateId) {
	sm.setupMutex.Lock()
	defer sm.setupMutex.Unlock()
	sm.impl.Initialize(initStateId)
}

// Dispatches an events to the state machine
// `event` The Event to dispatch
// No error will occur if the Event is unknown to the state machine
func (sm *StateMachine[C]) DispatchEvent(event Event) {
	sm.dispatchMutex.Lock()
	defer sm.dispatchMutex.Unlock()
	sm.impl.DispatchEvent(event)
}

// Sets the Debug Trace Logger for the state machine
func (sm *StateMachine[C]) SetDebugLogger(logger func(msg string, keysAndValues ...interface{})) {
	sm.impl.DebugLogger = logger
}

// Generates the UML diagram for the state machine, using
// `w` the io stream writer
// `umlSyntax` the generator syntax only only support PlantUML syntax for now [https://plantuml.com/state-diagram]
// `diagramType` the type of diagram to use for the generation
func (sm *StateMachine[C]) GenerateUml(w io.Writer, umlSyntax UmlSyntax, diagramType UmlDiagramType) {
	sm.impl.GenerateUml(w, umlSyntax, diagramType)
}
