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

// getTypePtrFromValue extracts the type pointer directly from a value by computing
// the reflect.Type and using getTypePtr for consistency.
// This ensures that RegisterTypes(T{}) and GetTypeMetadata(&T{}) use the same key.
//
// BUG FIX: Previously this used iface.typ directly, which is the pointer type when
// v is *T (e.g., &Company{}). Now we use reflect.TypeOf to get the actual type,
// then normalize via getTypePtr to match RegisterTypes behavior.
func getTypePtrFromValue(v any) TypePtr {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return getTypePtr(typ)
}

// getValuePtr extracts the value pointer from interface{}
func getValuePtr(v any) unsafe.Pointer {
	iface := (*emptyInterface)(unsafe.Pointer(&v))
	return iface.ptr
}

// getInterfaceValue extracts the interface{} value directly from memory
// without using reflect.NewAt (zero reflection).
// This is used for OpFieldInterface encoding to avoid reflect overhead.
func getInterfaceValue(ptr unsafe.Pointer) any {
	return *(*any)(ptr)
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
		// Note: Special case for []byte is handled during compilation (compileSlice in compiler.go)
		// For Kind extraction, we just return KindSlice - the element type check happens elsewhere
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

// Exported versions for testing
func GetTypePtr(t reflect.Type) TypePtr {
	return getTypePtr(t)
}

func GetTypePtrFromValue(v any) TypePtr {
	return getTypePtrFromValue(v)
}

func GetValuePtr(v any) unsafe.Pointer {
	return getValuePtr(v)
}

func GetInterfaceValue(ptr unsafe.Pointer) any {
	return getInterfaceValue(ptr)
}

func GetKind(rk reflect.Kind) Kind {
	return getKind(rk)
}

func ConvertToInt(value any) (int, bool) {
	return convertToInt(value)
}

func ConvertToInt8(value any) (int8, bool) {
	return convertToInt8(value)
}

func ConvertToInt16(value any) (int16, bool) {
	return convertToInt16(value)
}

func ConvertToInt32(value any) (int32, bool) {
	return convertToInt32(value)
}

func ConvertToInt64(value any) (int64, bool) {
	return convertToInt64(value)
}

func ConvertToUint(value any) (uint, bool) {
	return convertToUint(value)
}

func ConvertToUint8(value any) (uint8, bool) {
	return convertToUint8(value)
}

func ConvertToUint16(value any) (uint16, bool) {
	return convertToUint16(value)
}

func ConvertToUint32(value any) (uint32, bool) {
	return convertToUint32(value)
}

func ConvertToUint64(value any) (uint64, bool) {
	return convertToUint64(value)
}

func ConvertToFloat32(value any) (float32, bool) {
	return convertToFloat32(value)
}

func ConvertToFloat64(value any) (float64, bool) {
	return convertToFloat64(value)
}

func ExtractFieldMeta(field reflect.StructField) FieldMeta {
	return extractFieldMeta(field)
}



// TypeInfo contains pre-computed type information extracted at registration time
type TypeInfo struct {
	Name           string                    // Type name
	Fields         map[string]*FieldMeta     // field name -> metadata
	FieldsToProps  map[string]string         // Go field name -> Neo4j property name
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