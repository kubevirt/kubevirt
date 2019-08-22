package clientfake

import (
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

// This is used to prepend reactors to the k8s fake client. should be removed when client go is updated.
// TODO: see if we can merge the OLM ClientsetDecorator and this one.

// ClientsetDecorator defines decorator methods for a Clientset.
type ClientsetDecorator interface {
	// PrependReactor adds a reactor to the beginning of the chain.
	PrependReactor(verb, resource string, reaction testing.ReactionFunc)
}

// ReactionForwardingClientsetDecorator wraps a Clientset and "forwards" Action object mutations
// from all successful non-handling Reactors along the chain to the first handling Reactor. This is
// is a stopgap until we can upgrade to client-go v11.0, where the behavior is the default
// (see https://github.com/kubernetes/client-go/blob/6ee68ca5fd8355d024d02f9db0b3b667e8357a0f/testing/fake.go#L130).
type ReactionForwardingClientsetDecorator struct {
	fake.Clientset
	ReactionChain []testing.Reactor // shadow embedded ReactionChain
	actions       []testing.Action  // these may be castable to other types, but "Action" is the minimum
}

// NewReactionForwardingClientsetDecorator returns the ReactionForwardingClientsetDecorator wrapped Clientset result
// of calling NewSimpleClientset with the given objects.
func NewReactionForwardingClientsetDecorator(objects []runtime.Object, options ...Option) *ReactionForwardingClientsetDecorator {
	decorator := &ReactionForwardingClientsetDecorator{
		Clientset: *fake.NewSimpleClientset(objects...),
	}

	// Swap out the embedded ReactionChain with a Reactor that reduces over the decorator's ReactionChain.
	decorator.ReactionChain = decorator.Clientset.ReactionChain
	decorator.Clientset.ReactionChain = []testing.Reactor{&testing.SimpleReactor{"*", "*", decorator.reduceReactions}}

	// Apply options
	for _, option := range options {
		option(decorator)
	}

	return decorator
}

// reduceReactions reduces over all reactions in the chain while "forwarding" Action object mutations
// from all successful non-handling Reactors along the chain to the first handling Reactor.
func (c *ReactionForwardingClientsetDecorator) reduceReactions(action testing.Action) (handled bool, ret runtime.Object, err error) {
	// The embedded Client set is already locked, so there's no need to lock again
	actionCopy := action.DeepCopy()
	c.actions = append(c.actions, action.DeepCopy())
	for _, reactor := range c.ReactionChain {
		if !reactor.Handles(actionCopy) {
			continue
		}

		handled, ret, err = reactor.React(actionCopy)
		if !handled {
			continue
		}

		return
	}

	return
}

// PrependReactor adds a reactor to the beginning of the chain.
func (c *ReactionForwardingClientsetDecorator) PrependReactor(verb, resource string, reaction testing.ReactionFunc) {
	c.ReactionChain = append([]testing.Reactor{&testing.SimpleReactor{verb, resource, reaction}}, c.ReactionChain...)
}
