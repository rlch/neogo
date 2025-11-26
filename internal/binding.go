package internal

// binding.go provides value binding from Neo4j results to Go types.
//
// # Reflection Boundaries
//
// This file is the FALLBACK path for special cases that can't use zero-reflection codecs.
// The primary hot path uses BindingPlan (binding_plan.go) which pre-compiles decoders.
//
// Reflection is used here ONLY for special cases:
//   - Valuer interface implementations (custom unmarshaling)
//   - Abstract node binding (polymorphic type lookup by labels)
//   - Slice depth mismatches (wrapping single values in slices)
//   - Empty interface targets (any/interface{})
//
// For normal struct/primitive binding, use BindingPlan.DecodeSingle() or
// BindingPlan.DecodeMultiple() which delegate to zero-reflection codecs.

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rlch/neogo/internal/codec"
)

// Valuer is an interface for custom marshaling/unmarshaling of Neo4j record values.
// Types implementing this interface will use their custom Unmarshal method instead
// of the default codec decoding.
type Valuer[V neo4j.RecordValue] interface {
	Marshal() (*V, error)
	Unmarshal(*V) error
}

// Bind binds a Neo4j record value to a Go value.
// The 'to' parameter must be a pointer to the target value.
//
// This function delegates to the zero-reflection codec for most cases.
// Special handling is only used for:
// - Custom Valuer implementations
// - Abstract node (polymorphic) binding
// - Slice depth mismatches
// - Empty interface targets
func (r *Registry) Bind(from any, to any) error {
	toV := reflect.ValueOf(to)
	if toV.Kind() != reflect.Ptr {
		return fmt.Errorf("Bind requires pointer target, got %T", to)
	}
	return r.BindValue(from, toV.Elem())
}

// BindValue binds a Neo4j record value to a reflect.Value target.
// Uses zero-reflection codec as the primary path, with reflection fallbacks
// only for special cases that require runtime type inspection.
func (r *Registry) BindValue(from any, to reflect.Value) error {
	toT := to.Type()

	// Handle nil input - requires reflection to set zero value
	if from == nil {
		return r.bindNil(to)
	}

	// Fast path: bind to any/interface{} - direct assignment
	if isEmptyInterface(toT, to) {
		return bindToInterface(from, to)
	}

	// Try Valuer interface (custom unmarshaling) - requires interface check
	if ok, err := r.tryValuer(from, to); ok || err != nil {
		return err
	}

	// Handle abstract nodes (polymorphic lookup by labels)
	// This requires runtime label inspection to find concrete type
	if node, ok := from.(neo4j.Node); ok {
		if r.isAbstractTarget(toT) {
			return r.BindAbstractNode(node, to)
		}
		// Handle single node to slice of abstract types
		innerT := codec.UnwindType(toT)
		if innerT.Kind() == reflect.Slice {
			elemT := innerT.Elem()
			if r.isAbstractTarget(elemT) {
				return r.wrapInSlice(node, to)
			}
		}
	}

	// Handle slice depth mismatch (codec doesn't handle wrapping records)
	if fromT := reflect.TypeOf(from); fromT != nil && fromT.Kind() == reflect.Slice {
		toInnerT := codec.UnwindType(toT)
		if toInnerT.Kind() == reflect.Slice {
			// Direct assignment if types match (e.g., []int to []int)
			// The codec's sliceDecoder expects []any but we might have []T
			if fromT.AssignableTo(toInnerT) {
				target := to
				for target.Kind() == reflect.Ptr {
					if target.IsNil() {
						target.Set(reflect.New(target.Type().Elem()))
					}
					target = target.Elem()
				}
				target.Set(reflect.ValueOf(from))
				return nil
			}
			fromDepth := r.computeSliceDepthRuntime(from)
			toDepth := computeDepth(toInnerT)
			if fromDepth != toDepth {
				return r.bindSliceDepthMismatch(from, to, fromDepth, toDepth)
			}
			// Check if element type is abstract - needs special handling
			elemT := toInnerT.Elem()
			for elemT.Kind() == reflect.Slice {
				elemT = elemT.Elem()
			}
			if r.isAbstractTarget(elemT) {
				return r.bindSliceWithAbstractElements(from, to)
			}
		}
	}

	// Zero-reflection codec path for everything else:
	// - Structs (nodes, relationships)
	// - Primitives (int, string, bool, float, time.Time, etc.)
	// - Slices with matching depth
	// - Pointers
	// - Maps
	return r.decodeWithCodec(from, to)
}

// isEmptyInterface checks if the type is interface{} or *interface{}
func isEmptyInterface(t reflect.Type, v reflect.Value) bool {
	empty := reflect.TypeOf((*any)(nil)).Elem()
	if v.Kind() == reflect.Ptr && t.Elem() == empty {
		return true
	}
	return t == empty && v.CanSet()
}

// bindToInterface assigns any value directly to an interface{} target
func bindToInterface(from any, to reflect.Value) error {
	if to.Kind() == reflect.Ptr {
		to.Elem().Set(reflect.ValueOf(from))
	} else {
		to.Set(reflect.ValueOf(from))
	}
	return nil
}

// bindNil handles nil input by setting target to zero value
func (r *Registry) bindNil(to reflect.Value) error {
	// Unwrap to get the actual type
	target := to
	for target.Kind() == reflect.Ptr && !target.IsNil() {
		target = target.Elem()
	}

	// For slice targets, create a single-element slice with zero value
	if target.Kind() == reflect.Slice {
		target.Set(reflect.MakeSlice(target.Type(), 1, 1))
		return r.BindValue(nil, target.Index(0).Addr())
	}

	// For pointer targets, set to nil (zero)
	if to.Kind() == reflect.Ptr && to.CanSet() {
		to.Set(reflect.Zero(to.Type()))
		return nil
	}

	// For struct/value types, set to zero value
	if target.CanSet() {
		target.Set(reflect.Zero(target.Type()))
	}

	return nil
}

// tryValuer attempts to use the Valuer interface for custom unmarshaling
func (r *Registry) tryValuer(from any, to reflect.Value) (handled bool, err error) {
	// Try each Neo4j record value type
	switch v := from.(type) {
	case neo4j.Node:
		return bindValuer(v, to)
	case neo4j.Relationship:
		return bindValuer(v, to)
	case bool:
		return bindValuer(v, to)
	case int64:
		return bindValuer(v, to)
	case float64:
		return bindValuer(v, to)
	case string:
		return bindValuer(v, to)
	case neo4j.Point2D:
		return bindValuer(v, to)
	case neo4j.Point3D:
		return bindValuer(v, to)
	case neo4j.Date:
		return bindValuer(v, to)
	case neo4j.LocalTime:
		return bindValuer(v, to)
	case neo4j.LocalDateTime:
		return bindValuer(v, to)
	case neo4j.Time:
		return bindValuer(v, to)
	case neo4j.Duration:
		return bindValuer(v, to)
	case time.Time:
		return bindValuer(v, to)
	case []byte:
		return bindValuer(v, to)
	case []any:
		return bindValuer(v, to)
	case map[string]any:
		return bindValuer(v, to)
	}
	return false, nil
}

// bindValuer attempts to use Valuer interface on the target
func bindValuer[V neo4j.RecordValue](value V, to reflect.Value) (ok bool, err error) {
	if !to.CanInterface() {
		return false, nil
	}
	valuer, ok := to.Interface().(Valuer[V])
	if !ok {
		return false, nil
	}
	if err := valuer.Unmarshal(&value); err != nil {
		return false, err
	}
	return true, nil
}

// isAbstractTarget checks if the target type is an abstract interface.
// Delegates to isAbstractType (defined in binding_plan.go) which handles
// pointer unwrapping and IAbstract interface checking.
func (r *Registry) isAbstractTarget(toT reflect.Type) bool {
	return isAbstractType(toT)
}

// computeSliceDepthRuntime computes slice depth at runtime, checking actual element types
// This is needed because []any might contain slices at runtime
func (r *Registry) computeSliceDepthRuntime(from any) int {
	fromV := reflect.ValueOf(from)
	fromT := fromV.Type()
	depth := computeDepth(fromT)

	// Special case: []any might contain slices at runtime
	if fromT.Elem().Kind() == reflect.Interface && fromV.Len() > 0 {
		firstElem := fromV.Index(0).Interface()
		if firstElem != nil && reflect.TypeOf(firstElem).Kind() == reflect.Slice {
			depth++
		}
	}
	return depth
}

// bindSliceDepthMismatch handles cases where source and target slice depths differ
func (r *Registry) bindSliceDepthMismatch(from any, to reflect.Value, fromDepth, toDepth int) error {
	if to.Kind() == reflect.Ptr {
		to = to.Elem()
	}
	if to.Kind() != reflect.Slice {
		return errors.New("cannot bind slice to non-slice type")
	}

	if fromDepth+1 == toDepth {
		// Single record wrapping: from is one level shallower
		// e.g., []Person -> [][]Person (wrap as single result set)
		to.Set(reflect.MakeSlice(to.Type(), 1, 1))
		return r.BindValue(from, to.Index(0))
	}

	return fmt.Errorf("cannot bind slice of depth %d to slice of depth %d", fromDepth, toDepth)
}

// wrapInSlice creates a single-element slice and binds the value to it
func (r *Registry) wrapInSlice(from any, to reflect.Value) error {
	sliceV := to
	for sliceV.Kind() == reflect.Ptr {
		sliceV = sliceV.Elem()
	}
	sliceV.Set(reflect.MakeSlice(sliceV.Type(), 1, 1))
	return r.BindValue(from, sliceV.Index(0).Addr())
}

// bindSliceWithAbstractElements handles binding a slice where elements are abstract interfaces
func (r *Registry) bindSliceWithAbstractElements(from any, to reflect.Value) error {
	sliceV := to
	for sliceV.Kind() == reflect.Ptr {
		sliceV = sliceV.Elem()
	}

	fromV := reflect.ValueOf(from)
	n := fromV.Len()

	sliceV.Set(reflect.MakeSlice(sliceV.Type(), n, n))

	for i := 0; i < n; i++ {
		elemV := sliceV.Index(i)
		if elemV.CanAddr() {
			elemV = elemV.Addr()
		}
		fromElem := fromV.Index(i).Interface()
		if err := r.BindValue(fromElem, elemV.Elem()); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
	}
	return nil
}

// decodeWithCodec uses the zero-reflection codec for struct/primitive decoding
func (r *Registry) decodeWithCodec(from any, to reflect.Value) error {
	if to.Kind() == reflect.Ptr {
		return r.codecs.Decode(from, to.Interface())
	}
	if to.CanAddr() {
		return r.codecs.Decode(from, to.Addr().Interface())
	}
	return fmt.Errorf("cannot bind to non-pointer/unaddressable value %T", to.Interface())
}

// BindAbstractNode binds a Neo4j node to an abstract (interface) type
// by looking up the concrete implementation based on node labels.
func (r *Registry) BindAbstractNode(node neo4j.Node, to reflect.Value) error {
	typ := to.Type()
	if !typ.Implements(rAbstract) && !typ.Elem().Implements(rAbstract) {
		return errors.New("cannot bind abstract node to non-abstract type. Ensure your binding type or the value it references implements IAbstract")
	}

	implNode, err := r.GetConcreteImplementation(node.Labels)
	if err != nil {
		return err
	}

	toImpl := reflect.New(implNode.Type())
	if err := r.codecs.Decode(node, toImpl.Interface()); err != nil {
		return err
	}

	reflect.Indirect(to).Set(toImpl)
	return nil
}

// computeDepth returns the nesting depth of a slice type
func computeDepth(t reflect.Type) (depth int) {
	for t.Kind() == reflect.Slice {
		depth++
		t = t.Elem()
	}
	return depth
}
