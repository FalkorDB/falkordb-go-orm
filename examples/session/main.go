package main

import (
	"fmt"
	"log"
	"reflect"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/examples/internal/ciutil"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// Person entity for session demo
type Person struct {
	ID    *int64 `falkor:"id,generated"`
	Name  string `falkor:"name,required"`
	Age   int    `falkor:"age"`
	City  string `falkor:"city"`
	Email string `falkor:"email"`
}

func (*Person) Labels() []string {
	return []string{"Person"}
}

// Product entity for batch operations
type Product struct {
	ID    *int64  `falkor:"id,generated"`
	Name  string  `falkor:"name,required"`
	Price float64 `falkor:"price"`
	Stock int     `falkor:"stock"`
}

func (*Product) Labels() []string {
	return []string{"Product"}
}

func main() {
	fmt.Println("FalkorDB Go ORM - Session & Transactions Example (Phase 5)")
	fmt.Println("===========================================================")

	// 1. Connect to FalkorDB
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

	graph := db.SelectGraph("session_demo")
	fmt.Println("   ✓ Connected to FalkorDB")

	// 2. Clean up previous data
	fmt.Println("2. Cleaning up previous data...")
	_, _ = graph.Query("MATCH (n) DELETE n", nil, nil)
	fmt.Println("   ✓ Cleaned up")

	// Register entities
	orm.GetEntityRegistry().Register(&Person{})
	orm.GetEntityRegistry().Register(&Product{})

	// Demonstrate identity map
	demonstrateIdentityMap(graph)

	// Demonstrate dirty checking
	demonstrateDirtyChecking(graph)

	// Demonstrate transaction commit
	demonstrateTransactionCommit(graph)

	// Demonstrate transaction rollback
	demonstrateTransactionRollback(graph)

	// Demonstrate WithSession helper
	demonstrateWithSession(graph)

	// Demonstrate batch operations
	demonstrateBatchOperations(graph)

	fmt.Println("\n✅ Phase 5 session demonstration complete!")
	fmt.Println("Key capabilities demonstrated:")
	fmt.Println("  ✓ Identity map (prevents duplicate entity instances)")
	fmt.Println("  ✓ Dirty checking (tracks entity changes)")
	fmt.Println("  ✓ Transaction commit (persists changes)")
	fmt.Println("  ✓ Transaction rollback (discards changes)")
	fmt.Println("  ✓ WithSession helper (automatic lifecycle)")
	fmt.Println("  ✓ Batch operations (flush multiple changes)")
}

func demonstrateIdentityMap(graph *falkordb.Graph) {
	fmt.Println("3. Demonstrating Identity Map...")

	// Create a person first
	repo := orm.NewRepository[Person](graph)
	alice := &Person{Name: "Alice Johnson", Age: 30, City: "San Francisco", Email: "alice@example.com"}
	saved, err := repo.Save(alice)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created: %s (ID: %d)\n", saved.Name, *saved.ID)

	// Use session to get the same entity multiple times
	session := orm.NewSession(graph)
	defer session.Close()

	// First get - loads from database
	entity1, err := session.Get(reflect.TypeOf(Person{}), *saved.ID)
	if err != nil {
		log.Fatal(err)
	}
	person1 := entity1.(*Person)
	fmt.Printf("   First Get:  %s (Age: %d, City: %s)\n", person1.Name, person1.Age, person1.City)

	// Second get - returns from identity map (same instance)
	entity2, err := session.Get(reflect.TypeOf(Person{}), *saved.ID)
	if err != nil {
		log.Fatal(err)
	}
	person2 := entity2.(*Person)
	fmt.Printf("   Second Get: %s (Age: %d, City: %s)\n", person2.Name, person2.Age, person2.City)

	// Verify it's the same instance
	if person1 == person2 {
		fmt.Println("   ✓ Identity map working - same instance returned!")
	} else {
		fmt.Println("   ✗ Different instances - identity map not working")
	}

	// Modify one instance
	person1.Age = 31

	// Other instance should see the change (same memory location)
	if person2.Age == 31 {
		fmt.Println("   ✓ Changes reflected in both references (same instance)")
	}
}

func demonstrateDirtyChecking(graph *falkordb.Graph) {
	fmt.Println("4. Demonstrating Dirty Checking...")

	// Create a person
	repo := orm.NewRepository[Person](graph)
	bob := &Person{Name: "Bob Smith", Age: 28, City: "New York", Email: "bob@example.com"}
	saved, err := repo.Save(bob)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created: %s (ID: %d, Age: %d)\n", saved.Name, *saved.ID, saved.Age)

	// Use session with transaction
	session := orm.NewSession(graph)
	defer session.Close()

	err = session.BeginTransaction()
	if err != nil {
		log.Fatal(err)
	}

	// Load entity
	entity, err := session.Get(reflect.TypeOf(Person{}), *saved.ID)
	if err != nil {
		log.Fatal(err)
	}
	person := entity.(*Person)
	fmt.Printf("   Loaded: %s (Age: %d)\n", person.Name, person.Age)

	// Modify entity
	person.Age = 29
	person.City = "Boston"
	fmt.Printf("   Modified: Age %d -> %d, City %s -> %s\n", 28, person.Age, "New York", person.City)

	// Mark as dirty and flush
	session.MarkDirty(person)
	fmt.Println("   Marked as dirty")

	// Commit (will flush changes)
	err = session.Commit()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Changes committed")

	// Verify changes persisted
	updated, err := repo.FindByID(*saved.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Verified: %s (Age: %d, City: %s)\n", updated.Name, updated.Age, updated.City)
	if updated.Age == 29 && updated.City == "Boston" {
		fmt.Println("   ✓ Dirty checking and update successful!")
	}
}

func demonstrateTransactionCommit(graph *falkordb.Graph) {
	fmt.Println("5. Demonstrating Transaction Commit...")

	session := orm.NewSession(graph)
	defer session.Close()

	err := session.BeginTransaction()
	if err != nil {
		log.Fatal(err)
	}

	// Add multiple new entities
	charlie := &Person{Name: "Charlie Brown", Age: 35, City: "Chicago", Email: "charlie@example.com"}
	diana := &Person{Name: "Diana Prince", Age: 32, City: "Seattle", Email: "diana@example.com"}

	session.Add(charlie)
	session.Add(diana)
	fmt.Println("   Added 2 new entities to session")

	// Commit transaction (will flush)
	err = session.Commit()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Transaction committed")

	// Verify entities were saved
	repo := orm.NewRepository[Person](graph)
	all, err := repo.FindAll()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Total people in database: %d\n", len(all))
	for _, p := range all {
		fmt.Printf("      - %s (Age: %d, City: %s)\n", p.Name, p.Age, p.City)
	}
	fmt.Println()
}

func demonstrateTransactionRollback(graph *falkordb.Graph) {
	fmt.Println("6. Demonstrating Transaction Rollback...")

	// Count before
	repo := orm.NewRepository[Person](graph)
	beforeCount, _ := repo.Count()
	fmt.Printf("   People before: %d\n", beforeCount)

	session := orm.NewSession(graph)
	defer session.Close()

	err := session.BeginTransaction()
	if err != nil {
		log.Fatal(err)
	}

	// Add entity
	eve := &Person{Name: "Eve Adams", Age: 29, City: "Miami", Email: "eve@example.com"}
	session.Add(eve)
	fmt.Println("   Added Eve to session (not yet persisted)")

	// Rollback instead of commit
	err = session.Rollback()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Transaction rolled back")

	// Verify entity was NOT saved
	afterCount, _ := repo.Count()
	fmt.Printf("   People after rollback: %d\n", afterCount)

	if beforeCount == afterCount {
		fmt.Println("   ✓ Rollback successful - entity not persisted!")
	} else {
		log.Fatalf("Rollback failed: expected count %d, got %d", beforeCount, afterCount)
	}
}

func demonstrateWithSession(graph *falkordb.Graph) {
	fmt.Println("7. Demonstrating WithSession Helper...")

	// Success case - auto commit
	err := orm.WithSession(graph, func(session *orm.Session) error {
		frank := &Person{Name: "Frank Miller", Age: 40, City: "Austin", Email: "frank@example.com"}
		session.Add(frank)
		fmt.Println("   Added Frank in WithSession")
		return nil // Success - will auto-commit
	})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ WithSession auto-committed on success")

	// Verify saved
	repo := orm.NewRepository[Person](graph)
	all, _ := repo.FindAll()
	frankFound := false
	for _, p := range all {
		if p.Name == "Frank Miller" {
			frankFound = true
			fmt.Printf("   Found: %s (Age: %d)\n", p.Name, p.Age)
			break
		}
	}
	if frankFound {
		fmt.Println("   ✓ Auto-commit successful!")
	}

	// Error case - auto rollback
	beforeCount, _ := repo.Count()
	err = orm.WithSession(graph, func(session *orm.Session) error {
		grace := &Person{Name: "Grace Hopper", Age: 45, City: "Boston", Email: "grace@example.com"}
		session.Add(grace)
		fmt.Println("   Added Grace in WithSession")
		return fmt.Errorf("simulated error") // Error - will auto-rollback
	})

	if err != nil {
		fmt.Printf("   Expected error: %v\n", err)
	}
	fmt.Println("   ✓ WithSession auto-rolled back on error")

	// Verify not saved
	afterCount, _ := repo.Count()
	if beforeCount == afterCount {
		fmt.Println("   ✓ Auto-rollback successful - Grace not persisted!")
	}
}

func demonstrateBatchOperations(graph *falkordb.Graph) {
	fmt.Println("8. Demonstrating Batch Operations...")

	session := orm.NewSession(graph)
	defer session.Close()

	err := session.BeginTransaction()
	if err != nil {
		log.Fatal(err)
	}

	// Add multiple products in batch
	products := []*Product{
		{Name: "Laptop", Price: 999.99, Stock: 10},
		{Name: "Mouse", Price: 29.99, Stock: 50},
		{Name: "Keyboard", Price: 79.99, Stock: 30},
		{Name: "Monitor", Price: 299.99, Stock: 15},
		{Name: "Webcam", Price: 59.99, Stock: 25},
	}

	for _, product := range products {
		session.Add(product)
	}
	fmt.Printf("   Added %d products to session\n", len(products))

	// Flush all at once
	err = session.Commit()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Batch committed")

	// Verify all products saved
	productRepo := orm.NewRepository[Product](graph)
	allProducts, err := productRepo.FindAll()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Total products: %d\n", len(allProducts))
	var totalValue float64
	for _, p := range allProducts {
		value := p.Price * float64(p.Stock)
		totalValue += value
		fmt.Printf("      - %s: $%.2f × %d = $%.2f\n", p.Name, p.Price, p.Stock, value)
	}
	fmt.Printf("   Total inventory value: $%.2f\n", totalValue)
	fmt.Println("   ✓ Batch operations successful!")
}
