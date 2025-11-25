package internal

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

// BindValue binds a Neo4j record value to a Go value.
// The 'to' parameter must be a settable reflect.Value (typically an addressable value).
//
// This function handles:
// - Neo4j nodes and relationships (extracts Props)
// - Slices (with depth matching)
// - Custom Valuer implementations
// - Primitive type coercion
// - Struct decoding via compiled codecs (zero reflection)
func (r *Registry) BindValue(from any, to reflect.Value) error {
	toT := to.Type()

	// Fast path: bind to any/interface{}
	if isEmptyInterface(toT, to) {
		return bindToInterface(from, to)
	}

	// Handle nil input
	if from == nil {
		return r.bindNil(to)
	}

	// Try Valuer interface first (custom unmarshaling)
	if ok, err := r.tryValuer(from, to); ok || err != nil {
		return err
	}

	// Handle Neo4j types that need unwrapping
	switch v := from.(type) {
	case neo4j.Node:
		return r.bindNode(v, to, toT)
	case neo4j.Relationship:
		return r.bindRelationship(v, to, toT)
	}

	// Handle slice input
	if reflect.TypeOf(from).Kind() == reflect.Slice {
		return r.bindSlice(from, to, toT)
	}

	// Handle single value -> slice wrapping (non-slice from to slice to)
	innerToT := codec.UnwindType(toT)
	if innerToT.Kind() == reflect.Slice {
		return r.wrapInSlice(from, to)
	}

	// Handle primitive types directly (codec expects map/Node for struct decoding)
	if ok, err := r.bindPrimitive(from, to); ok || err != nil {
		return err
	}

	// Delegate to codec for struct decoding (zero reflection hot path)
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

// bindNil handles nil input
// - For slice targets: creates slice with one nil element
// - For pointer targets: sets the pointer to nil (zero value)
func (r *Registry) bindNil(to reflect.Value) error {
	// Unwrap to get the actual type
	target := to
	for target.Kind() == reflect.Ptr && !target.IsNil() {
		target = target.Elem()
	}

	// For slice targets, create a single-element slice
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

// bindPrimitive handles direct primitive type binding
func (r *Registry) bindPrimitive(from any, to reflect.Value) (handled bool, err error) {
	// Get the settable value (unwrap pointers)
	target := to
	for target.Kind() == reflect.Ptr {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}

	if !target.CanSet() {
		return false, nil
	}

	// Direct type match - use reflect to set value
	fromV := reflect.ValueOf(from)
	if fromV.Type().AssignableTo(target.Type()) {
		target.Set(fromV)
		return true, nil
	}

	// Handle type conversions
	if fromV.Type().ConvertibleTo(target.Type()) {
		target.Set(fromV.Convert(target.Type()))
		return true, nil
	}

	// Handle time.Time specially
	if target.Type() == reflect.TypeOf(time.Time{}) {
		return r.bindTime(from, target)
	}

	return false, nil
}

// bindTime handles binding various time types to time.Time
func (r *Registry) bindTime(from any, to reflect.Value) (handled bool, err error) {
	var t time.Time
	switch v := from.(type) {
	case time.Time:
		t = v
	case neo4j.Time:
		t = v.Time()
	case neo4j.Date:
		t = v.Time()
	case neo4j.LocalDateTime:
		t = v.Time()
	case neo4j.LocalTime:
		t = v.Time()
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return false, err
		}
		t = parsed
	default:
		return false, nil
	}
	to.Set(reflect.ValueOf(t))
	return true, nil
}

// bindNode handles neo4j.Node binding
func (r *Registry) bindNode(node neo4j.Node, to reflect.Value, toT reflect.Type) error {
	// Handle single node to slice
	if codec.UnwindType(toT).Kind() == reflect.Slice {
		return r.wrapInSlice(node, to)
	}

	// Check for abstract node (polymorphic)
	innerT := toT
	for innerT.Kind() == reflect.Ptr {
		innerT = innerT.Elem()
	}
	if (toT.Implements(rAbstract) || toT.Elem().Implements(rAbstract)) && innerT.Kind() == reflect.Interface {
		return r.BindAbstractNode(node, to)
	}

	// Decode node props to struct
	return r.decodeWithCodec(node, to)
}

// bindRelationship handles neo4j.Relationship binding
func (r *Registry) bindRelationship(rel neo4j.Relationship, to reflect.Value, toT reflect.Type) error {
	// Handle single relationship to slice
	if codec.UnwindType(toT).Kind() == reflect.Slice {
		return r.wrapInSlice(rel, to)
	}

	// Decode relationship props to struct
	return r.decodeWithCodec(rel, to)
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

// bindSlice handles slice input with depth matching
func (r *Registry) bindSlice(from any, to reflect.Value, _ reflect.Type) error {
	if to.Kind() == reflect.Ptr {
		to = to.Elem()
	}
	if to.Kind() != reflect.Slice {
		return errors.New("cannot bind slice to non-slice type")
	}
	toT := to.Type()

	fromV := reflect.ValueOf(from)
	n := fromV.Len()

	// For []any (common from Neo4j), we need to check the runtime type of elements
	// to determine if this is actually a nested slice
	fromT := fromV.Type()
	fromDepth := computeDepth(fromT)
	toDepth := computeDepth(toT)

	// Special case: []any might contain slices at runtime
	// Check first element to determine actual depth
	if fromT.Elem().Kind() == reflect.Interface && n > 0 {
		firstElem := fromV.Index(0).Interface()
		if firstElem != nil && reflect.TypeOf(firstElem).Kind() == reflect.Slice {
			// Elements are slices, so actual depth is one more than static type suggests
			fromDepth++
		}
	}

	if fromDepth == toDepth {
		// 1:1 mapping
		to.Set(reflect.MakeSlice(toT, n, n))
		for i := range n {
			toI := to.Index(i)
			if toI.CanAddr() {
				toI = toI.Addr()
			}
			if err := r.BindValue(fromV.Index(i).Interface(), toI); err != nil {
				return fmt.Errorf("error binding slice element %d: %w", i, err)
			}
		}
		return nil
	}

	if fromDepth+1 == toDepth {
		// Single record wrapping: from is one level shallower
		to.Set(reflect.MakeSlice(toT, 1, 1))
		return r.BindValue(from, to.Index(0))
	}

	return fmt.Errorf("cannot bind slice of depth %d to slice of depth %d", fromDepth, toDepth)
}

// decodeWithCodec uses the zero-reflection codec for struct decoding
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
