package codec

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Decoder is the interface for all decoders
type Decoder interface {
	Decode(data any, ptr unsafe.Pointer) error
}

// DecoderFunc adapts a function to the Decoder interface
type DecoderFunc func(data any, ptr unsafe.Pointer) error

func (f DecoderFunc) Decode(data any, ptr unsafe.Pointer) error {
	return f(data, ptr)
}

// --- Primitive Decoders ---

func intDecoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToInt(data)
	if !ok {
		return fmt.Errorf("expected int, got %T", data)
	}
	*(*int)(ptr) = v
	return nil
}

func int64Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToInt64(data)
	if !ok {
		return fmt.Errorf("expected int64, got %T", data)
	}
	*(*int64)(ptr) = v
	return nil
}

func stringDecoder(data any, ptr unsafe.Pointer) error {
	v, ok := data.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", data)
	}
	*(*string)(ptr) = v
	return nil
}

func boolDecoder(data any, ptr unsafe.Pointer) error {
	v, ok := data.(bool)
	if !ok {
		return fmt.Errorf("expected bool, got %T", data)
	}
	*(*bool)(ptr) = v
	return nil
}

func float64Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToFloat64(data)
	if !ok {
		return fmt.Errorf("expected float64, got %T", data)
	}
	*(*float64)(ptr) = v
	return nil
}

// --- Struct Decoder ---

type structDecoder struct {
	fields map[string]*fieldDecoder // Maps DB field name to decoder info
}

type fieldDecoder struct {
	offset  uintptr
	decoder Decoder
}

func (d *structDecoder) Decode(data any, ptr unsafe.Pointer) error {
	// Neo4j returns map[string]any for properties/nodes
	var props map[string]any

	switch v := data.(type) {
	case map[string]any:
		props = v
	case neo4j.Node:
		props = v.Props
		// TODO: Handle ID/Labels/ElementId mapping if struct has tags for them
	case neo4j.Relationship:
		props = v.Props
	default:
		return fmt.Errorf("expected map or Node, got %T", data)
	}

	for k, v := range props {
		if fd, ok := d.fields[k]; ok {
			fieldPtr := unsafe.Pointer(uintptr(ptr) + fd.offset)
			if err := fd.decoder.Decode(v, fieldPtr); err != nil {
				return fmt.Errorf("error decoding field %q: %w", k, err)
			}
		}
	}
	return nil
}

// --- Slice Decoder ---

type sliceDecoder struct {
	elemDecoder Decoder
	elemType    reflect.Type
	elemSize    uintptr
}

func (d *sliceDecoder) Decode(data any, ptr unsafe.Pointer) error {
	if data == nil {
		*(*sliceHeader)(ptr) = sliceHeader{}
		return nil
	}

	// Expect generic slice []any from Neo4j
	src, ok := data.([]any)
	if !ok {
		// Handle single element unwrapping if necessary (legacy behavior support)
		// For now, strict checking
		return fmt.Errorf("expected []any, got %T", data)
	}

	count := len(src)
	if count == 0 {
		*(*sliceHeader)(ptr) = sliceHeader{}
		return nil
	}

	// Allocate slice underlying array
	// We use reflect.MakeSlice to safely allocate the backing array of correct type
	sliceVal := reflect.MakeSlice(reflect.SliceOf(d.elemType), count, count)
	
	// Copy header to destination struct field
	// This sets Data, Len, Cap on the struct field
	valPtr := unsafe.Pointer(sliceVal.Pointer()) // Pointer to first element
	
	header := (*sliceHeader)(ptr)
	header.Data = valPtr
	header.Len = count
	header.Cap = count

	// Iterate and decode elements directly into memory
	base := uintptr(valPtr)
	for i := 0; i < count; i++ {
		elemPtr := unsafe.Pointer(base + uintptr(i)*d.elemSize)
		if err := d.elemDecoder.Decode(src[i], elemPtr); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
	}

	return nil
}

// --- Map Decoder ---

type mapDecoder struct {
	elemDecoder Decoder
	mapType     reflect.Type
	elemType    reflect.Type
}

func (d *mapDecoder) Decode(data any, ptr unsafe.Pointer) error {
	if data == nil {
		*(*unsafe.Pointer)(ptr) = nil
		return nil
	}

	src, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map[string]any, got %T", data)
	}

	// Create map
	m := reflect.MakeMapWithSize(d.mapType, len(src))
	
	// Temporary value for decoding elements
	// We allocate a new one each time or reuse? Reuse is tricky with reflection.
	// For zero-reflection map writing, we'd need direct runtime map access.
	// Fallback: Decode into new value, then MapIndex.Set
	
	// Optimization: Goccy avoids this by using runtime map functions.
	// For now, we use "Fast Reflection" pattern.
	
	for k, v := range src {
		// Create new element
		elemVal := reflect.New(d.elemType).Elem()
		
		// Decode into it using unsafe (safe because we own the memory)
		elemPtr := unsafe.Pointer(elemVal.UnsafeAddr())
		if err := d.elemDecoder.Decode(v, elemPtr); err != nil {
			return fmt.Errorf("map key %q: %w", k, err)
		}
		
		m.SetMapIndex(reflect.ValueOf(k), elemVal)
	}

	// Set map to struct field
	// ptr points to the map field in the struct (which is a pointer)
	// We need to set the pointer value
	dst := reflect.NewAt(d.mapType, ptr).Elem()
	dst.Set(m)

	return nil
}

// --- Pointer Decoder ---

type ptrDecoder struct {
	elemDecoder Decoder
	elemType    reflect.Type
}

func (d *ptrDecoder) Decode(data any, ptr unsafe.Pointer) error {
	if data == nil {
		*(*unsafe.Pointer)(ptr) = nil
		return nil
	}

	// Allocate memory
	val := reflect.New(d.elemType) // Returns *T wrapped in Value
	mem := unsafe.Pointer(val.Pointer())
	
	// Decode into allocated memory
	if err := d.elemDecoder.Decode(data, mem); err != nil {
		return err
	}

	// Set pointer field to point to new memory
	*(*unsafe.Pointer)(ptr) = mem
	return nil
}

// --- Interface Decoder ---

func interfaceDecoder(data any, ptr unsafe.Pointer) error {
	// Just assign the value directly
	*(*any)(ptr) = data
	return nil
}

// --- Additional Primitive Decoders ---

func uintDecoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToUint(data)
	if !ok {
		return fmt.Errorf("expected uint, got %T", data)
	}
	*(*uint)(ptr) = v
	return nil
}

func uint64Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToUint64(data)
	if !ok {
		return fmt.Errorf("expected uint64, got %T", data)
	}
	*(*uint64)(ptr) = v
	return nil
}

func float32Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToFloat32(data)
	if !ok {
		return fmt.Errorf("expected float32, got %T", data)
	}
	*(*float32)(ptr) = v
	return nil
}

// --- Neo4j Types ---

func timeDecoder(data any, ptr unsafe.Pointer) error {
	// Neo4j might return time.Time (local) or dbtype.Time
	switch v := data.(type) {
	case time.Time:
		*(*time.Time)(ptr) = v
	case neo4j.Time:
		*(*time.Time)(ptr) = v.Time()
	case neo4j.Date:
		*(*time.Time)(ptr) = v.Time()
	case neo4j.LocalDateTime:
		*(*time.Time)(ptr) = v.Time()
	case neo4j.LocalTime:
		*(*time.Time)(ptr) = v.Time()
	case string:
		// Try parsing ISO string?
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		*(*time.Time)(ptr) = t
	default:
		return fmt.Errorf("cannot decode %T into time.Time", data)
	}
	return nil
}
