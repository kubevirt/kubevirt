package dependencies

import "reflect"

type ComponentFactory func(CC ComponentCache, which string) (interface{}, error)

type ComponentKey struct {
	Type  reflect.Type
	which string
}

type ComponentCache struct {
	components map[ComponentKey]interface{}
	factories  map[ComponentKey]ComponentFactory
}

func NewComponentCache() ComponentCache {
	cc := ComponentCache{
		components: make(map[ComponentKey]interface{}),
		factories:  make(map[ComponentKey]ComponentFactory),
	}
	return cc
}

func (cc ComponentCache) Register(Type reflect.Type, factory ComponentFactory) {
	var which string
	which = ""
	key := ComponentKey{Type, which}
	cc.factories[key] = factory
}

func (cc ComponentCache) RegisterFactory(Type reflect.Type, which string, factory ComponentFactory) {
	key := ComponentKey{Type, which}
	cc.factories[key] = factory
}

func (cc ComponentCache) FetchComponent(Type reflect.Type, which string) interface{} {
	key := ComponentKey{Type, which}
	var err error
	if component, ok := cc.components[key]; ok {
		return component
	} else if factory, ok := cc.factories[key]; ok {
		//IDEALLY locked on a per key basis.
		component, err = factory(cc, which)
		if err != nil {
			panic(err)
		}
		cc.components[key] = component
		return component
	} else {
		panic(component)
	}
}

func (cc ComponentCache) Fetch(Type reflect.Type) interface{} {
	return cc.FetchComponent(Type, "")
}

func (cc ComponentCache) Clear() {
	//Note.  I originally tried to create a new map using
	// cc.components = make(map[ComponentKey]interface{})
	// but it left the old values in place.  THus, the brute force method below.
	for k := range cc.components {
		delete(cc.components, k)
	}
}
