// MIT License: https://github.com/hhassoubi/go-statechart/blob/master/LICENSE
// Copyright (c) 2023 Hicham Hassoubi

package statechart

import (
	"io"
	"sync"
)

type AsyncStateMachine[C any] struct {
	impl         stateMachineImpl[C]
	eventQueue   chan Event
	dispatcherWG sync.WaitGroup
}

// Creates an async state machine with a user context
func MakeAsyncStateMachine[C any](userContext_ *C) AsyncStateMachine[C] {
	return AsyncStateMachine[C]{impl: stateMachineImpl[C]{userContext: userContext_}}
}

// Adds a new State to the State Machine
// `state` the new state object to add
// returns the new stateId
func (sm *AsyncStateMachine[C]) AddState(state State[C]) StateId {
	return sm.impl.AddState(state)
}

// Adds a Sub-State to the State Machine
// `state` the new state object to add
// `parentId` the parent (super state) ID
func (sm *AsyncStateMachine[C]) AddSubState(state State[C], parentId StateId) StateId {
	return sm.impl.AddSubState(state, parentId)
}

// Initializes the state machine
// `initStateId` the initial starting state
func (sm *AsyncStateMachine[C]) Initialize(initStateId StateId) {
	sm.impl.Initialize(initStateId)
	sm.dispatcherWG.Add(1)
	sm.eventQueue = make(chan Event, 10)
	go sm.eventDispatcher()
}

// Dispatches an events to the state machine
// `event` The Event to dispatch
// No error will occur if the Event is unknown to the state machine
func (sm *AsyncStateMachine[C]) DispatchEvent(event Event) {
	sm.eventQueue <- event
}

// Closes the async channel
func (sm *AsyncStateMachine[C]) Close() {
	close(sm.eventQueue)
	sm.dispatcherWG.Wait()
}

// Sets the Debug Trace Logger for the state machine
func (sm *AsyncStateMachine[C]) SetDebugLogger(logger func(msg string, keysAndValues ...interface{})) {
	sm.impl.DebugLogger = logger
}

// Generates the UML diagram for the state machine, using
// `w` the io stream writer
// `umlSyntax` the generator syntax only only support PlantUML syntax for now [https://plantuml.com/state-diagram]
// `diagramType` the type of diagram to use for the generation
func (sm *AsyncStateMachine[C]) GenerateUml(w io.Writer, umlSyntax UmlSyntax, diagramType UmlDiagramType) {
	sm.impl.GenerateUml(w, umlSyntax, diagramType)
}

func (sm *AsyncStateMachine[C]) eventDispatcher() {
	for event := range sm.eventQueue {
		sm.impl.DispatchEvent(event)
	}
	sm.dispatcherWG.Done()
}
