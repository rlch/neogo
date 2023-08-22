package neogo

import "github.com/rlch/neogo/hooks"

type registry struct {
	hooks         *hooks.Registry
	abstractNodes []IAbstract
	nodes         []INode
	relationships []IRelationship
}

func (d *registry) UseHooks(hooks ...*hooks.Hook) {
	d.hooks.UseHooks(hooks...)
}

func (d *registry) RemoveHooks(names ...string) {
	d.hooks.RemoveHooks(names...)
}

// WithTypes is an option for [New] that allows you to register instances of
// [IAbstract], [INode] and [IRelationship] to be used with [neogo].
func WithTypes(types ...any) func(*driver) {
	return func(d *driver) {
		for _, t := range types {
			if v, ok := t.(IAbstract); ok {
				d.abstractNodes = append(d.abstractNodes, v)
				continue
			}
			if v, ok := t.(INode); ok {
				d.nodes = append(d.nodes, v)
				continue
			}
			if v, ok := t.(IRelationship); ok {
				d.relationships = append(d.relationships, v)
				continue
			}
		}
	}
}
