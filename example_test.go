// MIT License: https://github.com/hhassoubi/go-statechart/blob/master/LICENSE
// Copyright (c) 2023 Hicham Hassoubi

package statechart_test

import (
	"fmt"

	"github.com/hhassoubi/go-statechart"
)

// // Events
type StartStopEv struct {
	statechart.EventDefault
}

type ActivateEv struct {
	statechart.EventDefault
}

type DeactivateEv struct {
	statechart.EventDefault
}

type ResetEv struct {
	statechart.EventDefault
}

type MyContext struct {
}

func (*MyContext) StartCounter() {
	// Start Counter Code
	fmt.Println("Start Counter")
}

func (*MyContext) StopCounter() {
	// Start Counter Code
	fmt.Println("Stop Counter")
}

func (*MyContext) ResetCounter() {
	// Start Counter Code
	fmt.Println("Reset Counter")
}

// /// Idle State
type Idle struct {
	statechart.StateDefault[MyContext]
}

func (self *Idle) Setup(proxy statechart.StateSetupProxy[MyContext]) (statechart.EntryAction, statechart.ExitAction) {
	self.Init(proxy)
	// nil is for no action to take
	// signature: AddSimpleStateTransition[EventType, ToStateType]( state State, action func(EventType) )
	statechart.AddSimpleStateTransition[ActivateEv, Active](proxy, nil)
	return nil, nil
}

// /// Active State
type Active struct {
	statechart.StateDefault[MyContext]
}

func (self *Active) Setup(proxy statechart.StateSetupProxy[MyContext]) (statechart.EntryAction, statechart.ExitAction) {
	self.Init(proxy)
	// nil is for no action to take
	statechart.AddSimpleStateTransition[DeactivateEv, Idle](proxy, nil)
	statechart.AddSimpleStateTransition[ResetEv, Active](proxy, nil)
	// Add starting State to Transition to activated (optional). only needed for super states
	// signature: AddStartingState[State](self)
	statechart.SetStartingState[Stopped](proxy)
	return self.Enter, nil
}

func (self *Active) Enter() {
	// Reset Counter Code
	self.GetContext().ResetCounter()
}

// /// Stopped State
type Stopped struct {
	statechart.StateDefault[MyContext]
}

func (self *Stopped) Setup(proxy statechart.StateSetupProxy[MyContext]) (statechart.EntryAction, statechart.ExitAction) {
	self.Init(proxy)
	// nil is for no action to take
	statechart.AddSimpleStateTransition[StartStopEv, Running](proxy, nil)
	return nil, nil
}

// /// Running State
type Running struct {
	statechart.StateDefault[MyContext]
}

func (self *Running) Setup(proxy statechart.StateSetupProxy[MyContext]) (statechart.EntryAction, statechart.ExitAction) {
	self.Init(proxy)
	// example of custom handler
	// signature AddCustomReaction[EventType](state State, func(event EventType) Result)
	//todo AddCustomReaction[StartStopEv](self, self.handleStartStop)
	statechart.AddSimpleStateTransition[StartStopEv, Stopped](proxy, nil)
	return self.Enter, self.Exit
}

func (self *Running) Enter() {
	self.GetContext().StartCounter()
}

func (self *Running) Exit() {
	// Stop Counter Code
	self.GetContext().StopCounter()
}

func MakeStopWatch() *statechart.StateMachine[MyContext] {
	s := MyContext{}
	sm := statechart.MakeStateMachine(&s)
	idleState := sm.AddState(&Idle{})
	activeState := sm.AddState(&Active{})
	sm.AddSubState(&Stopped{}, activeState)
	sm.AddSubState(&Running{}, activeState)
	sm.Initialize(idleState)

	return &sm
}

func ExampleStateMachine() {
	sm := MakeStopWatch()
	// example
	sm.DispatchEvent(&ActivateEv{})   // go for Idle to Active to Stopped
	sm.DispatchEvent(&StartStopEv{})  // go from Stopped to Running
	sm.DispatchEvent(&ResetEv{})      // go from Running to Active to Stopped
	sm.DispatchEvent(&DeactivateEv{}) // go from Stopped to Idle
	// output:
	// Reset Counter
	// Start Counter
	// Stop Counter
	// Reset Counter

}

func MakeAsyncStopWatch() *statechart.AsyncStateMachine[MyContext] {
	s := MyContext{}
	sm := statechart.MakeAsyncStateMachine(&s)
	idleState := sm.AddState(&Idle{})
	activeState := sm.AddState(&Active{})
	sm.AddSubState(&Stopped{}, activeState)
	sm.AddSubState(&Running{}, activeState)
	sm.Initialize(idleState)

	return &sm
}

func ExampleAsyncStateMachine() {
	sm := MakeAsyncStopWatch()
	// example
	sm.DispatchEvent(&ActivateEv{})   // go for Idle to Active to Stopped
	sm.DispatchEvent(&StartStopEv{})  // go from Stopped to Running
	sm.DispatchEvent(&ResetEv{})      // go from Running to Active to Stopped
	sm.DispatchEvent(&DeactivateEv{}) // go from Stopped to Idle
	sm.Close()                        // wait for all the event to be processed
	// output:
	// Reset Counter
	// Start Counter
	// Stop Counter
	// Reset Counter

}
