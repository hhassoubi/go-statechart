package statechart

import (
	"testing"

	"github.com/stretchr/testify/assert"
	//	"github.com/stretchr/testify/require"
)

type ExpectedCall struct {
	expectedCallCount int
	calls             int
	t                 *testing.T
}

func (c *ExpectedCall) Reset(t *testing.T, count int) {
	c.expectedCallCount = count
	c.calls = 0
	c.t = t
}

func (c *ExpectedCall) ResetNoLimit(t *testing.T) {
	c.expectedCallCount = -1
	c.calls = 0
	c.t = t
}

func (c *ExpectedCall) Call(args ...any) {
	c.calls += 1
	if c.expectedCallCount != -1 && c.calls > c.expectedCallCount {
		if c.t != nil {
			assert.Fail(c.t, "Invalid Call Count")
		} else {
			panic("Invalid Call Count")
		}
	}
}

func (c *ExpectedCall) Validate(expectedCalls int) {
	assert.Equal(c.t, expectedCalls, c.calls)
}

type OnOffTestContext struct {
	OnId          StateId
	OffId         StateId
	OffDefaultId  StateId
	TagId         StateId
	OnEnter       ExpectedCall
	OnExit        ExpectedCall
	OffEnter      ExpectedCall
	OffExit       ExpectedCall
	TagEnter      ExpectedCall
	TagExit       ExpectedCall
	OnToOffAction ExpectedCall
}

type OnEvent struct {
	EventDefault
}

type OffEvent struct {
	EventDefault
}

type ToggleEvent struct {
	EventDefault
}

type TagEvent struct {
	EventDefault
}

type UnTagEvent struct {
	EventDefault
}

type On struct {
	StateDefault[OnOffTestContext]
}

// Setup implements State
func (s *On) Setup(proxy StateSetupProxy[OnOffTestContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	AddSimpleStateTransition[OffEvent, Off](proxy, func(e *OffEvent) { s.GetContext().OnToOffAction.Call(e) })
	AddSimpleStateTransition[ToggleEvent, Off](proxy, func(e *ToggleEvent) { s.GetContext().OnToOffAction.Call(e) })
	AddSimpleStateTransition[TagEvent, OffLockTag](proxy, nil)
	return func() { s.GetContext().OnEnter.Call() }, func() { s.GetContext().OnExit.Call() }
}

type Off struct {
	StateDefault[OnOffTestContext]
}

func (s *Off) Setup(proxy StateSetupProxy[OnOffTestContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	SetStartingState[OffDefault](proxy)
	AddSimpleStateTransition[OnEvent, On](proxy, nil)
	AddSimpleStateTransition[ToggleEvent, On](proxy, nil)
	AddSimpleStateTransition[TagEvent, OffLockTag](proxy, nil)
	return func() { s.GetContext().OffEnter.Call() }, func() { s.GetContext().OffExit.Call() }
}

type OffDefault struct {
	StateDefault[OnOffTestContext]
}

func (s *OffDefault) Setup(proxy StateSetupProxy[OnOffTestContext]) (EntryAction, ExitAction) {
	return nil, nil
}

type OffLockTag struct {
	StateDefault[OnOffTestContext]
}

func (s *OffLockTag) Setup(proxy StateSetupProxy[OnOffTestContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	AddSimpleStateTransition[UnTagEvent, OffDefault](proxy, nil)
	// defer the on
	AddDefer[OnEvent](proxy)
	AddDiscard[OffEvent](proxy)
	AddDiscard[ToggleEvent](proxy)
	return func() { s.GetContext().TagEnter.Call() }, func() { s.GetContext().TagExit.Call() }
}

func MakeOnOffStateMachine(t *testing.T, ctx *OnOffTestContext) *stateMachineImpl[OnOffTestContext] {

	sm := stateMachineImpl[OnOffTestContext]{userContext: ctx}
	on := On{}
	off := Off{}
	ctx.OnId = sm.AddState(&on)
	ctx.OffId = sm.AddState(&off)
	tag := OffLockTag{}
	ctx.TagId = sm.AddSubState(&tag, ctx.OffId)
	ctx.OffDefaultId = sm.AddSubState(&OffDefault{}, ctx.OffId)

	assert.Equal(t, &on, sm.states[ctx.OnId].userState)
	assert.Equal(t, &off, sm.states[ctx.OffId].userState)
	assert.Equal(t, &tag, sm.states[ctx.TagId].userState)

	return &sm
}

func TestAddState(t *testing.T) {
	ctx := OnOffTestContext{}
	sm := MakeOnOffStateMachine(t, &ctx)
	assert.Nil(t, sm.states[ctx.OnId].parent)
	assert.Nil(t, nil, sm.states[ctx.OffId].parent)
	assert.Equal(t, sm.states[ctx.OffId], sm.states[ctx.TagId].parent)

	assert.False(t, sm.states[ctx.OnId].isSuperState)
	assert.True(t, sm.states[ctx.OffId].isSuperState)
	assert.False(t, sm.states[ctx.TagId].isSuperState)

	assert.Equal(t, &ctx, sm.states[ctx.OnId].stateMachine.userContext)
	assert.Equal(t, &ctx, sm.states[ctx.OffId].stateMachine.userContext)
	assert.Equal(t, &ctx, sm.states[ctx.TagId].stateMachine.userContext)
}

func TestInit(t *testing.T) {
	ctx := OnOffTestContext{}
	sm := MakeOnOffStateMachine(t, &ctx)
	ctx.OffEnter.Reset(t, 1)
	sm.Initialize(ctx.OffId)
	assert.Equal(t, ctx.OffDefaultId, sm.currentState.id)
	ctx.OffEnter.Validate(1)

	assert.Equal(t, sm.states[ctx.OffId].userState, sm.states[ctx.TagId].GetAncestor(ctx.OffId))
}

func TestSimpleTransit(t *testing.T) {
	ctx := OnOffTestContext{}
	ctx.OffEnter.Reset(t, 1)
	ctx.OffExit.Reset(t, 1)
	ctx.OnEnter.Reset(t, 1)
	sm := MakeOnOffStateMachine(t, &ctx)
	sm.Initialize(ctx.OffId)
	sm.DispatchEvent(&OnEvent{})
	ctx.OnEnter.Validate(1)
	ctx.OffEnter.Validate(1)
	ctx.OffExit.Validate(1)
}

func TestSimpleTransitWithAction(t *testing.T) {
	ctx := OnOffTestContext{}
	ctx.OffEnter.ResetNoLimit(t)
	ctx.OffExit.ResetNoLimit(t)
	ctx.OnEnter.ResetNoLimit(t)
	ctx.OnExit.ResetNoLimit(t)
	ctx.OnToOffAction.ResetNoLimit(t)
	sm := MakeOnOffStateMachine(t, &ctx)
	sm.Initialize(ctx.OffId)
	sm.DispatchEvent(&OnEvent{})
	sm.DispatchEvent(&OffEvent{})
	ctx.OnToOffAction.Validate(1)

}

func TestSimpleTransitSubState(t *testing.T) {
	ctx := OnOffTestContext{}
	ctx.OffEnter.ResetNoLimit(t)
	ctx.OffExit.ResetNoLimit(t)
	ctx.OnEnter.ResetNoLimit(t)
	ctx.OnExit.ResetNoLimit(t)
	ctx.OnToOffAction.ResetNoLimit(t)
	ctx.TagEnter.ResetNoLimit(t)
	ctx.TagExit.ResetNoLimit(t)

	sm := MakeOnOffStateMachine(t, &ctx)
	sm.Initialize(ctx.OffId)
	sm.DispatchEvent(&OnEvent{})
	sm.DispatchEvent(&TagEvent{})
	ctx.OnEnter.Validate(1)
	ctx.OnExit.Validate(1)
	ctx.OffEnter.Validate(2)
	ctx.OffExit.Validate(1)
	ctx.TagEnter.Validate(1)
	ctx.TagExit.Validate(0)
	sm.DispatchEvent(&ToggleEvent{})
	assert.Equal(t, ctx.TagId, sm.currentState.id)
	sm.DispatchEvent(&OffEvent{})
	assert.Equal(t, ctx.TagId, sm.currentState.id)

	sm.DispatchEvent(&UnTagEvent{})
	assert.Equal(t, ctx.OffDefaultId, sm.currentState.id)
	ctx.OnEnter.Validate(1)
	ctx.OnExit.Validate(1)
	ctx.OffEnter.Validate(2)
	ctx.OffExit.Validate(1)
	ctx.TagEnter.Validate(1)
	ctx.TagExit.Validate(1)

}

func TestDefer(t *testing.T) {
	ctx := OnOffTestContext{}
	ctx.OffEnter.ResetNoLimit(t)
	ctx.OffExit.ResetNoLimit(t)
	ctx.OnEnter.ResetNoLimit(t)
	ctx.OnExit.ResetNoLimit(t)
	ctx.OnToOffAction.ResetNoLimit(t)
	ctx.TagEnter.ResetNoLimit(t)
	ctx.TagExit.ResetNoLimit(t)

	sm := MakeOnOffStateMachine(t, &ctx)
	sm.Initialize(ctx.OffId)
	sm.DispatchEvent(&TagEvent{})

	ctx.OnEnter.Validate(0)
	ctx.OnExit.Validate(0)
	ctx.OffEnter.Validate(1)
	ctx.OffExit.Validate(0)
	ctx.TagEnter.Validate(1)
	ctx.TagExit.Validate(0)

	sm.DispatchEvent(&OnEvent{})
	assert.Equal(t, ctx.TagId, sm.currentState.id)

	ctx.OnEnter.Validate(0)
	ctx.OnExit.Validate(0)
	ctx.OffEnter.Validate(1)
	ctx.OffExit.Validate(0)
	ctx.TagEnter.Validate(1)
	ctx.TagExit.Validate(0)

	sm.DispatchEvent(&UnTagEvent{})
	assert.Equal(t, ctx.OnId, sm.currentState.id)
	ctx.OnEnter.Validate(1)
	ctx.OnExit.Validate(0)
	ctx.OffEnter.Validate(1)
	ctx.OffExit.Validate(1)
	ctx.TagEnter.Validate(1)
	ctx.TagExit.Validate(1)

}

type TestEvent struct {
	EventDefault
}

type TestState1 struct {
	StateDefault[int]
}

func (s *TestState1) Setup(proxy StateSetupProxy[int]) (EntryAction, ExitAction) {
	AddSimpleStateTransition[TestEvent, TestState1](proxy, nil)
	onEnter := func() {
		*(proxy.GetContext()) += 1
	}
	return onEnter, nil
}

func TestReenter(t *testing.T) {
	ctx := 0
	sm := stateMachineImpl[int]{userContext: &ctx}
	id := sm.AddState(&TestState1{})
	sm.Initialize(id)
	sm.DispatchEvent(&TestEvent{})
	assert.Equal(t, 2, ctx)
}

type CallOrderContext struct {
	calls string
}

type StateA struct {
	StateDefault[CallOrderContext]
}

func (s *StateA) Setup(proxy StateSetupProxy[CallOrderContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	return func() { s.GetContext().calls += "A() " }, func() { s.GetContext().calls += "~A() " }
}

type StateB struct {
	StateDefault[CallOrderContext]
}

func (s *StateB) Setup(proxy StateSetupProxy[CallOrderContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	return func() { s.GetContext().calls += "B() " }, func() { s.GetContext().calls += "~B() " }
}

type StateC struct {
	StateDefault[CallOrderContext]
}

func (s *StateC) Setup(proxy StateSetupProxy[CallOrderContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	AddSimpleStateTransition[TestEvent, StateZ](proxy, func(evt *TestEvent) { s.GetContext().calls += "Action() " })
	return func() { s.GetContext().calls += "C() " }, func() { s.GetContext().calls += "~C() " }
}

type StateX struct {
	StateDefault[CallOrderContext]
}

func (s *StateX) Setup(proxy StateSetupProxy[CallOrderContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	return func() { s.GetContext().calls += "X() " }, func() { s.GetContext().calls += "~X() " }
}

type StateY struct {
	StateDefault[CallOrderContext]
}

func (s *StateY) Setup(proxy StateSetupProxy[CallOrderContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	return func() { s.GetContext().calls += "Y() " }, func() { s.GetContext().calls += "~Y() " }
}

type StateZ struct {
	StateDefault[CallOrderContext]
}

func (s *StateZ) Setup(proxy StateSetupProxy[CallOrderContext]) (EntryAction, ExitAction) {
	s.Init(proxy)
	return func() { s.GetContext().calls += "Z() " }, func() { s.GetContext().calls += "~Z() " }
}

func TestCallOrder(t *testing.T) {
	ctx := CallOrderContext{}
	sm := stateMachineImpl[CallOrderContext]{userContext: &ctx}
	firstId := sm.AddSubState(&StateC{}, sm.AddSubState(&StateB{}, sm.AddState(&StateA{})))
	sm.AddSubState(&StateZ{}, sm.AddSubState(&StateY{}, sm.AddState(&StateX{})))
	sm.Initialize(firstId)
	assert.Equal(t, "A() B() C() ", ctx.calls)
	ctx.calls = ""
	sm.DispatchEvent(&TestEvent{})
	assert.Equal(t, "~C() ~B() ~A() Action() X() Y() Z() ", ctx.calls)
}
