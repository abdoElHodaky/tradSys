package core

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
)

// Serializer provides serialization functionality for events and snapshots
type Serializer interface {
	// SerializeEvent serializes an event
	SerializeEvent(event *eventsourcing.Event) ([]byte, error)

	// DeserializeEvent deserializes an event
	DeserializeEvent(data []byte) (*eventsourcing.Event, error)

	// SerializeSnapshot serializes a snapshot
	SerializeSnapshot(snapshot interface{}) ([]byte, error)

	// DeserializeSnapshot deserializes a snapshot
	DeserializeSnapshot(data []byte, snapshotType reflect.Type) (interface{}, error)
}

// JSONSerializer provides JSON serialization for events and snapshots
type JSONSerializer struct {
	// Snapshot type registry
	snapshotTypes map[string]reflect.Type
}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{
		snapshotTypes: make(map[string]reflect.Type),
	}
}

// RegisterSnapshotType registers a snapshot type with the serializer
func (s *JSONSerializer) RegisterSnapshotType(aggregateType string, snapshot interface{}) {
	s.snapshotTypes[aggregateType] = reflect.TypeOf(snapshot).Elem()
}

// SerializeEvent serializes an event
func (s *JSONSerializer) SerializeEvent(event *eventsourcing.Event) ([]byte, error) {
	return json.Marshal(event)
}

// DeserializeEvent deserializes an event
func (s *JSONSerializer) DeserializeEvent(data []byte) (*eventsourcing.Event, error) {
	var event eventsourcing.Event
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// SerializeSnapshot serializes a snapshot
func (s *JSONSerializer) SerializeSnapshot(snapshot interface{}) ([]byte, error) {
	return json.Marshal(snapshot)
}

// DeserializeSnapshot deserializes a snapshot
func (s *JSONSerializer) DeserializeSnapshot(data []byte, snapshotType reflect.Type) (interface{}, error) {
	// Create a new instance of the snapshot type
	snapshotValue := reflect.New(snapshotType)
	snapshot := snapshotValue.Interface()

	// Unmarshal the data into the snapshot
	err := json.Unmarshal(data, snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// BinarySerializer provides binary serialization for events and snapshots
type BinarySerializer struct {
	// Snapshot type registry
	snapshotTypes map[string]reflect.Type
}

// NewBinarySerializer creates a new binary serializer
func NewBinarySerializer() *BinarySerializer {
	return &BinarySerializer{
		snapshotTypes: make(map[string]reflect.Type),
	}
}

// RegisterSnapshotType registers a snapshot type with the serializer
func (s *BinarySerializer) RegisterSnapshotType(aggregateType string, snapshot interface{}) {
	s.snapshotTypes[aggregateType] = reflect.TypeOf(snapshot).Elem()
}

// SerializeEvent serializes an event
func (s *BinarySerializer) SerializeEvent(event *eventsourcing.Event) ([]byte, error) {
	// For now, use JSON serialization
	// In a real implementation, this would use a binary protocol like Protocol Buffers or FlatBuffers
	return json.Marshal(event)
}

// DeserializeEvent deserializes an event
func (s *BinarySerializer) DeserializeEvent(data []byte) (*eventsourcing.Event, error) {
	// For now, use JSON deserialization
	var event eventsourcing.Event
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// SerializeSnapshot serializes a snapshot
func (s *BinarySerializer) SerializeSnapshot(snapshot interface{}) ([]byte, error) {
	// For now, use JSON serialization
	return json.Marshal(snapshot)
}

// DeserializeSnapshot deserializes a snapshot
func (s *BinarySerializer) DeserializeSnapshot(data []byte, snapshotType reflect.Type) (interface{}, error) {
	// For now, use JSON deserialization
	snapshotValue := reflect.New(snapshotType)
	snapshot := snapshotValue.Interface()

	err := json.Unmarshal(data, snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// Common errors
var (
	ErrSerializationFailed   = errors.New("serialization failed")
	ErrDeserializationFailed = errors.New("deserialization failed")
	ErrUnknownSnapshotType   = errors.New("unknown snapshot type")
)
