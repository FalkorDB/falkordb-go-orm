package orm

import (
	"reflect"
	"testing"
)

// Test entity
type TestPerson struct {
	ID      *int64 `falkor:"id,generated"`
	Name    string `falkor:"name,required"`
	Email   string `falkor:"email,unique"`
	Age     int    `falkor:"age,indexed"`
	Country string `falkor:"country,interned"`
}

func (*TestPerson) Labels() []string {
	return []string{"Person", "Individual"}
}

func TestEntityRegistration(t *testing.T) {
	registry := NewEntityRegistry()

	err := registry.Register(&TestPerson{})
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	metadata := registry.Get(reflect.TypeOf(TestPerson{}))
	if metadata == nil {
		t.Fatal("Expected metadata, got nil")
	}

	// Verify labels
	if len(metadata.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(metadata.Labels))
	}
	if metadata.PrimaryLabel != "Person" {
		t.Errorf("Expected primary label 'Person', got '%s'", metadata.PrimaryLabel)
	}

	// Verify properties
	if len(metadata.Properties) != 5 {
		t.Errorf("Expected 5 properties, got %d", len(metadata.Properties))
	}

	// Verify ID property
	if metadata.IDProperty == nil {
		t.Fatal("Expected ID property, got nil")
	}
	if metadata.IDProperty.FieldName != "ID" {
		t.Errorf("Expected ID field name 'ID', got '%s'", metadata.IDProperty.FieldName)
	}
	if !metadata.IDProperty.IsID {
		t.Error("Expected IsID to be true")
	}

	// Verify property options
	nameProp := metadata.GetPropertyByName("Name")
	if nameProp == nil {
		t.Fatal("Name property not found")
	}
	if !nameProp.Required {
		t.Error("Expected Name to be required")
	}

	emailProp := metadata.GetPropertyByName("Email")
	if emailProp == nil {
		t.Fatal("Email property not found")
	}
	if !emailProp.Unique {
		t.Error("Expected Email to be unique")
	}

	ageProp := metadata.GetPropertyByName("Age")
	if ageProp == nil {
		t.Fatal("Age property not found")
	}
	if !ageProp.Indexed {
		t.Error("Expected Age to be indexed")
	}

	countryProp := metadata.GetPropertyByName("Country")
	if countryProp == nil {
		t.Fatal("Country property not found")
	}
	if !countryProp.Interned {
		t.Error("Expected Country to be interned")
	}
}

func TestEntityRegistrationDefaultLabel(t *testing.T) {
	type SimpleEntity struct {
		ID   int    `falkor:"id,generated"`
		Name string `falkor:"name"`
	}

	registry := NewEntityRegistry()
	err := registry.Register(&SimpleEntity{})
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	metadata := registry.Get(reflect.TypeOf(SimpleEntity{}))
	if metadata == nil {
		t.Fatal("Expected metadata, got nil")
	}

	// Should use struct name as default label
	if len(metadata.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(metadata.Labels))
	}
	if metadata.Labels[0] != "SimpleEntity" {
		t.Errorf("Expected label 'SimpleEntity', got '%s'", metadata.Labels[0])
	}
}

func TestPropertyMetadataParsing(t *testing.T) {
	type TestEntity struct {
		Field1 string `falkor:"custom_name"`
		Field2 string `falkor:"field2,required"`
		Field3 string `falkor:"field3,unique,indexed"`
		Field4 string `falkor:"field4,indexed,index_type=FULLTEXT"`
	}

	registry := NewEntityRegistry()
	err := registry.Register(&TestEntity{})
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	metadata := registry.Get(reflect.TypeOf(TestEntity{}))

	// Check custom name
	field1 := metadata.GetPropertyByName("Field1")
	if field1.GraphName != "custom_name" {
		t.Errorf("Expected graph name 'custom_name', got '%s'", field1.GraphName)
	}

	// Check required
	field2 := metadata.GetPropertyByName("Field2")
	if !field2.Required {
		t.Error("Expected Field2 to be required")
	}

	// Check unique and indexed
	field3 := metadata.GetPropertyByName("Field3")
	if !field3.Unique {
		t.Error("Expected Field3 to be unique")
	}
	if !field3.Indexed {
		t.Error("Expected Field3 to be indexed")
	}

	// Check index type
	field4 := metadata.GetPropertyByName("Field4")
	if field4.IndexType != "FULLTEXT" {
		t.Errorf("Expected index type 'FULLTEXT', got '%s'", field4.IndexType)
	}
}

func TestRelationshipMetadata(t *testing.T) {
	type Friend struct {
		ID   int    `falkor:"id,generated"`
		Name string `falkor:"name"`
	}

	type PersonWithRelations struct {
		ID      int       `falkor:"id,generated"`
		Name    string    `falkor:"name"`
		Friends []*Friend `falkor:"friends,rel=KNOWS,dir=out,lazy"`
	}

	registry := NewEntityRegistry()
	err := registry.Register(&PersonWithRelations{})
	if err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	metadata := registry.Get(reflect.TypeOf(PersonWithRelations{}))

	if len(metadata.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(metadata.Relationships))
	}

	rel := metadata.Relationships[0]
	if rel.FieldName != "Friends" {
		t.Errorf("Expected field name 'Friends', got '%s'", rel.FieldName)
	}
	if rel.RelationshipType != "KNOWS" {
		t.Errorf("Expected relationship type 'KNOWS', got '%s'", rel.RelationshipType)
	}
	if rel.Direction != DirectionOutgoing {
		t.Errorf("Expected direction OUTGOING, got %v", rel.Direction)
	}
	if !rel.Lazy {
		t.Error("Expected lazy loading to be true")
	}
	if !rel.IsList {
		t.Error("Expected IsList to be true for slice type")
	}
}
