// MIT License: https://github.com/hhassoubi/go-statechart/blob/master/LICENSE
// Copyright (c) 2023 Hicham Hassoubi

package statechart

import (
	"reflect"
)

type stateImpl[C any] struct {
	id            StateId
	name          string
	userState     State[C]
	stateMachine  *stateMachineImpl[C]
	events        []EventReaction
	parent        *stateImpl[C]
	isSuperState  bool
	startingState *stateImpl[C]
	enterAction   func()
	exitAction    func()
}

func (s *stateImpl[C]) Name() string {
	return s.name
}

func (s *stateImpl[C]) SetName(name string) {
	s.name = name
}

func (s *stateImpl[C]) AddReaction(container EventReaction) {
	s.events = append(s.events, container)
}

func (s *stateImpl[C]) SetStartingState(state StateId) {
	startingState := s.stateMachine.getState(state)
	if startingState.parent != s {
		panic("Starting State has to be direct child")
	}
	s.startingState = startingState
}

func (s *stateImpl[C]) Transit(state StateId, reaction BaseAction) ReactionResult {
	return s.stateMachine.transit(state, reaction)
}

func (s *stateImpl[C]) Forward() ReactionResult {
	return ReactionResult{status: FORWARD}
}

func (s *stateImpl[C]) Discard() ReactionResult {
	return ReactionResult{status: DISCARD}
}

func (s *stateImpl[C]) Defer() ReactionResult {
	return ReactionResult{status: DEFER}
}

func (s *stateImpl[C]) PostEvent(event Event) {
	s.stateMachine.postedEvents = append(s.stateMachine.postedEvents, event)
}

func (s *stateImpl[C]) GetContext() *C {
	return s.stateMachine.userContext
}

func (s *stateImpl[C]) GetAncestor(ancestorStateId StateId) State[C] {
	ancestor := s.parent
	for ancestor != nil {
		if ancestor.id == ancestorStateId {
			break
		}
		ancestor = ancestor.parent
	}
	if ancestor == nil {
		panic("Ancestor not found")
	}
	return ancestor.userState
}

func (s *stateImpl[C]) FindStateId(selector func(state State[C]) bool) StateId {
	if id, ok := s.stateMachine.findStateId(selector); ok {
		return id
	}
	panic("State not found")
}

func (s *stateImpl[C]) findReaction(e Event) *EventReaction {
	for _, r := range s.events {
		if r.eventSelector(e) {
			return &r
		}
	}
	return nil
}

func (s *stateImpl[C]) processEvent(e Event) ReactionResult {
	logger := s.stateMachine.DebugLogger
	r := s.findReaction(e)
	if r != nil && r.reaction != nil {
		if logger != nil {
			logger("Process Event", "event", reflect.TypeOf(e), "state", s.name)
		}
		return r.reaction(e)
	}
	if s.stateMachine.DebugLogger != nil {
		logger("Forward Event", "event", reflect.TypeOf(e), "state", s.name)
	}
	return ReactionResult{status: FORWARD}
}

func Transit[S any, C any, PS StateCst[S, C]](from StateProxy[C]) ReactionResult {
	toId := FindStateId[S, C, PS](from)
	return from.Transit(toId, nil)
}

func TransitWithAction[S any, C any, E any, PS StateCst[S, C], PE EventCst[E]](from StateProxy[C], action Action[E, PE]) ReactionResult {
	toId := FindStateId[S, C, PS](from)
	return from.Transit(toId, ToBaseAction(action))
}

func GetAncestor[S any, C any, PS StateCst[S, C]](state StateProxy[C]) *S {
	// cast to interface is required because of no generic upcast operation
	var ancestor interface{}
	id := FindStateId[S, C, PS](state)
	ancestor = state.GetAncestor(id)
	return ancestor.(*S)
}

////////////////////////////////////////////////////

type stateMachineImpl[C any] struct {
	states         []*stateImpl[C]
	currentState   *stateImpl[C]
	DebugLogger    func(msg string, keysAndValues ...interface{})
	userContext    *C
	initialized    bool
	postedEvents   []Event
	deferredEvents []Event
}

func (sm *stateMachineImpl[C]) addStateImpl(state State[C]) *stateImpl[C] {
	if sm.states == nil {
		sm.states = make([]*stateImpl[C], 0, 10)
	}
	// The only place we use reflection. It is ok because it's not in the hot path
	newStateType := reflect.TypeOf(state)
	selector := func(s State[C]) bool {
		return reflect.TypeOf(s) == newStateType
	}
	if _, ok := sm.findStateId(selector); ok {
		panic("State kind already exist")
	}
	newStateImpl := &stateImpl[C]{id: (StateId)(len(sm.states)), userState: state, stateMachine: sm, events: make([]EventReaction, 0, 16)}
	sm.states = append(sm.states, newStateImpl)
	return newStateImpl
}

func (sm *stateMachineImpl[C]) getState(id StateId) *stateImpl[C] {
	if (int)(id) < len(sm.states) {
		return sm.states[id]
	}
	panic("State not found")
}

func (sm *stateMachineImpl[C]) AddState(state State[C]) StateId {
	if sm.initialized {
		panic("Cannot call AddState after calling Initialized")
	}
	return sm.addStateImpl(state).id
}

func (sm *stateMachineImpl[C]) AddSubState(state State[C], parentId StateId) StateId {
	if sm.initialized {
		panic("Cannot call AddSubState after calling Initialize")
	}
	newStateImpl := sm.addStateImpl(state)
	parentImpl := sm.getState(parentId)
	if parentImpl.userState == state {
		panic("parent can't be self")
	}
	parentImpl.isSuperState = true
	newStateImpl.parent = parentImpl
	return newStateImpl.id
}

func (sm *stateMachineImpl[C]) findStateId(selector func(state State[C]) bool) (StateId, bool) {
	for _, state := range sm.states {
		if selector(state.userState) {
			return state.id, true
		}
	}
	return 0, false
}

func (sm *stateMachineImpl[C]) Initialize(initStateId StateId) {
	if sm.initialized {
		panic("Cannot call Initialize more then once")
	}
	sm.initialized = true
	sm.postedEvents = make([]Event, 0, 10)
	for _, state := range sm.states {
		state.enterAction, state.exitAction = state.userState.Setup(state)
		if len(state.name) == 0 {
			// the default name is struct name
			state.name = reflect.TypeOf(state.userState).Elem().Name()
		}
	}
	nextState := sm.getState(initStateId)
	if nextState.isSuperState {
		if nextState.startingState != nil {
			nextState = nextState.startingState
		} else {
			panic("Not a allowed in UML (SupperState cannot be current). Set a sub-state to initial state, or create an empty initial sate")
			// TODO add build or library flag to support this.
		}
	}
	doEnters(nextState, nil)
	sm.currentState = nextState
}

func (sm *stateMachineImpl[C]) DispatchEvent(event Event) {
	// Add event to the queue first
	sm.postedEvents = append(sm.postedEvents, event)
	for len(sm.postedEvents) > 0 {
		result, nextState := processEvent(sm.currentState, sm.currentState, sm.postedEvents[0])
		if result == TRANSIT {
			if sm.DebugLogger != nil {
				sm.DebugLogger("Change State", "from", sm.currentState.name, "to", nextState.name)
			}
			sm.currentState = nextState
			if len(sm.deferredEvents) > 0 {
				// push deferredEvents to the front of the queue
				sm.postedEvents = append(sm.deferredEvents, sm.postedEvents[1:]...)
				// clear deferredEvents
				sm.deferredEvents = sm.deferredEvents[:0]
				// force a start over
				continue
			}
		} else if result == DEFER {
			sm.deferredEvents = append(sm.deferredEvents, event)
		}
		// pop front
		sm.postedEvents = sm.postedEvents[1:]
	}

}

func processEvent[C any](currentState *stateImpl[C], activeState *stateImpl[C], event Event) (ResultType, *stateImpl[C]) {
	result := activeState.processEvent(event)
	switch result.status {
	case FORWARD:
		if activeState.parent != nil {
			return processEvent(currentState, activeState.parent, event)
		}
		// The top state will discard
		return DISCARD, nil
	case DISCARD:
		return DISCARD, nil
	case TRANSIT:
		if result.targetState == nil {
			panic("next state is empty Transit was not call in the event handler")
		}
		nextState := result.targetState.(*stateImpl[C])
		// if the next state is supper state and has a starting start
		if nextState.isSuperState {
			if nextState.startingState != nil {
				nextState = nextState.startingState
			} else {
				panic("Not a allowed in UML (SupperState cannot be current). Set a sub-state to initial state, or create an empty initial sate")
				// TODO add build or library flag to support this.
			}
		}
		// Find LCN
		lca := findRoot(currentState, nextState)
		// Run all the exits
		doExits(currentState, lca)
		// Run the action
		if result.action != nil {
			result.action(event)
		}
		// Run all the enters
		doEnters(nextState, lca)

		return TRANSIT, nextState
	case DEFER:
		return DEFER, nil
	}
	panic("Invalid ResultType")
}

func (sm *stateMachineImpl[C]) transit(to StateId, transitionAction BaseAction) ReactionResult {
	targetState := sm.getState(to)
	return ReactionResult{TRANSIT, targetState, transitionAction}
}

func calcHierarchyLevel[C any](state *stateImpl[C]) int {
	count := 0
	for state != nil {
		count++
		state = state.parent
	}
	return count
}

func findRoot[C any](left, right *stateImpl[C]) *stateImpl[C] {
	// Note Level could be saved in BaseState
	ll := calcHierarchyLevel(left)
	rl := calcHierarchyLevel(right)
	if ll > rl {
		// swap left and right
		left, right = right, left
		ll, rl = rl, ll
	}
	// move right to the same level as left
	for i := 0; i < rl-ll; i++ {
		right = right.parent
	}
	// find root
	for {
		if left == nil || right == nil {
			panic("Invalid State Machine")
		}
		left = left.parent
		right = right.parent
		if left == right {
			break
		}

	}
	return left
}

func doEnters[C any](state *stateImpl[C], root *stateImpl[C]) {
	if state == root {
		return
	}
	doEnters(state.parent, root)
	if state.enterAction != nil {
		state.enterAction()
	}
}

func doExits[C any](state *stateImpl[C], root *stateImpl[C]) {
	if state == root {
		return
	}
	if state.exitAction != nil {
		state.exitAction()
	}
	doExits(state.parent, root)
}
