// Package codec provides encoding and decoding of Go structs to and from Neo4j
// property maps. It uses compiled opcodes for efficient struct traversal and
// supports Neo4j-specific types like dates, times, points, and durations.
package codec

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Compiler builds opcode sequences for encoding/decoding structs.
// REGISTRATION PHASE: All Compile methods use heavy reflection and should
// only be called during type registration at startup.
type Compiler struct {
	encCache map[TypePtr]*Opcode
	decCache map[TypePtr]Decoder
}

func NewCompiler() *Compiler {
	return &Compiler{
		encCache: make(map[TypePtr]*Opcode),
		decCache: make(map[TypePtr]Decoder),
	}
}

// --- Encoder Compilation ---

func (c *Compiler) Compile(typ reflect.Type) (*Opcode, error) {
	ptr := getTypePtr(typ)
	if op, ok := c.encCache[ptr]; ok {
		return op, nil
	}

	// Check for special types FIRST (before kind switch)
	// These are struct types that need primitive opcodes, not struct traversal
	if op := c.checkSpecialTypeEncoder(typ); op != nil {
		return op, nil
	}

	switch typ.Kind() {
	case reflect.Struct:
		return c.compileStruct(typ)
	case reflect.Pointer:
		return c.compilePtr(typ)
	case reflect.Slice:
		return c.compileSlice(typ)
	case reflect.Map:
		return c.compileMap(typ)
	case reflect.Interface:
		return &Opcode{Op: OpFieldInterface, Meta: FieldMeta{Type: typ, Size: typ.Size()}}, nil
	default:
		return c.compilePrimitive(typ)
	}
}

// checkSpecialTypeEncoder returns an opcode for special types that are structs
// but should NOT use struct traversal (e.g., time.Time, neo4j.Duration).
// Returns nil if the type is not special.
func (c *Compiler) checkSpecialTypeEncoder(typ reflect.Type) *Opcode {
	var op OpType
	switch typ {
	case reflect.TypeOf(time.Time{}):
		op = OpFieldTime
	case reflect.TypeOf(neo4j.Date{}):
		op = OpFieldDate
	case reflect.TypeOf(neo4j.LocalTime{}):
		op = OpFieldLocalTime
	case reflect.TypeOf(neo4j.LocalDateTime{}):
		op = OpFieldLocalDateTime
	case reflect.TypeOf(neo4j.Duration{}):
		op = OpFieldDuration
	case reflect.TypeOf(neo4j.Point2D{}):
		op = OpFieldPoint2D
	case reflect.TypeOf(neo4j.Point3D{}):
		op = OpFieldPoint3D
	default:
		return nil
	}
	return &Opcode{Op: op, Meta: FieldMeta{Type: typ, Size: typ.Size()}}
}

func (c *Compiler) compilePrimitive(typ reflect.Type) (*Opcode, error) {
	var op OpType

	// Check special types (redundant check for safety, main check is in checkSpecialTypeEncoder)
	switch typ {
	case reflect.TypeOf(time.Time{}):
		op = OpFieldTime
	case reflect.TypeOf(neo4j.Date{}):
		op = OpFieldDate
	case reflect.TypeOf(neo4j.LocalTime{}):
		op = OpFieldLocalTime
	case reflect.TypeOf(neo4j.LocalDateTime{}):
		op = OpFieldLocalDateTime
	case reflect.TypeOf(neo4j.Duration{}):
		op = OpFieldDuration
	case reflect.TypeOf(neo4j.Point2D{}):
		op = OpFieldPoint2D
	case reflect.TypeOf(neo4j.Point3D{}):
		op = OpFieldPoint3D
	default:
		switch typ.Kind() {
		case reflect.Bool:
			op = OpFieldBool
		case reflect.Int:
			op = OpFieldInt
		case reflect.Int8:
			op = OpFieldInt8
		case reflect.Int16:
			op = OpFieldInt16
		case reflect.Int32:
			op = OpFieldInt32
		case reflect.Int64:
			op = OpFieldInt64
		case reflect.Uint:
			op = OpFieldUint
		case reflect.Uint8:
			op = OpFieldUint8
		case reflect.Uint16:
			op = OpFieldUint16
		case reflect.Uint32:
			op = OpFieldUint32
		case reflect.Uint64:
			op = OpFieldUint64
		case reflect.Float32:
			op = OpFieldFloat32
		case reflect.Float64:
			op = OpFieldFloat64
		case reflect.String:
			op = OpFieldString
		default:
			return nil, fmt.Errorf("unsupported primitive type: %s", typ.Kind())
		}
	}

	return &Opcode{Op: op, Meta: FieldMeta{Type: typ, Size: typ.Size()}}, nil
}

func (c *Compiler) compileStruct(typ reflect.Type) (*Opcode, error) {
	head := &Opcode{Op: OpStructStart, Meta: FieldMeta{Type: typ}}
	c.encCache[getTypePtr(typ)] = head

	current := head

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		info := parseFieldInfo(field)

		if info.IsSkip {
			continue
		}

		if info.IsEmbed {
			subOp, err := c.Compile(field.Type)
			if err != nil {
				return nil, err
			}
			op := &Opcode{
				Op:         OpFieldEmbed,
				Meta:       info.Meta,
				SubOpcodes: subOp,
			}
			current.Next = op
			current = op
			continue
		}

		typeOp, err := c.Compile(field.Type)
		if err != nil {
			return nil, err
		}

		op := &Opcode{
			Op:         typeOp.Op,
			DBName:     info.DBName,
			Meta:       info.Meta,
			SubOpcodes: typeOp.SubOpcodes,
			IsSkip:     info.IsSkip,
		}

		if typeOp.Op == OpStructStart {
			op.Op = OpFieldStruct
			op.SubOpcodes = typeOp
		} else if typeOp.Op == OpFieldSlice {
			op.Meta.Size = typeOp.Meta.Size
		} else if isComplexOp(typeOp.Op) {
			op.SubOpcodes = typeOp.SubOpcodes
		}

		current.Next = op
		current = op
	}

	current.Next = &Opcode{Op: OpStructEnd}
	return head, nil
}

func (c *Compiler) compilePtr(typ reflect.Type) (*Opcode, error) {
	elemOp, err := c.Compile(typ.Elem())
	if err != nil {
		return nil, err
	}

	return &Opcode{
		Op:         OpFieldPtr,
		Meta:       FieldMeta{Type: typ, Size: typ.Size()},
		SubOpcodes: elemOp,
	}, nil
}

func (c *Compiler) compileSlice(typ reflect.Type) (*Opcode, error) {
	if typ.Elem().Kind() == reflect.Uint8 {
		return &Opcode{Op: OpFieldBytes, Meta: FieldMeta{Type: typ, Size: typ.Size()}}, nil
	}

	elemOp, err := c.Compile(typ.Elem())
	if err != nil {
		return nil, err
	}

	return &Opcode{
		Op:         OpFieldSlice,
		Meta:       FieldMeta{Type: typ, Size: typ.Elem().Size()},
		SubOpcodes: elemOp,
	}, nil
}

func (c *Compiler) compileMap(typ reflect.Type) (*Opcode, error) {
	if typ.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("map keys must be strings, got %s", typ.Key().Kind())
	}

	elemOp, err := c.Compile(typ.Elem())
	if err != nil {
		return nil, err
	}

	return &Opcode{
		Op:         OpFieldMap,
		Meta:       FieldMeta{Type: typ, Size: typ.Size()},
		SubOpcodes: elemOp,
	}, nil
}

func isComplexOp(op OpType) bool {
	return op == OpFieldSlice || op == OpFieldMap || op == OpFieldPtr || op == OpFieldStruct || op == OpFieldEmbed || op == OpStructStart
}

// --- Decoder Compilation ---

func (c *Compiler) CompileDecoder(typ reflect.Type) (Decoder, error) {
	ptr := getTypePtr(typ)
	if dec, ok := c.decCache[ptr]; ok {
		return dec, nil
	}

	// Check for special types FIRST (before kind switch)
	// These are struct types that need special decoders, not structDecoder
	if dec := c.checkSpecialTypeDecoder(typ); dec != nil {
		return dec, nil
	}

	switch typ.Kind() {
	case reflect.Struct:
		return c.compileStructDecoder(typ)
	case reflect.Ptr:
		return c.compilePtrDecoder(typ)
	case reflect.Slice:
		// Special case: []byte is treated as a primitive (bytes decoder)
		if typ.Elem().Kind() == reflect.Uint8 {
			return c.compilePrimitiveDecoder(typ)
		}
		return c.compileSliceDecoder(typ)
	case reflect.Map:
		return nil, fmt.Errorf("maps are not supported for Neo4j codec - Neo4j data model only supports scalar properties")
	case reflect.Interface:
		return DecoderFunc(interfaceDecoder), nil
	default:
		return c.compilePrimitiveDecoder(typ)
	}
}

// checkSpecialTypeDecoder returns a decoder for special types that are structs
// but should NOT use structDecoder (e.g., time.Time, neo4j.Duration).
// Returns nil if the type is not special.
func (c *Compiler) checkSpecialTypeDecoder(typ reflect.Type) Decoder {
	switch typ {
	case reflect.TypeOf(time.Time{}):
		return DecoderFunc(timeDecoder)
	case reflect.TypeOf(neo4j.Date{}):
		return DecoderFunc(dateDecoder)
	case reflect.TypeOf(neo4j.LocalTime{}):
		return DecoderFunc(localTimeDecoder)
	case reflect.TypeOf(neo4j.LocalDateTime{}):
		return DecoderFunc(localDateTimeDecoder)
	case reflect.TypeOf(neo4j.Time{}):
		return DecoderFunc(neo4jTimeDecoder)
	case reflect.TypeOf(neo4j.Duration{}):
		return DecoderFunc(durationDecoder)
	case reflect.TypeOf(neo4j.Point2D{}):
		return DecoderFunc(point2DDecoder)
	case reflect.TypeOf(neo4j.Point3D{}):
		return DecoderFunc(point3DDecoder)
	}
	return nil
}

func (c *Compiler) compilePrimitiveDecoder(typ reflect.Type) (Decoder, error) {
	// Check special types (redundant check for safety, main check is in checkSpecialTypeDecoder)
	if typ == reflect.TypeOf(time.Time{}) {
		return DecoderFunc(timeDecoder), nil
	}

	switch typ.Kind() {
	case reflect.Int:
		return DecoderFunc(intDecoder), nil
	case reflect.Int8:
		return DecoderFunc(int8Decoder), nil
	case reflect.Int16:
		return DecoderFunc(int16Decoder), nil
	case reflect.Int32:
		return DecoderFunc(int32Decoder), nil
	case reflect.Int64:
		return DecoderFunc(int64Decoder), nil
	case reflect.String:
		return DecoderFunc(stringDecoder), nil
	case reflect.Bool:
		return DecoderFunc(boolDecoder), nil
	case reflect.Float64:
		return DecoderFunc(float64Decoder), nil
	case reflect.Uint:
		return DecoderFunc(uintDecoder), nil
	case reflect.Uint8:
		return DecoderFunc(uint8Decoder), nil
	case reflect.Uint16:
		return DecoderFunc(uint16Decoder), nil
	case reflect.Uint32:
		return DecoderFunc(uint32Decoder), nil
	case reflect.Uint64:
		return DecoderFunc(uint64Decoder), nil
	case reflect.Float32:
		return DecoderFunc(float32Decoder), nil
	case reflect.Slice:
		// Special case for []byte
		if typ.Elem().Kind() == reflect.Uint8 {
			return DecoderFunc(bytesDecoder), nil
		}
	}
	// Fallback
	return DecoderFunc(func(data any, ptr unsafe.Pointer) error {
		return fmt.Errorf("unsupported primitive decoder type: %s", typ)
	}), nil
}

func (c *Compiler) compileStructDecoder(typ reflect.Type) (Decoder, error) {
	dec := &structDecoder{
		fields: make(map[string]*fieldDecoder),
	}
	c.decCache[getTypePtr(typ)] = dec // Register early for recursion

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		info := parseFieldInfo(field)

		if info.IsSkip {
			continue
		}

		// Handle embedded structs: flatten their fields into parent
		if info.IsEmbed {
			embeddedType := field.Type
			// If it's a pointer, get the element type
			if embeddedType.Kind() == reflect.Ptr {
				embeddedType = embeddedType.Elem()
			}

			// Recursively compile the embedded struct's decoder
			embeddedDec, err := c.CompileDecoder(embeddedType)
			if err != nil {
				return nil, err
			}

			// Extract the struct decoder's fields and merge them
			if structDec, ok := embeddedDec.(*structDecoder); ok {
				for dbName, fieldDec := range structDec.fields {
					// Adjust offset to account for embedding
					dec.fields[dbName] = &fieldDecoder{
						offset:  field.Offset + fieldDec.offset,
						decoder: fieldDec.decoder,
					}
				}
			}
			continue
		}

		fieldDec, err := c.CompileDecoder(field.Type)
		if err != nil {
			return nil, err
		}

		dec.fields[info.DBName] = &fieldDecoder{
			offset:  field.Offset,
			decoder: fieldDec,
		}
	}
	return dec, nil
}

func (c *Compiler) compilePtrDecoder(typ reflect.Type) (Decoder, error) {
	elemDec, err := c.CompileDecoder(typ.Elem())
	if err != nil {
		return nil, err
	}
	return &ptrDecoder{
		elemDecoder: elemDec,
		allocate:    makePtrAllocator(typ.Elem()), // Compile-time allocation strategy
	}, nil
}

func (c *Compiler) compileSliceDecoder(typ reflect.Type) (Decoder, error) {
	elemDec, err := c.CompileDecoder(typ.Elem())
	if err != nil {
		return nil, err
	}
	return &sliceDecoder{
		elemDecoder: elemDec,
		elemSize:    typ.Elem().Size(),
		allocate:    makeSliceAllocator(typ.Elem()), // Compile-time allocation strategy
	}, nil
}
