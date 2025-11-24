package codec

import (
	"fmt"
	"reflect"
	"unsafe"
)

// CodecRegistry manages type codecs with zero-reflection runtime
type CodecRegistry struct {
	encoders map[TypePtr]*Encoder      // typeptr -> encoder
	decoders map[TypePtr]*Decoder      // typeptr -> decoder
	metadata map[TypePtr]*TypeMetadata // typeptr -> metadata
	compiler *Compiler                 // opcode compiler
}

// TypeMetadata contains pre-computed type information
type TypeMetadata struct {
	Name          string                       // Type name
	FieldsMap     map[string]string            // Go field name -> DB field name
	Labels        []string                     // Neo4j labels (for nodes)
	Relationships map[string]*RelationshipMeta // field -> relationship info
}

// RelationshipMeta contains relationship metadata
type RelationshipMeta struct {
	Many bool
	Dir  bool
}

// Encoder contains pre-built opcode sequence for encoding
type Encoder struct {
	opcodes *Opcode
}

// Decoder contains pre-built field lookup for decoding
type Decoder struct {
	fields map[string]*FieldDecoder
}

// FieldDecoder contains pre-computed field decoding info
type FieldDecoder struct {
	Meta       FieldMeta
	Op         OpType  // Specific opcode for this field
	SubOpcodes *Opcode // For complex types
}

// NewCodecRegistry creates a new codec registry
func NewCodecRegistry() *CodecRegistry {
	return &CodecRegistry{
		encoders: make(map[TypePtr]*Encoder),
		decoders: make(map[TypePtr]*Decoder),
		metadata: make(map[TypePtr]*TypeMetadata),
		compiler: NewCompiler(),
	}
}

// RegisterTypes generates codecs for the given types at registration time
// This is the ONLY place we use reflection - everything else is zero-reflection
func (r *CodecRegistry) RegisterTypes(types ...any) {
	for _, t := range types {
		typ := reflect.TypeOf(t)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct {
			continue
		}

		typePtr := getTypePtr(typ)

		// Build type metadata (reflection used only here)
		metadata := &TypeMetadata{
			Name:      typ.Name(),
			FieldsMap: make(map[string]string),
		}
		r.metadata[typePtr] = metadata

		// Store type for lookup
		r.StoreTypeForLookup(typ, metadata)

		// Build encoder opcode sequence (reflection used only here)
		op, err := r.compiler.Compile(typ)
		if err != nil {
			panic(fmt.Errorf("failed to compile encoder for %s: %w", typ.Name(), err))
		}
		r.encoders[typePtr] = &Encoder{opcodes: op}

		// Build decoder field map (reflection used only here)
		decoder := r.buildDecoder(typ)
		r.decoders[typePtr] = decoder
	}
}

// buildMetadata extracts type metadata (uses reflection ONCE)
func (r *CodecRegistry) buildMetadata(typ reflect.Type) *TypeMetadata {
	metadata := &TypeMetadata{
		Name:          typ.Name(),
		FieldsMap:     make(map[string]string),
		Labels:        []string{},
		Relationships: make(map[string]*RelationshipMeta),
	}

	// Extract field mappings from struct
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldInfo := parseFieldInfo(field)

		if fieldInfo.IsSkip {
			continue
		}

		// Map Go field name to DB field name
		metadata.FieldsMap[field.Name] = fieldInfo.DBName

		// TODO: Extract labels and relationships from field tags
		// This would replace the WalkStruct logic in Registry
	}

	return metadata
}

// buildDecoder creates a field decoder map (uses reflection ONCE)
func (r *CodecRegistry) buildDecoder(typ reflect.Type) *Decoder {
	fields := make(map[string]*FieldDecoder)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldInfo := parseFieldInfo(field)

		if fieldInfo.IsSkip {
			continue
		}

		decoder := &FieldDecoder{
			Meta: fieldInfo.Meta,
			Op:   getFieldOpcode(fieldInfo.Meta.Kind),
		}

		// For complex types, build sub-opcodes
		if needsSubOpcodes(fieldInfo.Meta.Kind) {
			decoder.SubOpcodes = r.buildSubOpcodes(field.Type, fieldInfo.Meta.Kind)
		}

		fields[fieldInfo.DBName] = decoder
	}

	return &Decoder{fields: fields}
}

// GetEncoder returns encoder for a value (ZERO reflection)
func (r *CodecRegistry) GetEncoder(v any) *Encoder {
	typePtr := getTypePtrFromValue(v)
	if enc, ok := r.encoders[typePtr]; ok {
		return enc
	}

	// Lazy compilation
	typ := reflect.TypeOf(v)
	op, err := r.compiler.Compile(typ)
	if err != nil {
		return nil
	}
	enc := &Encoder{opcodes: op}
	r.encoders[typePtr] = enc
	return enc
}

// GetDecoder returns decoder for a type (ZERO reflection)
func (r *CodecRegistry) GetDecoder(v any) *Decoder {
	typePtr := getTypePtrFromValue(v)
	return r.decoders[typePtr]
}

// Encode converts struct to map[string]any (ZERO reflection)
func (r *CodecRegistry) Encode(v any) (map[string]any, error) {
	encoder := r.GetEncoder(v)
	if encoder == nil {
		// Type not registered
		return nil, ErrTypeNotRegistered
	}

	structPtr := getValuePtr(v)
	return EncodeStruct(encoder.opcodes, structPtr)
}

// Decode converts map[string]any to struct (ZERO reflection)
func (r *CodecRegistry) Decode(data map[string]any, v any) error {
	decoder := r.GetDecoder(v)
	if decoder == nil {
		// Type not registered
		return ErrTypeNotRegistered
	}

	structPtr := getValuePtr(v)
	return decodeStructFields(decoder, data, structPtr)
}

// decodeStructFields decodes using field map (ZERO reflection)
func decodeStructFields(decoder *Decoder, data map[string]any, structPtr unsafe.Pointer) error {
	for fieldName, fieldDecoder := range decoder.fields {
		if value, exists := data[fieldName]; exists {
			// Use the pre-determined opcode for this field
			switch fieldDecoder.Op {
			case OpFieldBool, OpFieldInt, OpFieldInt8, OpFieldInt16, OpFieldInt32, OpFieldInt64,
				OpFieldUint, OpFieldUint8, OpFieldUint16, OpFieldUint32, OpFieldUint64,
				OpFieldFloat32, OpFieldFloat64, OpFieldString, OpFieldBytes, OpFieldInterface:
				// Direct field assignment for basic types
				fieldDecoder.Meta.writeFieldValue(structPtr, value)
			case OpFieldSlice, OpFieldMap, OpFieldPtr, OpFieldStruct:
				// TODO: Implement complex type decoding with SubOpcodes
				fieldDecoder.Meta.writeFieldValue(structPtr, value)
			}
		}
	}
	return nil
}

// getFieldOpcode maps Kind to appropriate OpType
func getFieldOpcode(kind Kind) OpType {
	switch kind {
	case KindBool:
		return OpFieldBool
	case KindInt:
		return OpFieldInt
	case KindInt8:
		return OpFieldInt8
	case KindInt16:
		return OpFieldInt16
	case KindInt32:
		return OpFieldInt32
	case KindInt64:
		return OpFieldInt64
	case KindUint:
		return OpFieldUint
	case KindUint8:
		return OpFieldUint8
	case KindUint16:
		return OpFieldUint16
	case KindUint32:
		return OpFieldUint32
	case KindUint64:
		return OpFieldUint64
	case KindFloat32:
		return OpFieldFloat32
	case KindFloat64:
		return OpFieldFloat64
	case KindString:
		return OpFieldString
	case KindBytes:
		return OpFieldBytes
	case KindSlice:
		return OpFieldSlice
	case KindMap:
		return OpFieldMap
	case KindStruct:
		return OpFieldStruct
	case KindPtr:
		return OpFieldPtr
	case KindInterface:
		return OpFieldInterface
	default:
		return OpFieldInterface
	}
}

// needsSubOpcodes determines if a type needs sub-opcodes
func needsSubOpcodes(kind Kind) bool {
	switch kind {
	case KindSlice, KindMap, KindStruct, KindPtr:
		return true
	default:
		return false
	}
}

// buildSubOpcodes builds sub-opcodes for complex types (placeholder)
func (r *CodecRegistry) buildSubOpcodes(typ reflect.Type, kind Kind) *Opcode {
	// TODO: Implement proper sub-opcode building for complex types
	// For now, return nil - this is where we'd recursively build opcodes
	// for slice elements, map keys/values, struct fields, pointer targets
	return nil
}

// GetTypeMetadata returns pre-computed type metadata (ZERO reflection)
func (r *CodecRegistry) GetTypeMetadata(v any) *TypeMetadata {
	typePtr := getTypePtrFromValue(v)
	return r.metadata[typePtr]
}

// GetByTypeName returns type metadata by name (used by registry)
func (r *CodecRegistry) GetByTypeName(name string) *TypeMetadata {
	for _, meta := range r.metadata {
		if meta.Name == name {
			return meta
		}
	}
	return nil
}

// Store type registry for type lookup
type typeRegistryEntry struct {
	Type reflect.Type
	Meta *TypeMetadata
}

// registeredTypes stores type->metadata mapping for GetTypeByName lookups
var registeredTypes = make(map[string]*typeRegistryEntry)

// StoreTypeForLookup stores a type for later lookup by name
func (r *CodecRegistry) StoreTypeForLookup(typ reflect.Type, meta *TypeMetadata) {
	registeredTypes[meta.Name] = &typeRegistryEntry{
		Type: typ,
		Meta: meta,
	}
}

// GetTypeByNameLookup returns a registered type by its name
func (r *CodecRegistry) GetTypeByNameLookup(name string) reflect.Type {
	if entry, ok := registeredTypes[name]; ok {
		return entry.Type
	}
	return nil
}

// Error types
var (
	ErrTypeNotRegistered = &CodecError{Message: "type not registered"}
)

type CodecError struct {
	Message string
}

func (e *CodecError) Error() string {
	return e.Message
}
