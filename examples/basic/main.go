package main

import (
	"fmt"
	"log"
	"reflect"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// Person represents a person entity in the graph
type Person struct {
	ID      *int64 `falkor:"id,generated"`
	Name    string `falkor:"name,required"`
	Email   string `falkor:"email,unique"`
	Age     int    `falkor:"age,indexed"`
	Country string `falkor:"country,interned"` // Deduplicated string
}

// Labels defines the node labels for Person
func (*Person) Labels() []string {
	return []string{"Person", "Individual"}
}

// Company represents a company entity
type Company struct {
	ID      *int64 `falkor:"id,generated"`
	Name    string `falkor:"name,required,unique"`
	Founded int    `falkor:"founded,indexed"`
}

// Labels defines the node labels for Company
func (*Company) Labels() []string {
	return []string{"Company", "Organization"}
}

func main() {
	fmt.Println("FalkorDB Go ORM - Basic Example (Phase 1)")
	fmt.Println("==========================================")

	// Register entities
	fmt.Println("1. Registering entities...")
	if err := orm.Register(&Person{}); err != nil {
		log.Fatal("Failed to register Person:", err)
	}
	if err := orm.Register(&Company{}); err != nil {
		log.Fatal("Failed to register Company:", err)
	}
	fmt.Println("   ✓ Entities registered")

	// Create mapper
	mapper := orm.NewMapper()

	// Example 1: Create a Person
	fmt.Println("2. Creating a Person entity...")
	alice := &Person{
		Name:    "Alice Johnson",
		Email:   "alice@example.com",
		Age:     28,
		Country: "USA",
	}

	cypher, params, err := mapper.MapToCreate(alice)
	if err != nil {
		log.Fatal("Failed to map person:", err)
	}

	fmt.Println("   Generated Cypher:")
	fmt.Printf("   %s\n", cypher)
	fmt.Println("\n   Parameters:")
	for k, v := range params {
		fmt.Printf("   %s: %v\n", k, v)
	}
	fmt.Println()

	// Example 2: Create a Company
	fmt.Println("3. Creating a Company entity...")
	company := &Company{
		Name:    "TechCorp Inc.",
		Founded: 2010,
	}

	cypher2, params2, err := mapper.MapToCreate(company)
	if err != nil {
		log.Fatal("Failed to map company:", err)
	}

	fmt.Println("   Generated Cypher:")
	fmt.Printf("   %s\n", cypher2)
	fmt.Println("\n   Parameters:")
	for k, v := range params2 {
		fmt.Printf("   %s: %v\n", k, v)
	}
	fmt.Println()

	// Example 3: Connect to FalkorDB and execute (requires running FalkorDB)
	fmt.Println("4. Connecting to FalkorDB (optional - requires running instance)...")
	db, err := falkordb.FalkorDBNew(&falkordb.ConnectionOption{
		Addr: "localhost:6379",
	})
	if err != nil {
		fmt.Printf("   ⚠ Could not connect to FalkorDB: %v\n", err)
		fmt.Println("   This is expected if FalkorDB is not running.")
		fmt.Println("\n5. Phase 1 demonstration complete!")
		return
	}

	graph := db.SelectGraph("orm_example")
	fmt.Println("   ✓ Connected to FalkorDB")

	// Execute CREATE for Alice
	fmt.Println("5. Executing CREATE query for Alice...")
	result, err := graph.Query(cypher, params, nil)
	if err != nil {
		log.Fatal("Failed to create person:", err)
	}

	if result.Next() {
		record := result.Record()
		nodeID, _ := record.GetByIndex(1) // node_id from RETURN clause
		mapper.UpdateEntityID(alice, nodeID)
		fmt.Printf("   ✓ Created person with ID: %d\n", *alice.ID)
	}
	fmt.Println()

	// Execute CREATE for Company
	fmt.Println("6. Executing CREATE query for TechCorp...")
	result2, err := graph.Query(cypher2, params2, nil)
	if err != nil {
		log.Fatal("Failed to create company:", err)
	}

	if result2.Next() {
		record := result2.Record()
		nodeID, _ := record.GetByIndex(1)
		mapper.UpdateEntityID(company, nodeID)
		fmt.Printf("   ✓ Created company with ID: %d\n", *company.ID)
	}
	fmt.Println()

	// Example 4: Update using MERGE
	fmt.Println("7. Updating Alice using MERGE...")
	alice.Age = 29 // Birthday!

	mergeCypher, mergeParams, err := mapper.MapToMerge(alice)
	if err != nil {
		log.Fatal("Failed to generate MERGE:", err)
	}

	fmt.Println("   Generated MERGE Cypher:")
	fmt.Printf("   %s\n", mergeCypher)
	fmt.Println("\n   Parameters:")
	for k, v := range mergeParams {
		fmt.Printf("   %s: %v\n", k, v)
	}

	result3, err := graph.Query(mergeCypher, mergeParams, nil)
	if err != nil {
		log.Fatal("Failed to update person:", err)
	}
	fmt.Printf("   ✓ Updated person (affected %d nodes)\n", result3.NodesCreated()+result3.NodesDeleted())
	fmt.Println()

	// Example 5: Read back from graph
	fmt.Println("8. Reading back person from graph...")
	readCypher := fmt.Sprintf("MATCH (n:%s) WHERE id(n) = $id RETURN n", "Person")
	readParams := map[string]interface{}{"id": *alice.ID}

	readResult, err := graph.Query(readCypher, readParams, nil)
	if err != nil {
		log.Fatal("Failed to read person:", err)
	}

	if readResult.Next() {
		record := readResult.Record()
		readPerson, err := mapper.MapFromRecord(record, reflect.TypeOf(Person{}), "n")
		if err != nil {
			log.Fatal("Failed to map person from record:", err)
		}

		p := readPerson.(*Person)
		fmt.Printf("   ✓ Read person from graph:\n")
		fmt.Printf("     ID: %d\n", *p.ID)
		fmt.Printf("     Name: %s\n", p.Name)
		fmt.Printf("     Email: %s\n", p.Email)
		fmt.Printf("     Age: %d\n", p.Age)
		fmt.Printf("     Country: %s\n", p.Country)
	}
	fmt.Println()

	fmt.Println("✅ Phase 1 demonstration complete!")
	fmt.Println("\nNext: Repository pattern with CRUD operations (Phase 2)")
}
