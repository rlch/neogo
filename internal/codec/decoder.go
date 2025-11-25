package codec

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Decoder is the interface for all decoders.
// HOT PATH: All Decode implementations use zero reflection - only unsafe pointer arithmetic.
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
	elemSize    uintptr
	allocate    sliceAllocator // Pre-computed allocator (no runtime reflection)
}

func (d *sliceDecoder) Decode(data any, ptr unsafe.Pointer) error {
	if data == nil {
		*(*sliceHeader)(ptr) = sliceHeader{}
		return nil
	}

	// Expect generic slice []any from Neo4j
	src, ok := data.([]any)
	if !ok {
		// Handle single element wrapping: if data is not a slice, wrap it
		// This supports binding a single value (e.g., Node) to a slice target
		src = []any{data}
	}

	count := len(src)
	if count == 0 {
		*(*sliceHeader)(ptr) = sliceHeader{}
		return nil
	}

	// Allocate slice backing array using pre-computed allocator (ZERO reflection)
	valPtr, _, capacity := d.allocate(count)
	
	// Copy header to destination struct field
	// This sets Data, Len, Cap on the struct field
	header := (*sliceHeader)(ptr)
	header.Data = valPtr
	header.Len = count
	header.Cap = capacity

	// Iterate and decode elements directly into memory
	for i := 0; i < count; i++ {
		// Proper pointer arithmetic: convert to uintptr, add, convert back
		elemPtr := unsafe.Pointer(uintptr(valPtr) + uintptr(i)*d.elemSize)
		if err := d.elemDecoder.Decode(src[i], elemPtr); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
	}

	return nil
}

// --- Pointer Decoder ---

type ptrDecoder struct {
	elemDecoder Decoder
	allocate    ptrAllocator // Pre-computed allocator (ZERO reflection at decode time)
}

func (d *ptrDecoder) Decode(data any, ptr unsafe.Pointer) error {
	if data == nil {
		*(*unsafe.Pointer)(ptr) = nil
		return nil
	}

	// Allocate memory using pre-computed allocator (ZERO reflection)
	mem := d.allocate()
	
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

func bytesDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case []byte:
		*(*[]byte)(ptr) = v
		return nil
	case string:
		*(*[]byte)(ptr) = []byte(v)
		return nil
	default:
		return fmt.Errorf("expected []byte or string, got %T", data)
	}
}

func int8Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToInt8(data)
	if !ok {
		return fmt.Errorf("expected int8, got %T", data)
	}
	*(*int8)(ptr) = v
	return nil
}

func int16Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToInt16(data)
	if !ok {
		return fmt.Errorf("expected int16, got %T", data)
	}
	*(*int16)(ptr) = v
	return nil
}

func int32Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToInt32(data)
	if !ok {
		return fmt.Errorf("expected int32, got %T", data)
	}
	*(*int32)(ptr) = v
	return nil
}

func uint8Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToUint8(data)
	if !ok {
		return fmt.Errorf("expected uint8, got %T", data)
	}
	*(*uint8)(ptr) = v
	return nil
}

func uint16Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToUint16(data)
	if !ok {
		return fmt.Errorf("expected uint16, got %T", data)
	}
	*(*uint16)(ptr) = v
	return nil
}

func uint32Decoder(data any, ptr unsafe.Pointer) error {
	v, ok := convertToUint32(data)
	if !ok {
		return fmt.Errorf("expected uint32, got %T", data)
	}
	*(*uint32)(ptr) = v
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

func dateDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case neo4j.Date:
		*(*neo4j.Date)(ptr) = v
	case time.Time:
		*(*neo4j.Date)(ptr) = neo4j.DateOf(v)
	default:
		return fmt.Errorf("cannot decode %T into neo4j.Date", data)
	}
	return nil
}

func localTimeDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case neo4j.LocalTime:
		*(*neo4j.LocalTime)(ptr) = v
	case time.Time:
		*(*neo4j.LocalTime)(ptr) = neo4j.LocalTimeOf(v)
	default:
		return fmt.Errorf("cannot decode %T into neo4j.LocalTime", data)
	}
	return nil
}

func localDateTimeDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case neo4j.LocalDateTime:
		*(*neo4j.LocalDateTime)(ptr) = v
	case time.Time:
		*(*neo4j.LocalDateTime)(ptr) = neo4j.LocalDateTimeOf(v)
	default:
		return fmt.Errorf("cannot decode %T into neo4j.LocalDateTime", data)
	}
	return nil
}

func neo4jTimeDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case neo4j.Time:
		*(*neo4j.Time)(ptr) = v
	case time.Time:
		*(*neo4j.Time)(ptr) = neo4j.Time(v)
	default:
		return fmt.Errorf("cannot decode %T into neo4j.Time", data)
	}
	return nil
}

func durationDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case neo4j.Duration:
		*(*neo4j.Duration)(ptr) = v
	default:
		return fmt.Errorf("cannot decode %T into neo4j.Duration", data)
	}
	return nil
}

func point2DDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case neo4j.Point2D:
		*(*neo4j.Point2D)(ptr) = v
	default:
		return fmt.Errorf("cannot decode %T into neo4j.Point2D", data)
	}
	return nil
}

func point3DDecoder(data any, ptr unsafe.Pointer) error {
	switch v := data.(type) {
	case neo4j.Point3D:
		*(*neo4j.Point3D)(ptr) = v
	default:
		return fmt.Errorf("cannot decode %T into neo4j.Point3D", data)
	}
	return nil
}