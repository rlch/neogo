package codec

import (
	"fmt"
	"reflect"
	"unsafe"
)

// CodecRegistry manages type codecs with zero-reflection runtime.
// It is the SINGLE SOURCE OF TRUTH for all type metadata, schema, encoders, and decoders.
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

	// Neo4j-specific metadata (single source of truth)
	nodeMeta map[string]*Neo4jNodeMetadata       // type name -> node metadata
	relMeta  map[string]*RelationshipStructMeta  // type name -> relationship metadata
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
		nodeMeta: make(map[string]*Neo4jNodeMetadata),
		relMeta:  make(map[string]*RelationshipStructMeta),
	}
}

// RegisterTypes generates codecs for the given types at registration time.
// REGISTRATION PHASE: Uses heavy reflection, should only be called at startup.
// This does ALL extraction in one pass - codec compilation AND Neo4j metadata.
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
		typeName := typ.Name()

		// Build type metadata
		metadata := r.buildMetadata(typ)
		r.metadata[typePtr] = metadata

		// Store type for lookup
		r.StoreTypeForLookup(typ, metadata)

		// Build encoder opcode sequence
		op, err := r.compiler.Compile(typ)
		if err != nil {
			panic(fmt.Errorf("failed to compile encoder for %s: %w", typeName, err))
		}
		r.encoders[typePtr] = &Encoder{opcodes: op}

		// Build decoder
		dec, err := r.compiler.CompileDecoder(typ)
		if err != nil {
			panic(fmt.Errorf("failed to compile decoder for %s: %w", typeName, err))
		}
		r.decoders[typePtr] = dec

		// Extract Neo4j-specific metadata based on type
		// Check if type implements INode
		if r.implementsINode(typ) || r.implementsINode(reflect.PtrTo(typ)) {
			nodeMeta, err := r.extractNeo4jNodeMetaFromType(typ)
			if err != nil {
				panic(fmt.Errorf("failed to extract Neo4j node metadata for %s: %w", typeName, err))
			}
			r.nodeMeta[typeName] = nodeMeta
		}

		// Check if type implements IRelationship
		if r.implementsIRelationship(typ) || r.implementsIRelationship(reflect.PtrTo(typ)) {
			relMeta, err := r.extractRelationshipMetaFromType(typ)
			if err != nil {
				panic(fmt.Errorf("failed to extract relationship metadata for %s: %w", typeName, err))
			}
			r.relMeta[typeName] = relMeta
		}
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

// DecodeMultiple decodes n values into a slice target.
// The slicePtr must be a pointer to a slice (e.g., *[]Person).
// This is used for batch decoding multiple records into a slice binding.
// HOT PATH: Uses pre-compiled decoders and allocators.
func (r *CodecRegistry) DecodeMultiple(values []any, slicePtr any) error {
	n := len(values)
	if n == 0 {
		return nil
	}

	decoder := r.GetDecoder(slicePtr)
	if decoder == nil {
		return ErrTypeNotRegistered
	}

	// Get the slice decoder to access its allocator and element decoder
	sliceDec, ok := decoder.(*sliceDecoder)
	if !ok {
		return fmt.Errorf("DecodeMultiple requires slice type, got decoder %T", decoder)
	}

	// Allocate slice using pre-compiled allocator (ZERO reflection)
	dataPtr, _, capacity := sliceDec.allocate(n)

	// Get the slice header from the target
	targetPtr := getValuePtr(slicePtr)
	header := (*sliceHeader)(targetPtr)
	header.Data = dataPtr
	header.Len = n
	header.Cap = capacity

	// Decode each value into the slice elements
	for i := 0; i < n; i++ {
		if values[i] == nil {
			continue // Leave as zero value
		}
		elemPtr := unsafe.Pointer(uintptr(dataPtr) + uintptr(i)*sliceDec.elemSize)
		if err := sliceDec.elemDecoder.Decode(values[i], elemPtr); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
	}

	return nil
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

// GetNodeMeta returns cached Neo4j node metadata by type name.
// Returns nil if the type was not registered or is not a node.
func (r *CodecRegistry) GetNodeMeta(name string) *Neo4jNodeMetadata {
	return r.nodeMeta[name]
}

// StoreNodeMeta stores Neo4j node metadata by type name.
// This is used for lazy registration when types are registered individually.
func (r *CodecRegistry) StoreNodeMeta(name string, meta *Neo4jNodeMetadata) {
	r.nodeMeta[name] = meta
}

// GetRelMeta returns cached relationship metadata by type name.
// Returns nil if the type was not registered or is not a relationship.
func (r *CodecRegistry) GetRelMeta(name string) *RelationshipStructMeta {
	return r.relMeta[name]
}

// StoreRelMeta stores relationship metadata by type name.
// This is used for lazy registration when types are registered individually.
func (r *CodecRegistry) StoreRelMeta(name string, meta *RelationshipStructMeta) {
	r.relMeta[name] = meta
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