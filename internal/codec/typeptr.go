package codec

import (
	"reflect"
	"unsafe"
)

// sliceHeader represents the runtime structure of a slice
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// emptyInterface represents the internal structure of interface{}
type emptyInterface struct {
	typ unsafe.Pointer
	ptr unsafe.Pointer
}

// TypePtr represents a unique identifier for a Go type
type TypePtr uintptr

// getTypePtr extracts the type pointer from a reflect.Type
func getTypePtr(t reflect.Type) TypePtr {
	return TypePtr(uintptr((*emptyInterface)(unsafe.Pointer(&t)).ptr))
}

// getTypePtrFromValue extracts the type pointer directly from a value
func getTypePtrFromValue(v any) TypePtr {
	iface := (*emptyInterface)(unsafe.Pointer(&v))
	return TypePtr(uintptr(iface.typ))
}

// getValuePtr extracts the value pointer from interface{}
func getValuePtr(v any) unsafe.Pointer {
	iface := (*emptyInterface)(unsafe.Pointer(&v))
	return iface.ptr
}

// FieldMeta contains pre-computed field metadata (no reflect.Type needed)
type FieldMeta struct {
	Offset uintptr      // Memory offset in struct
	Size   uintptr      // Field size in bytes
	Kind   Kind         // Type kind
	Type   reflect.Type // Go type
}

// Kind represents Go type kinds (avoiding reflect.Kind)
type Kind uint8

const (
	KindBool Kind = iota
	KindInt
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindUint
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindFloat32
	KindFloat64
	KindString
	KindBytes
	KindSlice
	KindMap
	KindStruct
	KindInterface
	KindPtr
)

// getKind converts reflect.Kind to our Kind
func getKind(rk reflect.Kind) Kind {
	switch rk {
	case reflect.Bool:
		return KindBool
	case reflect.Int:
		return KindInt
	case reflect.Int8:
		return KindInt8
	case reflect.Int16:
		return KindInt16
	case reflect.Int32:
		return KindInt32
	case reflect.Int64:
		return KindInt64
	case reflect.Uint:
		return KindUint
	case reflect.Uint8:
		return KindUint8
	case reflect.Uint16:
		return KindUint16
	case reflect.Uint32:
		return KindUint32
	case reflect.Uint64:
		return KindUint64
	case reflect.Float32:
		return KindFloat32
	case reflect.Float64:
		return KindFloat64
	case reflect.String:
		return KindString
	case reflect.Slice:
		if rk == reflect.Slice {
			return KindBytes // Special case for []byte
		}
		return KindSlice
	case reflect.Map:
		return KindMap
	case reflect.Struct:
		return KindStruct
	case reflect.Interface:
		return KindInterface
	case reflect.Ptr:
		return KindPtr
	default:
		return KindInterface
	}
}

// extractFieldMeta extracts field metadata from reflect.StructField
func extractFieldMeta(field reflect.StructField) FieldMeta {
	return FieldMeta{
		Offset: field.Offset,
		Size:   field.Type.Size(),
		Kind:   getKind(field.Type.Kind()),
		Type:   field.Type,
	}
}

// readFieldValue reads a field value using unsafe pointer arithmetic (ZERO reflection)
func (fm FieldMeta) readFieldValue(structPtr unsafe.Pointer) any {
	fieldPtr := unsafe.Pointer(uintptr(structPtr) + fm.Offset)

	switch fm.Kind {
	case KindBool:
		return *(*bool)(fieldPtr)
	case KindInt:
		return *(*int)(fieldPtr)
	case KindInt8:
		return *(*int8)(fieldPtr)
	case KindInt16:
		return *(*int16)(fieldPtr)
	case KindInt32:
		return *(*int32)(fieldPtr)
	case KindInt64:
		return *(*int64)(fieldPtr)
	case KindUint:
		return *(*uint)(fieldPtr)
	case KindUint8:
		return *(*uint8)(fieldPtr)
	case KindUint16:
		return *(*uint16)(fieldPtr)
	case KindUint32:
		return *(*uint32)(fieldPtr)
	case KindUint64:
		return *(*uint64)(fieldPtr)
	case KindFloat32:
		return *(*float32)(fieldPtr)
	case KindFloat64:
		return *(*float64)(fieldPtr)
	case KindString:
		return *(*string)(fieldPtr)
	case KindBytes:
		return *(*[]byte)(fieldPtr)
	case KindSlice:
		// Return slice header as-is - complex decoding handled by opcodes
		return *(*any)(fieldPtr)
	case KindMap:
		// Return map as-is - complex decoding handled by opcodes
		return *(*any)(fieldPtr)
	case KindStruct:
		// Return struct as-is - complex decoding handled by opcodes
		return *(*any)(fieldPtr)
	case KindPtr:
		// Return pointer as-is - complex decoding handled by opcodes
		return *(*any)(fieldPtr)
	case KindInterface:
		// Return interface as-is
		return *(*any)(fieldPtr)
	default:
		// Unknown type - return as interface{}
		return *(*any)(fieldPtr)
	}
}

// TypeInfo contains pre-computed type information extracted at registration time
type TypeInfo struct {
	Name           string                    // Type name
	Fields         map[string]*FieldMeta     // field name -> metadata
	FieldsToProps  map[string]string         // Go field name -> JSON property name
	Neo4jRelations map[string]*Neo4jRelation // field name -> neo4j relation info
	IsNode         bool
	IsRelationship bool
	IsAbstract     bool
	Labels         []string // Node labels
	RelType        string   // Relationship type
}

// Neo4jRelation contains neo4j relationship information
type Neo4jRelation struct {
	Type      string // "startNode", "endNode", or relationship type
	Direction string // "in", "out", or ""
	IsMany    bool   // true for slice relationships
	NodeType  string // target node type name
}

// writeFieldValue writes a value to a field using unsafe pointer arithmetic (ZERO reflection)
func (fm FieldMeta) writeFieldValue(structPtr unsafe.Pointer, value any) {
	fieldPtr := unsafe.Pointer(uintptr(structPtr) + fm.Offset)

	switch fm.Kind {
	case KindBool:
		if v, ok := value.(bool); ok {
			*(*bool)(fieldPtr) = v
		}
	case KindInt:
		if v, ok := convertToInt(value); ok {
			*(*int)(fieldPtr) = v
		}
	case KindInt8:
		if v, ok := convertToInt8(value); ok {
			*(*int8)(fieldPtr) = v
		}
	case KindInt16:
		if v, ok := convertToInt16(value); ok {
			*(*int16)(fieldPtr) = v
		}
	case KindInt32:
		if v, ok := convertToInt32(value); ok {
			*(*int32)(fieldPtr) = v
		}
	case KindInt64:
		if v, ok := convertToInt64(value); ok {
			*(*int64)(fieldPtr) = v
		}
	case KindUint:
		if v, ok := convertToUint(value); ok {
			*(*uint)(fieldPtr) = v
		}
	case KindUint8:
		if v, ok := convertToUint8(value); ok {
			*(*uint8)(fieldPtr) = v
		}
	case KindUint16:
		if v, ok := convertToUint16(value); ok {
			*(*uint16)(fieldPtr) = v
		}
	case KindUint32:
		if v, ok := convertToUint32(value); ok {
			*(*uint32)(fieldPtr) = v
		}
	case KindUint64:
		if v, ok := convertToUint64(value); ok {
			*(*uint64)(fieldPtr) = v
		}
	case KindFloat32:
		if v, ok := convertToFloat32(value); ok {
			*(*float32)(fieldPtr) = v
		}
	case KindFloat64:
		if v, ok := convertToFloat64(value); ok {
			*(*float64)(fieldPtr) = v
		}
	case KindString:
		if v, ok := value.(string); ok {
			*(*string)(fieldPtr) = v
		}
	case KindBytes:
		if v, ok := value.([]byte); ok {
			*(*[]byte)(fieldPtr) = v
		}
	case KindSlice, KindMap, KindStruct, KindPtr, KindInterface:
		// For complex types, direct assignment (no reflection!)
		// Complex decoding should be handled by dedicated opcodes
		*(*any)(fieldPtr) = value
	default:
		// Unknown type - direct assignment as interface{}
		*(*any)(fieldPtr) = value
	}
}

// Type conversion helpers (ZERO reflection)
func convertToInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func convertToInt8(value any) (int8, bool) {
	switch v := value.(type) {
	case int8:
		return v, true
	case int:
		return int8(v), true
	case int64:
		return int8(v), true
	case float64:
		return int8(v), true
	default:
		return 0, false
	}
}

func convertToInt16(value any) (int16, bool) {
	switch v := value.(type) {
	case int16:
		return v, true
	case int:
		return int16(v), true
	case int64:
		return int16(v), true
	case float64:
		return int16(v), true
	default:
		return 0, false
	}
}

func convertToInt32(value any) (int32, bool) {
	switch v := value.(type) {
	case int32:
		return v, true
	case int:
		return int32(v), true
	case int64:
		return int32(v), true
	case float64:
		return int32(v), true
	default:
		return 0, false
	}
}

func convertToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func convertToUint(value any) (uint, bool) {
	switch v := value.(type) {
	case uint:
		return v, true
	case uint64:
		return uint(v), true
	case int:
		return uint(v), true
	case int64:
		return uint(v), true
	case float64:
		return uint(v), true
	default:
		return 0, false
	}
}

func convertToUint8(value any) (uint8, bool) {
	switch v := value.(type) {
	case uint8:
		return v, true
	case uint:
		return uint8(v), true
	case uint64:
		return uint8(v), true
	case int:
		return uint8(v), true
	case int64:
		return uint8(v), true
	case float64:
		return uint8(v), true
	default:
		return 0, false
	}
}

func convertToUint16(value any) (uint16, bool) {
	switch v := value.(type) {
	case uint16:
		return v, true
	case uint:
		return uint16(v), true
	case uint64:
		return uint16(v), true
	case int:
		return uint16(v), true
	case int64:
		return uint16(v), true
	case float64:
		return uint16(v), true
	default:
		return 0, false
	}
}

func convertToUint32(value any) (uint32, bool) {
	switch v := value.(type) {
	case uint32:
		return v, true
	case uint:
		return uint32(v), true
	case uint64:
		return uint32(v), true
	case int:
		return uint32(v), true
	case int64:
		return uint32(v), true
	case float64:
		return uint32(v), true
	default:
		return 0, false
	}
}

func convertToUint64(value any) (uint64, bool) {
	switch v := value.(type) {
	case uint64:
		return v, true
	case uint:
		return uint64(v), true
	case int:
		return uint64(v), true
	case int64:
		return uint64(v), true
	case float64:
		return uint64(v), true
	default:
		return 0, false
	}
}

func convertToFloat32(value any) (float32, bool) {
	switch v := value.(type) {
	case float32:
		return v, true
	case float64:
		return float32(v), true
	case int:
		return float32(v), true
	case int64:
		return float32(v), true
	default:
		return 0, false
	}
}

func convertToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
