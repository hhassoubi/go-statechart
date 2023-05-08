package statechart

import (
	"fmt"
	"io"
)

// struct to inverse the tree from (child -> parent) to (parent -> children)
type stateNode[C any] struct {
	self     *stateImpl[C]
	children []*stateNode[C]
}

// Makes a tree of States Parent -> Children direction
// Note This is need to navigate parent to children
func makeStateTree[C any](states []*stateImpl[C]) stateNode[C] {
	root := stateNode[C]{}
	nodes_cache := make(map[StateId]*stateNode[C])

	var dfs func(node *stateImpl[C]) *stateNode[C]
	dfs = func(node *stateImpl[C]) *stateNode[C] {
		if node == nil {
			// we are at the top return root of the tree
			return &root
		} else {
			// check cache
			n, ok := nodes_cache[node.id]
			if ok {
				// found it in cache
				return n
			}
			// not found, create a new one.
			newNode := &stateNode[C]{node, make([]*stateNode[C], 0)}
			// find the parent, too add new node to it
			parent_node := dfs(node.parent)
			// add new node to parent node
			parent_node.children = append(parent_node.children, newNode)
			// update cache
			nodes_cache[node.id] = newNode
			return newNode
		}
	}
	for _, s := range states {
		dfs(s)
	}

	return root
}

// Prints the header of the state
func plantUmlPrintStateHeader[C any](w io.Writer, node *stateNode[C], withParentName bool, tab string) {
	if withParentName && node.self.parent != nil {
		fmt.Fprintf(w, "%sstate \"%s : %s\" as %s {\n", tab, node.self.name, node.self.parent.name, node.self.name)
	} else {
		fmt.Fprintf(w, "%sstate %s {\n", tab, node.self.name)
	}
}

// Print the footer of the state
func plantUmlPrintStateFooter[C any](w io.Writer, node *stateNode[C], tab string) {
	fmt.Fprintf(w, "%s}\n", tab)
}

func plantUmlPrintStateInnerActions[C any](w io.Writer, node *stateNode[C], tab string) {

	if node.self.enterAction != nil {
		fmt.Fprintf(w, "%s%s: entry / With Action \n", tab, node.self.name)
	}
	// exit action
	if node.self.exitAction != nil {
		fmt.Fprintf(w, "%s%s: exit / With Action \n", tab, node.self.name)
	}
	// in state events
	for _, ev := range node.self.events {
		for _, umlDoc := range ev.umlDoc {
			switch umlDoc.ReactionResult {
			case DISCARD:
				if len(umlDoc.ActionText) == 0 {
					fmt.Fprintf(w, "%s%s: %s[%s] / DISCARD \n", tab, node.self.name, ev.docEventName, umlDoc.GuardText)
				} else {
					fmt.Fprintf(w, "%s%s: %s[%s] / %s \n", tab, node.self.name, ev.docEventName, umlDoc.GuardText, umlDoc.ActionText)
				}
			case DEFER:
				fmt.Fprintf(w, "%s%s: %s[%s] / DEFER \n", tab, node.self.name, ev.docEventName, umlDoc.GuardText)
			}
		}
	}

}

// Print the body of the state, and recursively print the sub states
func plantUmlPrintStateBody[C any](w io.Writer, node *stateNode[C], tab string) {

	// the root node has no state element just children
	if node.self != nil {
		plantUmlPrintStateInnerActions(w, node, tab)
		// starting state
		if node.self.isSuperState && node.self.startingState != nil {
			fmt.Fprintf(w, "%s[*] -> %s \n", tab, node.self.startingState.name)
		}
	}
	// children
	for _, n := range node.children {
		plantUmlPrintStateHeader(w, n, false, tab)
		plantUmlPrintStateBody(w, n, tab+"  ")
		plantUmlPrintStateFooter(w, n, tab)
	}
}

// Print the body of the state, and recursively print the sub states
func plantUmlPrintStateBodyFlat[C any](w io.Writer, node *stateNode[C], tab string) {

	// the root node has no state element just children
	if node.self != nil {
		plantUmlPrintStateHeader(w, node, true, tab)
		// starting state
		if node.self.isSuperState {
			fmt.Fprintf(w, "%s%s: Supper-State = True \n", tab+"  ", node.self.name)
			if node.self.startingState != nil {
				fmt.Fprintf(w, "%s%s: Starting-State = %s \n", tab+"  ", node.self.name, node.self.startingState.name)
			}
		}
		plantUmlPrintStateInnerActions(w, node, tab+"  ")

		plantUmlPrintStateFooter(w, node, tab)
	}
	// children
	for _, n := range node.children {
		plantUmlPrintStateBodyFlat(w, n, tab)
	}
}

func plantUmlPrint[C any](w io.Writer, sm *stateMachineImpl[C], diagramType UmlDiagramType) {

	fmt.Fprintf(w, "@startuml\n")
	root := makeStateTree(sm.states)
	if diagramType == HIERARCHY_ONLY || diagramType == HIERARCHY_WITH_TRANSITION {
		plantUmlPrintStateBody(w, &root, "")
	} else if diagramType == FLAT_WITH_TRANSITION {
		plantUmlPrintStateBodyFlat(w, &root, "")
	}

	if diagramType == HIERARCHY_WITH_TRANSITION || diagramType == FLAT_WITH_TRANSITION {
		// print Transitions
		for _, s := range sm.states {
			for _, ev := range s.events {
				for _, umlDoc := range ev.umlDoc {
					switch umlDoc.ReactionResult {
					case TRANSIT:
						toStateName := "Unknown"
						if umlDoc.TargetState != INVALID_STATE_ID {
							toStateName = sm.getState(umlDoc.TargetState).name
						}
						fmt.Fprintf(w, "%s -> %s : %s", s.name, toStateName, ev.docEventName)
						if len(umlDoc.GuardText) != 0 {
							fmt.Fprintf(w, "[%s]", umlDoc.GuardText)
						}

						if len(umlDoc.ActionText) != 0 {
							fmt.Fprintf(w, " / %s", umlDoc.ActionText)
						}
						fmt.Fprintf(w, "\n")
					}
				}
			}
		}
	}

	fmt.Fprintf(w, "@enduml\n")

}
