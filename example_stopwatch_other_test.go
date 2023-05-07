// MIT License: https://github.com/hhassoubi/go-statechart/blob/master/LICENSE
// Copyright (c) 2023 Hicham Hassoubi

package statechart_test

import (
	"github.com/hhassoubi/go-statechart"
)

// This Example Creates an Async State Machine. Please refer to [ExampleStateMachine]
// the link over here https://github.com/hhassoubi/go-statechart/blob/master/async_state_machine.go
func ExampleAsyncStateMachine() {
	// Refer to [https://github.com/hhassoubi/go-statechart/blob/master/async_state_machine.go]
	// Create State Machine
	context := MyContext{}
	sm := statechart.MakeAsyncStateMachine(&context)
	idleState := sm.AddState(&Idle{})
	activeState := sm.AddState(&Active{})
	sm.AddSubState(&Stopped{}, activeState)
	sm.AddSubState(&Running{}, activeState)
	sm.Initialize(idleState)

	// example event dispatch
	sm.DispatchEvent(&ActivateEv{})   // go for Idle to Active to Stopped
	sm.DispatchEvent(&StartStopEv{})  // go from Stopped to Running
	sm.DispatchEvent(&ResetEv{})      // go from Running to Active to Stopped
	sm.DispatchEvent(&DeactivateEv{}) // go from Stopped to Idle
	sm.Close()                        // this will wait

	// output:
	// Reset Counter
	// Start Counter
	// Stop Counter
	// Reset Counter

}
