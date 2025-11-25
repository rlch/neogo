package codec

import (
	"fmt"
	"reflect"
)

// CodecRegistry manages type codecs with zero-reflection runtime.
//
// # Reflection Boundaries
//
// This codec system distinguishes between two phases:
//
// REGISTRATION PHASE (heavy reflection OK - happens once at startup):
//   - RegisterTypes, buildMetadata
//   - ExtractNeo4jNodeMeta, ExtractRelationshipMeta
//   - Compile, CompileDecoder
//
// HOT PATH (zero reflection - called per encode/decode operation):
//   - Encode, EncodeValue, Decode
//   - EncodeStruct, EncodeAny, all decoder.Decode methods
//
// LOOKUP (minimal reflection - cached, called once per type per operation):
//   - GetEncoder, GetDecoder use getTypePtrFromValue which calls reflect.TypeOf
//   - This is acceptable as lookup is O(1) after first call and results are cached
type CodecRegistry struct {
	encoders map[TypePtr]*Encoder      // typeptr -> encoder
	decoders map[TypePtr]Decoder       // typeptr -> decoder interface
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

// NewCodecRegistry creates a new codec registry
func NewCodecRegistry() *CodecRegistry {
	return &CodecRegistry{
		encoders: make(map[TypePtr]*Encoder),
		decoders: make(map[TypePtr]Decoder),
		metadata: make(map[TypePtr]*TypeMetadata),
		compiler: NewCompiler(),
	}
}

// RegisterTypes generates codecs for the given types at registration time.
// REGISTRATION PHASE: Uses heavy reflection, should only be called at startup.
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

		// Build type metadata
		metadata := r.buildMetadata(typ)
		r.metadata[typePtr] = metadata

		// Store type for lookup
		r.StoreTypeForLookup(typ, metadata)

		// Build encoder opcode sequence
		op, err := r.compiler.Compile(typ)
		if err != nil {
			panic(fmt.Errorf("failed to compile encoder for %s: %w", typ.Name(), err))
		}
		r.encoders[typePtr] = &Encoder{opcodes: op}

		// Build decoder
		dec, err := r.compiler.CompileDecoder(typ)
		if err != nil {
			panic(fmt.Errorf("failed to compile decoder for %s: %w", typ.Name(), err))
		}
		r.decoders[typePtr] = dec
	}
}

// buildMetadata extracts type metadata.
// REGISTRATION PHASE: Uses reflection, called only during RegisterTypes.
func (r *CodecRegistry) buildMetadata(typ reflect.Type) *TypeMetadata {
	metadata := &TypeMetadata{
		Name:          typ.Name(),
		FieldsMap:     make(map[string]string),
		Labels:        []string{},
		Relationships: make(map[string]*RelationshipMeta),
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldInfo := parseFieldInfo(field)

		if fieldInfo.IsSkip {
			continue
		}

		metadata.FieldsMap[field.Name] = fieldInfo.DBName
	}

	return metadata
}

// GetEncoder returns encoder for a value
// Note: This is a fast path that uses cached encoders. The cache key lookup
// uses getTypePtrFromValue which calls reflect.TypeOf(). This is acceptable
// because encoder lookup is only called once per Encode() operation, and the
// same type will typically be encoded multiple times (benefiting from caching).
func (r *CodecRegistry) GetEncoder(v any) *Encoder {
	// First try the fast path with a reflection-free type pointer
	// (getTypePtrFromValue uses reflect but is cached by encoder lookup)
	typePtr := getTypePtrFromValue(v)
	if enc, ok := r.encoders[typePtr]; ok {
		return enc
	}

	// Lazy compilation (Slow path) - we need reflection to compile anyway
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	op, err := r.compiler.Compile(typ)
	if err != nil {
		return nil
	}
	enc := &Encoder{opcodes: op}
	r.encoders[typePtr] = enc
	return enc
}

// GetDecoder returns decoder for a type
// Note: Like GetEncoder, this uses reflection for lookup which is acceptable
// as it's only called once per Decode() operation and benefits from caching.
func (r *CodecRegistry) GetDecoder(v any) Decoder {
	typePtr := getTypePtrFromValue(v)
	if dec, ok := r.decoders[typePtr]; ok {
		return dec
	}

	// Lazy compilation (Slow path) - we need reflection to compile anyway
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	dec, err := r.compiler.CompileDecoder(typ)
	if err != nil {
		return nil
	}
	r.decoders[typePtr] = dec
	return dec
}

// Encode converts struct to map[string]any.
// HOT PATH: Zero reflection after initial lookup.
func (r *CodecRegistry) Encode(v any) (map[string]any, error) {
	encoder := r.GetEncoder(v)
	if encoder == nil {
		return nil, ErrTypeNotRegistered
	}

	structPtr := getValuePtr(v)
	return EncodeStruct(encoder.opcodes, structPtr)
}

// EncodeValue encodes any value using opcodes.
// HOT PATH: Zero reflection after initial lookup.
func (r *CodecRegistry) EncodeValue(v any) (any, error) {
	encoder := r.GetEncoder(v)
	if encoder == nil {
		return nil, ErrTypeNotRegistered
	}

	ptr := getValuePtr(v)
	return EncodeAny(encoder.opcodes, ptr)
}

// Decode converts map/Node to struct.
// HOT PATH: Zero reflection after initial lookup.
func (r *CodecRegistry) Decode(data any, v any) error {
	decoder := r.GetDecoder(v)
	if decoder == nil {
		return ErrTypeNotRegistered
	}

	structPtr := getValuePtr(v)
	return decoder.Decode(data, structPtr)
}

// GetTypeMetadata returns pre-computed type metadata.
// LOOKUP: Uses getTypePtrFromValue (reflect.TypeOf) for cache key.
func (r *CodecRegistry) GetTypeMetadata(v any) *TypeMetadata {
	typePtr := getTypePtrFromValue(v)
	return r.metadata[typePtr]
}

// GetByTypeName returns type metadata by name
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

var registeredTypes = make(map[string]*typeRegistryEntry)

func (r *CodecRegistry) StoreTypeForLookup(typ reflect.Type, meta *TypeMetadata) {
	registeredTypes[meta.Name] = &typeRegistryEntry{
		Type: typ,
		Meta: meta,
	}
}

func (r *CodecRegistry) GetTypeByNameLookup(name string) reflect.Type {
	if entry, ok := registeredTypes[name]; ok {
		return entry.Type
	}
	return nil
}

var (
	ErrTypeNotRegistered = &CodecError{Message: "type not registered"}
)

type CodecError struct {
	Message string
}

func (e *CodecError) Error() string {
	return e.Message
}