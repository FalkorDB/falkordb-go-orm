package main

import (
	"fmt"
	"log"
	"reflect"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/examples/internal/ciutil"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// Person entity with relationships
type Person struct {
	ID      *int64 `falkor:"id,generated"`
	Name    string `falkor:"name,required"`
	Age     int    `falkor:"age"`
	Email   string `falkor:"email,unique"`
	City    string `falkor:"city"`
	Country string `falkor:"country,interned"`
}

func (*Person) Labels() []string {
	return []string{"Person"}
}

// Company entity
type Company struct {
	ID       *int64 `falkor:"id,generated"`
	Name     string `falkor:"name,required"`
	Industry string `falkor:"industry"`
	Founded  int    `falkor:"founded"`
}

func (*Company) Labels() []string {
	return []string{"Company"}
}

func main() {
	fmt.Println("FalkorDB Go ORM - Relationships Example (Phase 4)")
	fmt.Println("=================================================")
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

	graph := db.SelectGraph("relationships_example")
	fmt.Println("   ✓ Connected to FalkorDB")
	fmt.Println()

	// Create repositories
	fmt.Println("2. Creating repositories...")
	personRepo := orm.NewRepository[Person](graph)
	companyRepo := orm.NewRepository[Company](graph)
	fmt.Println("   ✓ Repositories created")
	fmt.Println()

	// Clean up previous data
	fmt.Println("3. Cleaning up previous data...")
	personRepo.DeleteAllEntities()
	companyRepo.DeleteAllEntities()
	fmt.Println("   ✓ Cleaned up")
	fmt.Println()

	// Create companies
	fmt.Println("4. Creating companies...")
	companies := []*Company{
		{Name: "TechCorp", Industry: "Technology", Founded: 2010},
		{Name: "DataSystems", Industry: "Software", Founded: 2015},
		{Name: "CloudBase", Industry: "Cloud Services", Founded: 2018},
	}

	savedCompanies, err := companyRepo.SaveAll(companies)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Created %d companies\n", len(savedCompanies))
	for _, company := range savedCompanies {
		fmt.Printf("     - %s (ID: %d, Industry: %s)\n", company.Name, *company.ID, company.Industry)
	}
	fmt.Println()

	// Create people
	fmt.Println("5. Creating people...")
	people := []*Person{
		{Name: "Alice Johnson", Age: 30, Email: "alice@example.com", City: "San Francisco", Country: "USA"},
		{Name: "Bob Smith", Age: 28, Email: "bob@example.com", City: "New York", Country: "USA"},
		{Name: "Charlie Brown", Age: 35, Email: "charlie@example.com", City: "London", Country: "UK"},
		{Name: "Diana Prince", Age: 32, Email: "diana@example.com", City: "Seattle", Country: "USA"},
		{Name: "Eve Adams", Age: 29, Email: "eve@example.com", City: "Boston", Country: "USA"},
	}

	savedPeople, err := personRepo.SaveAll(people)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Created %d people\n", len(savedPeople))
	for _, person := range savedPeople {
		fmt.Printf("     - %s (ID: %d, Age: %d)\n", person.Name, *person.ID, person.Age)
	}
	fmt.Println()

	// Manually create relationships using Cypher (demonstrating relationship creation)
	fmt.Println("6. Creating relationships manually...")

	// Alice works for TechCorp
	createRelQuery := fmt.Sprintf(
		"MATCH (p:Person) WHERE id(p) = $person_id MATCH (c:Company) WHERE id(c) = $company_id CREATE (p)-[:WORKS_FOR]->(c)",
	)
	_, err = graph.Query(createRelQuery, map[string]interface{}{
		"person_id":  *savedPeople[0].ID,    // Alice
		"company_id": *savedCompanies[0].ID, // TechCorp
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Alice works for TechCorp")

	// Bob works for DataSystems
	_, err = graph.Query(createRelQuery, map[string]interface{}{
		"person_id":  *savedPeople[1].ID,    // Bob
		"company_id": *savedCompanies[1].ID, // DataSystems
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Bob works for DataSystems")

	// Charlie works for TechCorp
	_, err = graph.Query(createRelQuery, map[string]interface{}{
		"person_id":  *savedPeople[2].ID,    // Charlie
		"company_id": *savedCompanies[0].ID, // TechCorp
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Charlie works for TechCorp")

	// Create KNOWS relationships (friendships)
	knowsRelQuery := "MATCH (p1:Person) WHERE id(p1) = $person1_id MATCH (p2:Person) WHERE id(p2) = $person2_id CREATE (p1)-[:KNOWS]->(p2)"

	// Alice knows Bob
	_, err = graph.Query(knowsRelQuery, map[string]interface{}{
		"person1_id": *savedPeople[0].ID,
		"person2_id": *savedPeople[1].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Alice knows Bob")

	// Alice knows Diana
	_, err = graph.Query(knowsRelQuery, map[string]interface{}{
		"person1_id": *savedPeople[0].ID,
		"person2_id": *savedPeople[3].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Alice knows Diana")

	// Bob knows Charlie
	_, err = graph.Query(knowsRelQuery, map[string]interface{}{
		"person1_id": *savedPeople[1].ID,
		"person2_id": *savedPeople[2].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Bob knows Charlie")

	// Charlie knows Alice (bidirectional)
	_, err = graph.Query(knowsRelQuery, map[string]interface{}{
		"person1_id": *savedPeople[2].ID,
		"person2_id": *savedPeople[0].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Charlie knows Alice")
	fmt.Println()

	// Demonstrate relationship queries using raw Cypher
	fmt.Println("7. Querying relationships...")

	// Find Alice's friends
	fmt.Println()
	fmt.Println("   a) Finding Alice's friends (KNOWS relationship)...")
	friendsQuery := "MATCH (p:Person)-[:KNOWS]->(friend:Person) WHERE id(p) = $person_id RETURN friend"
	result, err := graph.Query(friendsQuery, map[string]interface{}{
		"person_id": *savedPeople[0].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	friendCount := 0
	mapper := orm.NewMapper()
	for result.Next() {
		record := result.Record()
		var zero Person
		friend, err := mapper.MapFromRecord(record, reflect.TypeOf(zero), "friend")
		if err != nil {
			log.Fatal(err)
		}
		friendPerson := friend.(*Person)
		fmt.Printf("      - %s (Age: %d, City: %s)\n", friendPerson.Name, friendPerson.Age, friendPerson.City)
		friendCount++
	}
	fmt.Printf("   ✓ Found %d friends for Alice\n", friendCount)
	fmt.Println()

	// Find who knows Alice (incoming relationships)
	fmt.Println("   b) Finding who knows Alice (incoming KNOWS)...")
	knowsAliceQuery := "MATCH (p:Person)-[:KNOWS]->(alice:Person) WHERE id(alice) = $person_id RETURN p"
	result, err = graph.Query(knowsAliceQuery, map[string]interface{}{
		"person_id": *savedPeople[0].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	knowsCount := 0
	for result.Next() {
		record := result.Record()
		var zero Person
		person, err := mapper.MapFromRecord(record, reflect.TypeOf(zero), "p")
		if err != nil {
			log.Fatal(err)
		}
		knowsPerson := person.(*Person)
		fmt.Printf("      - %s knows Alice\n", knowsPerson.Name)
		knowsCount++
	}
	fmt.Printf("   ✓ %d people know Alice\n", knowsCount)
	fmt.Println()

	// Find employees of TechCorp
	fmt.Println("   c) Finding TechCorp employees (WORKS_FOR relationship)...")
	employeesQuery := "MATCH (p:Person)-[:WORKS_FOR]->(c:Company) WHERE id(c) = $company_id RETURN p"
	result, err = graph.Query(employeesQuery, map[string]interface{}{
		"company_id": *savedCompanies[0].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	employeeCount := 0
	for result.Next() {
		record := result.Record()
		var zero Person
		employee, err := mapper.MapFromRecord(record, reflect.TypeOf(zero), "p")
		if err != nil {
			log.Fatal(err)
		}
		empPerson := employee.(*Person)
		fmt.Printf("      - %s (Age: %d)\n", empPerson.Name, empPerson.Age)
		employeeCount++
	}
	fmt.Printf("   ✓ Found %d employees at TechCorp\n", employeeCount)
	fmt.Println()

	// Find Alice's company
	fmt.Println("   d) Finding where Alice works...")
	companyQuery := "MATCH (p:Person)-[:WORKS_FOR]->(c:Company) WHERE id(p) = $person_id RETURN c"
	result, err = graph.Query(companyQuery, map[string]interface{}{
		"person_id": *savedPeople[0].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	if result.Next() {
		record := result.Record()
		var zeroCompany Company
		company, err := mapper.MapFromRecord(record, reflect.TypeOf(zeroCompany), "c")
		if err != nil {
			log.Fatal(err)
		}
		aliceCompany := company.(*Company)
		fmt.Printf("      - Alice works for: %s (Industry: %s, Founded: %d)\n",
			aliceCompany.Name, aliceCompany.Industry, aliceCompany.Founded)
	}
	fmt.Println()

	// Demonstrate path queries
	fmt.Println("8. Path queries (multi-hop relationships)...")

	// Find friends of Alice's friends (2-hop)
	fmt.Println("   a) Friends of Alice's friends (2-hop)...")
	friendsOfFriendsQuery := `
		MATCH (alice:Person)-[:KNOWS]->(friend:Person)-[:KNOWS]->(fof:Person)
		WHERE id(alice) = $person_id AND id(fof) <> $person_id
		RETURN DISTINCT fof
	`
	result, err = graph.Query(friendsOfFriendsQuery, map[string]interface{}{
		"person_id": *savedPeople[0].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fofCount := 0
	for result.Next() {
		record := result.Record()
		var zero Person
		fof, err := mapper.MapFromRecord(record, reflect.TypeOf(zero), "fof")
		if err != nil {
			log.Fatal(err)
		}
		fofPerson := fof.(*Person)
		fmt.Printf("      - %s (via mutual friend)\n", fofPerson.Name)
		fofCount++
	}
	if fofCount == 0 {
		fmt.Println("      - No friends of friends found (would need more connections)")
	} else {
		fmt.Printf("   ✓ Found %d friends of friends\n", fofCount)
	}
	fmt.Println()

	// Find colleagues (people who work at the same company)
	fmt.Println("   b) Finding Alice's colleagues...")
	colleaguesQuery := `
		MATCH (alice:Person)-[:WORKS_FOR]->(c:Company)<-[:WORKS_FOR]-(colleague:Person)
		WHERE id(alice) = $person_id AND id(colleague) <> $person_id
		RETURN colleague
	`
	result, err = graph.Query(colleaguesQuery, map[string]interface{}{
		"person_id": *savedPeople[0].ID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	colleagueCount := 0
	for result.Next() {
		record := result.Record()
		var zero Person
		colleague, err := mapper.MapFromRecord(record, reflect.TypeOf(zero), "colleague")
		if err != nil {
			log.Fatal(err)
		}
		colPerson := colleague.(*Person)
		fmt.Printf("      - %s (colleague at TechCorp)\n", colPerson.Name)
		colleagueCount++
	}
	fmt.Printf("   ✓ Found %d colleagues\n", colleagueCount)
	fmt.Println()

	// Statistics
	fmt.Println("9. Relationship statistics...")

	// Count total KNOWS relationships
	countKnowsQuery := "MATCH ()-[r:KNOWS]->() RETURN count(r) as count"
	result, err = graph.Query(countKnowsQuery, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	if result.Next() {
		record := result.Record()
		count, _ := record.GetByIndex(0)
		fmt.Printf("   - Total KNOWS relationships: %v\n", count)
	}

	// Count total WORKS_FOR relationships
	countWorksQuery := "MATCH ()-[r:WORKS_FOR]->() RETURN count(r) as count"
	result, err = graph.Query(countWorksQuery, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	if result.Next() {
		record := result.Record()
		count, _ := record.GetByIndex(0)
		fmt.Printf("   - Total WORKS_FOR relationships: %v\n", count)
	}

	// Count people per company
	fmt.Println()
	fmt.Println("   People per company:")
	peoplePerCompanyQuery := `
		MATCH (c:Company)<-[:WORKS_FOR]-(p:Person)
		RETURN c.name as company, count(p) as employee_count
		ORDER BY employee_count DESC
	`
	result, err = graph.Query(peoplePerCompanyQuery, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	for result.Next() {
		record := result.Record()
		companyName, _ := record.GetByIndex(0)
		empCount, _ := record.GetByIndex(1)
		fmt.Printf("      - %v: %v employees\n", companyName, empCount)
	}
	fmt.Println()

	// Demonstrate N+1 prevention concept
	fmt.Println("10. Demonstrating query efficiency...")
	fmt.Println("    (In a full implementation, eager loading would prevent N+1 queries)")
	fmt.Println()
	fmt.Println("    Example: Instead of loading person then company separately,")
	fmt.Println("    use: person, _ := repo.FindByID(id, \"Company\")")
	fmt.Println("    This would load person + company in ONE query using OPTIONAL MATCH")
	fmt.Println()

	fmt.Println("✅ Phase 4 relationships demonstration complete!")
	fmt.Println()
	fmt.Println("Key capabilities demonstrated:")
	fmt.Println("  ✓ Creating entities with repositories")
	fmt.Println("  ✓ Creating relationships (KNOWS, WORKS_FOR)")
	fmt.Println("  ✓ Querying outgoing relationships")
	fmt.Println("  ✓ Querying incoming relationships")
	fmt.Println("  ✓ Bidirectional relationships")
	fmt.Println("  ✓ Multi-hop path queries")
	fmt.Println("  ✓ Relationship statistics")
	fmt.Println()
	fmt.Println("Relationship features ready for use:")
	fmt.Println("  - LazyList[T] and LazySingle[T] for lazy loading")
	fmt.Println("  - Eager loading support with fetch parameter")
	fmt.Println("  - Cascade save/delete infrastructure")
	fmt.Println("  - RelationshipManager for relationship operations")
	fmt.Println("  - Support for 3 directions: OUTGOING, INCOMING, BOTH")
}
