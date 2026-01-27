package main

import (
	"fmt"
	"log"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/examples/internal/ciutil"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// Person entity
type Person struct {
	ID      *int64 `falkor:"id,generated"`
	Name    string `falkor:"name,required"`
	Email   string `falkor:"email,unique"`
	Age     int    `falkor:"age,indexed"`
	Country string `falkor:"country,interned"`
}

func (*Person) Labels() []string {
	return []string{"Person"}
}

func main() {
	fmt.Println("FalkorDB Go ORM - Repository Pattern Example (Phase 2)")
	fmt.Println("=======================================================")

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

	graph := db.SelectGraph("repository_example")
	fmt.Println("   ✓ Connected to FalkorDB")

	// Create repository
	fmt.Println("2. Creating Person repository...")
	repo := orm.NewRepository[Person](graph)
	fmt.Println("   ✓ Repository created")

	// Clean up previous data
	fmt.Println("3. Cleaning up previous data...")
	repo.DeleteAllEntities()
	fmt.Println("   ✓ Cleaned up")

	// Example 1: Save entities
	fmt.Println("4. Saving entities...")
	alice := &Person{Name: "Alice Johnson", Email: "alice@example.com", Age: 28, Country: "USA"}
	bob := &Person{Name: "Bob Smith", Email: "bob@example.com", Age: 35, Country: "UK"}
	charlie := &Person{Name: "Charlie Brown", Email: "charlie@example.com", Age: 22, Country: "Canada"}

	alice, err = repo.Save(alice)
	if err != nil {
		log.Fatal("Failed to save Alice:", err)
	}
	fmt.Printf("   ✓ Saved Alice with ID: %d\n", *alice.ID)

	bob, err = repo.Save(bob)
	if err != nil {
		log.Fatal("Failed to save Bob:", err)
	}
	fmt.Printf("   ✓ Saved Bob with ID: %d\n", *bob.ID)

	charlie, err = repo.Save(charlie)
	if err != nil {
		log.Fatal("Failed to save Charlie:", err)
	}
	fmt.Printf("   ✓ Saved Charlie with ID: %d\n", *charlie.ID)
	fmt.Println()

	// Example 2: FindByID
	fmt.Println("5. Finding entity by ID...")
	found, err := repo.FindByID(*alice.ID)
	if err != nil {
		log.Fatal("Failed to find by ID:", err)
	}
	if found != nil {
		fmt.Printf("   ✓ Found: %s (ID: %d, Age: %d)\n", found.Name, *found.ID, found.Age)
	}
	fmt.Println()

	// Example 3: FindAll
	fmt.Println("6. Finding all entities...")
	all, err := repo.FindAll()
	if err != nil {
		log.Fatal("Failed to find all:", err)
	}
	fmt.Printf("   ✓ Found %d entities:\n", len(all))
	for _, p := range all {
		fmt.Printf("     - %s (Age: %d, Country: %s)\n", p.Name, p.Age, p.Country)
	}
	fmt.Println()

	// Example 4: Count
	fmt.Println("7. Counting entities...")
	count, err := repo.Count()
	if err != nil {
		log.Fatal("Failed to count:", err)
	}
	fmt.Printf("   ✓ Total count: %d\n\n", count)

	// Example 5: ExistsByID
	fmt.Println("8. Checking if entity exists...")
	exists, err := repo.ExistsByID(*bob.ID)
	if err != nil {
		log.Fatal("Failed to check exists:", err)
	}
	fmt.Printf("   ✓ Bob exists: %v\n\n", exists)

	// Example 6: Update entity
	fmt.Println("9. Updating entity...")
	alice.Age = 29 // Birthday!
	alice, err = repo.Save(alice)
	if err != nil {
		log.Fatal("Failed to update Alice:", err)
	}
	fmt.Printf("   ✓ Updated Alice's age to %d\n\n", alice.Age)

	// Example 7: Pagination
	fmt.Println("10. Paginated query (page 0, size 2)...")
	pageable := orm.NewPageable(0, 2).WithSort("age", "ASC")
	page, err := repo.FindAllPaginated(pageable)
	if err != nil {
		log.Fatal("Failed to paginate:", err)
	}
	fmt.Printf("   ✓ Page %d of %d (Total: %d)\n", page.PageNumber+1, page.TotalPages, page.TotalElements)
	for _, p := range page.Content {
		fmt.Printf("     - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Printf("   Has next page: %v\n\n", page.HasNext())

	// Example 8: Query Builder API
	fmt.Println("11. Query builder - find people older than 25...")
	chain := repo.Query().
		Where("age", ">", 25).
		OrderByAsc("name").
		Limit(10)

	results, err := repo.ExecuteQuery(chain)
	if err != nil {
		log.Fatal("Failed to execute query:", err)
	}
	fmt.Printf("   ✓ Found %d people older than 25:\n", len(results))
	for _, p := range results {
		fmt.Printf("     - %s (Age: %d)\n", p.Name, p.Age)
	}
	fmt.Println()

	// Example 9: SaveAll
	fmt.Println("12. Saving multiple entities at once...")
	newPeople := []*Person{
		{Name: "David Lee", Email: "david@example.com", Age: 42, Country: "USA"},
		{Name: "Eve Wilson", Email: "eve@example.com", Age: 31, Country: "UK"},
	}
	saved, err := repo.SaveAll(newPeople)
	if err != nil {
		log.Fatal("Failed to save all:", err)
	}
	fmt.Printf("   ✓ Saved %d entities\n\n", len(saved))

	// Example 10: FindAllByID
	fmt.Println("13. Finding multiple entities by IDs...")
	ids := []interface{}{*alice.ID, *bob.ID}
	foundMany, err := repo.FindAllByID(ids)
	if err != nil {
		log.Fatal("Failed to find by IDs:", err)
	}
	fmt.Printf("   ✓ Found %d entities by IDs\n\n", len(foundMany))

	// Final count
	fmt.Println("14. Final count...")
	finalCount, err := repo.Count()
	if err != nil {
		log.Fatal("Failed to count:", err)
	}
	fmt.Printf("   ✓ Total entities: %d\n\n", finalCount)

	// Example 11: Delete
	fmt.Println("15. Deleting an entity...")
	err = repo.Delete(charlie)
	if err != nil {
		log.Fatal("Failed to delete:", err)
	}
	fmt.Printf("   ✓ Deleted Charlie\n\n")

	// Verify deletion
	afterDeleteCount, err := repo.Count()
	if err != nil {
		log.Fatal("Failed to count after delete:", err)
	}
	fmt.Printf("   Count after deletion: %d\n\n", afterDeleteCount)
	ciutil.Check(afterDeleteCount == finalCount-1, "expected count after delete to be %d, got %d", finalCount-1, afterDeleteCount)

	fmt.Println("✅ Phase 2 demonstration complete!")
	fmt.Println("\nRepository Pattern Features Demonstrated:")
	fmt.Println("  ✓ Save (create and update)")
	fmt.Println("  ✓ FindByID")
	fmt.Println("  ✓ FindAll")
	fmt.Println("  ✓ FindAllByID")
	fmt.Println("  ✓ Count")
	fmt.Println("  ✓ ExistsByID")
	fmt.Println("  ✓ Delete / DeleteByID")
	fmt.Println("  ✓ SaveAll")
	fmt.Println("  ✓ Pagination with sorting")
	fmt.Println("  ✓ Query Builder API")
}
