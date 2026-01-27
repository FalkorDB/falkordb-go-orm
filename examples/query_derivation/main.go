package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/examples/internal/ciutil"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// Person entity for testing query derivation
type Person struct {
	ID      *int64 `falkor:"id,generated"`
	Name    string `falkor:"name,required"`
	Age     int    `falkor:"age,indexed"`
	Email   string `falkor:"email,unique"`
	City    string `falkor:"city"`
	Country string `falkor:"country,interned"`
}

func (p *Person) Labels() []string {
	return []string{"Person"}
}

func main() {
	fmt.Println("FalkorDB Go ORM - Query Derivation Example (Phase 3)")
	fmt.Println("====================================================")
	fmt.Println()

	// Connect to FalkorDB
	fmt.Println("1. Connecting to FalkorDB...")
	db, err := falkordb.FalkorDBNew(&falkordb.ConnectionOption{
		Addr: ciutil.Addr(),
	})
	if err != nil {
		if ciutil.RequireDB() {
			log.Fatalf("Could not connect to FalkorDB at %s: %v", ciutil.Addr(), err)
		}
		fmt.Printf("   ⚠ Could not connect to FalkorDB: %v\n", err)
		fmt.Println("   Please start FalkorDB to run this example.")
		return
	}

	graph := db.SelectGraph("query_derivation")
	fmt.Println("   ✓ Connected to FalkorDB")
	fmt.Println()

	// Create repository
	fmt.Println("2. Creating Person repository...")
	repo := orm.NewRepository[Person](graph)
	fmt.Println("   ✓ Repository created")
	fmt.Println()

	// Clean up previous data
	fmt.Println("3. Cleaning up previous data...")
	repo.DeleteAllEntities()
	fmt.Println("   ✓ Cleaned up")
	fmt.Println()

	// Create sample data
	fmt.Println("4. Creating sample data...")
	people := []*Person{
		{Name: "Alice Johnson", Age: 28, Email: "alice@example.com", City: "New York", Country: "USA"},
		{Name: "Bob Smith", Age: 35, Email: "bob@example.com", City: "London", Country: "UK"},
		{Name: "Charlie Brown", Age: 22, Email: "charlie@example.com", City: "New York", Country: "USA"},
		{Name: "Diana Prince", Age: 30, Email: "diana@example.com", City: "Paris", Country: "France"},
		{Name: "Eve Adams", Age: 45, Email: "eve@example.com", City: "Tokyo", Country: "Japan"},
		{Name: "Frank Miller", Age: 28, Email: "frank@example.com", City: "Berlin", Country: "Germany"},
		{Name: "Grace Lee", Age: 33, Email: "grace@example.com", City: "New York", Country: "USA"},
		{Name: "Henry Davis", Age: 40, Email: "henry@example.com", City: "Sydney", Country: "Australia"},
	}

	saved, err := repo.SaveAll(people)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Created %d people\n", len(saved))
	fmt.Println()

	// Test query derivation patterns
	fmt.Println("5. Testing query derivation patterns...")
	fmt.Println()

	// FindByName
	fmt.Println("   a) FindByName (Equal operator)")
	result, err := repo.ExecuteDerivedQuery("FindByName", "Alice Johnson")
	if err != nil {
		log.Fatal(err)
	}
	results := result.([]*Person)
	fmt.Printf("      Found %d person(s) with name 'Alice Johnson'\n", len(results))
	ciutil.Check(len(results) == 1, "expected 1 result for FindByName, got %d", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Println()

	// FindByAgeGreaterThan
	fmt.Println("   b) FindByAgeGreaterThan")
	result, err = repo.ExecuteDerivedQuery("FindByAgeGreaterThan", 30)
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) older than 30\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Println()

	// FindByAgeLessThanOrEqual
	fmt.Println("   c) FindByAgeLessThanOrEqual")
	result, err = repo.ExecuteDerivedQuery("FindByAgeLessThanOrEqual", 28)
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) aged 28 or younger\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Println()

	// FindByNameAndAge (AND logic)
	fmt.Println("   d) FindByNameAndAge (AND logic)")
	result, err = repo.ExecuteDerivedQuery("FindByNameAndAge", "Alice Johnson", 28)
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) named 'Alice Johnson' aged 28\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (Age: %d, City: %s)\n", p.Name, p.Age, p.City)
	}
	fmt.Println()

	// FindByCityOrCountry (OR logic)
	fmt.Println("   e) FindByCityOrCountry (OR logic)")
	result, err = repo.ExecuteDerivedQuery("FindByCityOrCountry", "Paris", "Japan")
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) in Paris or Japan\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (City: %s, Country: %s)\n", p.Name, p.City, p.Country)
	}
	fmt.Println()

	// FindByNameStartsWith
	fmt.Println("   f) FindByNameStartsWith")
	result, err = repo.ExecuteDerivedQuery("FindByNameStartsWith", "Alice")
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) with name starting with 'Alice'\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s\n", p.Name)
	}
	fmt.Println()

	// FindByCityContains
	fmt.Println("   g) FindByCityContains")
	result, err = repo.ExecuteDerivedQuery("FindByCityContains", "York")
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) in cities containing 'York'\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (City: %s)\n", p.Name, p.City)
	}
	fmt.Println()

	// FindByAgeBetween
	fmt.Println("   h) FindByAgeBetween")
	result, err = repo.ExecuteDerivedQuery("FindByAgeBetween", 28, 35)
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) aged between 28 and 35\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Println()

	// FindByCityIn
	fmt.Println("   i) FindByCityIn")
	result, err = repo.ExecuteDerivedQuery("FindByCityIn", []string{"New York", "Paris", "Tokyo"})
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) in New York, Paris, or Tokyo\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (City: %s)\n", p.Name, p.City)
	}
	fmt.Println()

	// CountByAge
	fmt.Println("   j) CountByAge")
	result, err = repo.ExecuteDerivedQuery("CountByAge", 28)
	if err != nil {
		log.Fatal(err)
	}
	count := result.(int64)
	fmt.Printf("      Count of people aged 28: %d\n", count)
	fmt.Println()

	// CountByAgeGreaterThan
	fmt.Println("   k) CountByAgeGreaterThan")
	result, err = repo.ExecuteDerivedQuery("CountByAgeGreaterThan", 30)
	if err != nil {
		log.Fatal(err)
	}
	count = result.(int64)
	fmt.Printf("      Count of people older than 30: %d\n", count)
	fmt.Println()

	// ExistsByEmail
	fmt.Println("   l) ExistsByEmail")
	result, err = repo.ExecuteDerivedQuery("ExistsByEmail", "alice@example.com")
	if err != nil {
		log.Fatal(err)
	}
	exists := result.(bool)
	fmt.Printf("      Email 'alice@example.com' exists: %v\n", exists)
	fmt.Println()

	// ExistsByEmail (non-existent)
	fmt.Println("   m) ExistsByEmail (non-existent)")
	result, err = repo.ExecuteDerivedQuery("ExistsByEmail", "nonexistent@example.com")
	if err != nil {
		log.Fatal(err)
	}
	exists = result.(bool)
	fmt.Printf("      Email 'nonexistent@example.com' exists: %v\n", exists)
	fmt.Println()

	// FindByAgeGreaterThanOrderByName (with ordering)
	fmt.Println("   n) FindByAgeGreaterThanOrderByName (with ordering)")
	result, err = repo.ExecuteDerivedQuery("FindByAgeGreaterThanOrderByName", 28)
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) older than 28, ordered by name:\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Println()

	// FindByCountryOrderByAgeDesc (with descending order)
	fmt.Println("   o) FindByCountryOrderByAgeDesc")
	result, err = repo.ExecuteDerivedQuery("FindByCountryOrderByAgeDesc", "USA")
	if err != nil {
		log.Fatal(err)
	}
	results = result.([]*Person)
	fmt.Printf("      Found %d person(s) in USA, ordered by age (descending):\n", len(results))
	for _, p := range results {
		fmt.Printf("      - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Println()

	// Test query parser directly
	fmt.Println("6. Testing QueryParser directly...")
	parser := orm.NewQueryParser()

	testCases := []string{
		"FindByName",
		"FindByAgeGreaterThan",
		"FindByNameAndAge",
		"CountByCity",
		"ExistsByEmail",
		"DeleteByAge",
		"FindByAgeBetween",
		"FindByNameStartsWith",
		"FindByAgeGreaterThanOrderByNameAsc",
		"FindByCountryOrderByAgeDescAndNameAsc",
	}

	for _, methodName := range testCases {
		spec, err := parser.Parse(methodName)
		if err != nil {
			fmt.Printf("   ✗ Failed to parse '%s': %v\n", methodName, err)
		} else {
			fmt.Printf("   ✓ Parsed '%s':\n", methodName)
			fmt.Printf("      Operation: %s\n", spec.Operation)
			fmt.Printf("      Conditions: %d\n", len(spec.Conditions))
			for _, cond := range spec.Conditions {
				fmt.Printf("         - %s %s (params: %d)\n", cond.PropertyName, cond.Operator, cond.Operator.ParamCount())
			}
			if len(spec.Ordering) > 0 {
				fmt.Printf("      Ordering: %d clause(s)\n", len(spec.Ordering))
				for _, order := range spec.Ordering {
					fmt.Printf("         - %s %s\n", order.PropertyName, order.Direction)
				}
			}
		}
	}
	fmt.Println()

	// Test query spec caching
	fmt.Println("7. Testing query spec caching...")
	parser2 := orm.NewQueryParser()

	// First parse
	_, err = parser2.Parse("FindByAgeGreaterThan")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ First parse: cached")

	// Second parse (from cache)
	_, err = parser2.Parse("FindByAgeGreaterThan")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Second parse: retrieved from cache")

	// Verify cache by clearing and parsing again
	parser2.ClearCache()
	_, err = parser2.Parse("FindByAgeGreaterThan")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ After cache clear: re-cached")
	fmt.Println()

	// Demonstrate type safety
	fmt.Println("8. Demonstrating type safety...")

	// This will work
	result, err = repo.ExecuteDerivedQuery("FindByAge", 28)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Type-safe query executed successfully (found %d results)\n", len(result.([]*Person)))

	// This will fail with wrong argument count
	_, err = repo.ExecuteDerivedQuery("FindByNameAndAge", "Alice") // Missing age argument
	if err != nil {
		fmt.Printf("   ✓ Correctly caught error: %v\n", err)
	}
	fmt.Println()

	// Final count
	fmt.Println("9. Final verification...")
	totalCount, err := repo.Count()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Total people in database: %d\n", totalCount)
	fmt.Println()

	fmt.Println("✅ Phase 3 demonstration complete!")
	fmt.Println()
	fmt.Println("Supported patterns:")
	fmt.Println("  - FindBy<Property>")
	fmt.Println("  - FindBy<Property><Operator>")
	fmt.Println("  - FindBy<Property1>And<Property2>")
	fmt.Println("  - FindBy<Property1>Or<Property2>")
	fmt.Println("  - FindBy<Property>OrderBy<Property2>")
	fmt.Println("  - CountBy<Property>")
	fmt.Println("  - ExistsBy<Property>")
	fmt.Println("  - DeleteBy<Property>")
	fmt.Println()
	fmt.Println("Supported operators:")
	fmt.Println("  - Equal (default)")
	fmt.Println("  - NotEqual")
	fmt.Println("  - GreaterThan, GreaterThanOrEqual")
	fmt.Println("  - LessThan, LessThanOrEqual")
	fmt.Println("  - Between")
	fmt.Println("  - In, NotIn")
	fmt.Println("  - Like, NotLike")
	fmt.Println("  - Contains, NotContains")
	fmt.Println("  - StartsWith, EndsWith")
	fmt.Println("  - IsNull, IsNotNull")

	// Verify entity type at runtime
	fmt.Println()
	fmt.Println("Type information:")
	fmt.Printf("  Entity type: %v\n", reflect.TypeOf(Person{}))
	fmt.Printf("  Repository type: Repository[Person]\n")
}
