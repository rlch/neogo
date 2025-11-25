package codec

import (
	"time"
	"unsafe"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// OpType represents different types of encoding/decoding operations
type OpType int

const (
	// Struct operations
	OpStructStart OpType = iota
	OpStructEnd

	// Basic field operations
	OpFieldBool
	OpFieldInt
	OpFieldInt8
	OpFieldInt16
	OpFieldInt32
	OpFieldInt64
	OpFieldUint
	OpFieldUint8
	OpFieldUint16
	OpFieldUint32
	OpFieldUint64
	OpFieldFloat32
	OpFieldFloat64
	OpFieldString
	OpFieldBytes

	// Neo4j specific types
	OpFieldTime
	OpFieldDate
	OpFieldLocalTime
	OpFieldLocalDateTime
	OpFieldDuration
	OpFieldPoint2D
	OpFieldPoint3D

	// Complex field operations
	OpFieldSlice
	OpFieldMap
	OpFieldPtr
	OpFieldStruct
	OpFieldInterface

	// Special operations
	OpFieldEmbed
	OpFieldSkip

	// Optimized combined operations
	OpStructFieldStart
	OpFieldStructEnd

	OpEnd
)

// Opcode represents a single encoding/decoding operation
type Opcode struct {
	Op   OpType
	Key  []byte    // Field name for encoding (pre-computed)
	Meta FieldMeta // Pre-computed field metadata
	Next *Opcode   // Next operation in sequence

	// For complex types
	SubOpcodes *Opcode // For slices, maps, structs, pointers

	// Field info from db tags
	DBName  string
	Options []string
	IsEmbed bool
	IsSkip  bool
}

// EncodeStruct executes the opcode sequence to encode a struct to map[string]any.
// HOT PATH: Zero reflection - uses only pre-compiled opcodes and unsafe pointer arithmetic.
func EncodeStruct(head *Opcode, structPtr unsafe.Pointer) (map[string]any, error) {
	result := make(map[string]any)
	err := encodeStructToMap(head, structPtr, result)
	return result, err
}

func encodeStructToMap(head *Opcode, structPtr unsafe.Pointer, result map[string]any) error {
	current := head

	for current != nil {
		ptr := unsafe.Pointer(uintptr(structPtr) + current.Meta.Offset)

		switch current.Op {
		case OpStructStart:
			// No-op
		case OpStructEnd:
			// No-op
		case OpStructFieldStart:
			// Optimization hook
		case OpFieldBool:
			result[current.DBName] = *(*bool)(ptr)
		case OpFieldInt:
			result[current.DBName] = int64(*(*int)(ptr))
		case OpFieldInt8:
			result[current.DBName] = int64(*(*int8)(ptr))
		case OpFieldInt16:
			result[current.DBName] = int64(*(*int16)(ptr))
		case OpFieldInt32:
			result[current.DBName] = int64(*(*int32)(ptr))
		case OpFieldInt64:
			result[current.DBName] = *(*int64)(ptr)
		case OpFieldUint:
			result[current.DBName] = int64(*(*uint)(ptr))
		case OpFieldUint8:
			result[current.DBName] = int64(*(*uint8)(ptr))
		case OpFieldUint16:
			result[current.DBName] = int64(*(*uint16)(ptr))
		case OpFieldUint32:
			result[current.DBName] = int64(*(*uint32)(ptr))
		case OpFieldUint64:
			// Warning: Potential overflow if > MaxInt64
			result[current.DBName] = int64(*(*uint64)(ptr))
		case OpFieldFloat32:
			result[current.DBName] = float64(*(*float32)(ptr))
		case OpFieldFloat64:
			result[current.DBName] = *(*float64)(ptr)
		case OpFieldString:
			result[current.DBName] = *(*string)(ptr)
		case OpFieldBytes:
			result[current.DBName] = *(*[]byte)(ptr)
		case OpFieldTime:
			result[current.DBName] = *(*time.Time)(ptr)
		case OpFieldDate:
			result[current.DBName] = *(*neo4j.Date)(ptr)
		case OpFieldLocalTime:
			result[current.DBName] = *(*neo4j.LocalTime)(ptr)
		case OpFieldLocalDateTime:
			result[current.DBName] = *(*neo4j.LocalDateTime)(ptr)
		case OpFieldDuration:
			result[current.DBName] = *(*neo4j.Duration)(ptr)
		case OpFieldPoint2D:
			result[current.DBName] = *(*neo4j.Point2D)(ptr)
		case OpFieldPoint3D:
			result[current.DBName] = *(*neo4j.Point3D)(ptr)

		case OpFieldPtr:
			// Read the pointer
			p := *(*unsafe.Pointer)(ptr)
			if p == nil {
				result[current.DBName] = nil
			} else {
				// Encode the pointed value using SubOpcodes
				// If SubOpcodes is OpFieldStruct (nested), we expect a map.
				// If SubOpcodes is primitive, we expect a value.
				// But Opcode structure assumes linear list. SubOpcodes is usually the start of a chain for the inner type.
				
				// Simplified: assume SubOpcodes handles the inner type encoding.
				// We need a helper "encodeValue" that returns 'any'.
				val, err := EncodeAny(current.SubOpcodes, p)
				if err != nil {
					return err
				}
				result[current.DBName] = val
			}

		case OpFieldSlice:
			// Zero-reflection slice iteration
			header := (*sliceHeader)(ptr)
			if header.Data == nil && header.Len == 0 {
				result[current.DBName] = nil
			} else {
				sliceOut := make([]any, header.Len)
				elemSize := current.Meta.Size

				for i := 0; i < header.Len; i++ {
					// Proper pointer arithmetic: convert to uintptr, add, convert back
					elemPtr := unsafe.Pointer(uintptr(header.Data) + uintptr(i)*elemSize)
					val, err := EncodeAny(current.SubOpcodes, elemPtr)
					if err != nil {
						return err
					}
					sliceOut[i] = val
				}
				result[current.DBName] = sliceOut
			}

		case OpFieldStruct:
			// Nested struct
			// We need to encode it into a map
			val, err := EncodeAny(current.SubOpcodes, ptr)
			if err != nil {
				return err
			}
			result[current.DBName] = val

		case OpFieldEmbed:
			// Embedded struct - merge into current result
			// We iterate the SubOpcodes and write directly to 'result'
			err := encodeStructToMap(current.SubOpcodes, ptr, result)
			if err != nil {
				return err
			}

		case OpFieldInterface:
			// Extract interface{} value directly from memory (ZERO reflection)
			// This avoids reflect.NewAt which allocates a reflect.Value wrapper
			v := getInterfaceValue(ptr)
			result[current.DBName] = v
			// Note: This preserves the interface{} contents as-is.
			// If the interface holds a struct/slice, it will be encoded on next round.
			// This is the correct behavior for Neo4j compatibility.

		case OpFieldSkip:
			// Skip
		case OpEnd:
			// Continue loop to finish? No, loop condition is current != nil
		}
		current = current.Next
	}
	return nil
}

// EncodeAny encodes a single value based on the opcode.
// HOT PATH: Zero reflection - uses only pre-compiled opcodes and unsafe pointer arithmetic.
func EncodeAny(op *Opcode, ptr unsafe.Pointer) (any, error) {
	if op == nil {
		return nil, nil
	}
	
	switch op.Op {
	case OpStructStart:
		// It's a struct, so we expect a sequence of fields.
		// We return a map[string]any
		res := make(map[string]any)
		err := encodeStructToMap(op, ptr, res)
		return res, err
		
	case OpFieldBool:
		return *(*bool)(ptr), nil
	case OpFieldInt:
		return int64(*(*int)(ptr)), nil
	case OpFieldInt8:
		return int64(*(*int8)(ptr)), nil
	case OpFieldInt16:
		return int64(*(*int16)(ptr)), nil
	case OpFieldInt32:
		return int64(*(*int32)(ptr)), nil
	case OpFieldInt64:
		return *(*int64)(ptr), nil
	case OpFieldUint:
		return int64(*(*uint)(ptr)), nil
	case OpFieldUint8:
		return int64(*(*uint8)(ptr)), nil
	case OpFieldUint16:
		return int64(*(*uint16)(ptr)), nil
	case OpFieldUint32:
		return int64(*(*uint32)(ptr)), nil
	case OpFieldUint64:
		return int64(*(*uint64)(ptr)), nil
	case OpFieldFloat32:
		return float64(*(*float32)(ptr)), nil
	case OpFieldFloat64:
		return *(*float64)(ptr), nil
	case OpFieldString:
		return *(*string)(ptr), nil
	case OpFieldBytes:
		return *(*[]byte)(ptr), nil
	case OpFieldTime:
		return *(*time.Time)(ptr), nil
	case OpFieldDate:
		return *(*neo4j.Date)(ptr), nil
	case OpFieldLocalTime:
		return *(*neo4j.LocalTime)(ptr), nil
	case OpFieldLocalDateTime:
		return *(*neo4j.LocalDateTime)(ptr), nil
	case OpFieldDuration:
		return *(*neo4j.Duration)(ptr), nil
	case OpFieldPoint2D:
		return *(*neo4j.Point2D)(ptr), nil
	case OpFieldPoint3D:
		return *(*neo4j.Point3D)(ptr), nil
		
	case OpFieldPtr:
		p := *(*unsafe.Pointer)(ptr)
		if p == nil {
			return nil, nil
		}
		return EncodeAny(op.SubOpcodes, p)
		
	case OpFieldSlice:
		header := (*sliceHeader)(ptr)
		if header.Data == nil && header.Len == 0 {
			return nil, nil
		}
		sliceOut := make([]any, header.Len)
		elemSize := op.Meta.Size

		for i := 0; i < header.Len; i++ {
			// Proper pointer arithmetic: convert to uintptr, add, convert back
			elemPtr := unsafe.Pointer(uintptr(header.Data) + uintptr(i)*elemSize)
			val, err := EncodeAny(op.SubOpcodes, elemPtr)
			if err != nil {
				return nil, err
			}
			sliceOut[i] = val
		}
		return sliceOut, nil
		
	case OpFieldStruct:
		// Nested struct not at top level (e.g. slice element)
		// OpFieldStruct usually points to OpStructStart in SubOpcodes?
		// Or it IS the opcode.
		// If op.Op is OpFieldStruct, it likely wraps the struct definition.
		// If SubOpcodes is OpStructStart, we recurse.
		return EncodeAny(op.SubOpcodes, ptr)

	case OpFieldInterface:
		// Extract interface{} value directly from memory (ZERO reflection)
		return getInterfaceValue(ptr), nil
	}
	
	return nil, nil
}