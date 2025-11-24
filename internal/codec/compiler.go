package codec

import (
	"fmt"
	"reflect"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Compiler struct {
	cache map[reflect.Type]*Opcode
}

func NewCompiler() *Compiler {
	return &Compiler{
		cache: make(map[reflect.Type]*Opcode),
	}
}

func (c *Compiler) Compile(typ reflect.Type) (*Opcode, error) {
	if op, ok := c.cache[typ]; ok {
		return op, nil
	}

	// Placeholder to prevent infinite recursion
	// We might need a better strategy for recursive types (e.g. creating the opcode first)
	// For now, we'll handle recursion via pointers specifically.
	
	switch typ.Kind() {
	case reflect.Struct:
		return c.compileStruct(typ)
	case reflect.Ptr:
		return c.compilePtr(typ)
	case reflect.Slice:
		return c.compileSlice(typ)
	case reflect.Map:
		return c.compileMap(typ)
	case reflect.Interface:
		return &Opcode{Op: OpFieldInterface, Meta: FieldMeta{Type: typ}}, nil
	default:
		return c.compilePrimitive(typ)
	}
}

func (c *Compiler) compilePrimitive(typ reflect.Type) (*Opcode, error) {
	var op OpType
	
	// Check for special types first
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
	c.cache[typ] = head // Register to break recursion if needed? 
	// Actually recursion usually happens via pointers/slices/maps fields, not direct struct embedding (which is infinite size).
	
	current := head
	
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		info := parseFieldInfo(field)
		
		if info.IsSkip {
			continue
		}
		
		if info.IsEmbed {
			// Embedded struct: compile it and link it
			// But for opcodes, we might want to inline the fields or use OpFieldEmbed
			// Let's use OpFieldEmbed for simplicity
			subOp, err := c.Compile(field.Type)
			if err != nil {
				return nil, err
			}
			op := &Opcode{
				Op:         OpFieldEmbed,
				Meta:       info.Meta, // Offset is correct relative to outer struct
				SubOpcodes: subOp,
			}
			current.Next = op
			current = op
			continue
		}
		
		// Compile field type
		typeOp, err := c.Compile(field.Type)
		if err != nil {
			return nil, err
		}
		
		// Create field opcode
		op := &Opcode{
			Op:         typeOp.Op, // Use the opcode type we found (e.g. OpFieldInt)
			DBName:     info.DBName,
			Meta:       info.Meta,
			SubOpcodes: typeOp.SubOpcodes, // Pass through sub-opcodes if any
			IsSkip:     info.IsSkip,
			// For complex types, typeOp might describe the structure.
			// But typeOp is usually the "value" encoder.
			// For OpFieldInt, typeOp.Op is OpFieldInt.
			// For OpFieldSlice, typeOp.Op is OpFieldSlice and typeOp.SubOpcodes describes the element.
		}
		
		// Copy over any complex type info if typeOp was a complex type
		if isComplexOp(typeOp.Op) {
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
		Meta:       FieldMeta{Type: typ, Size: typ.Elem().Size()}, // Size is element size for iteration
		SubOpcodes: elemOp,
	}, nil
}

func (c *Compiler) compileMap(typ reflect.Type) (*Opcode, error) {
	// Neo4j maps must have string keys
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
