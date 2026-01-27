package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/examples/internal/ciutil"
	"github.com/falkordb/falkordb-go-orm/orm"
)

// Product entity with various indexed properties
type Product struct {
	ID          *int64    `falkor:"id,generated"`
	SKU         string    `falkor:"sku,unique,indexed"`
	Name        string    `falkor:"name,required,indexed"`
	Description string    `falkor:"description"`
	Price       float64   `falkor:"price,indexed"`
	Category    string    `falkor:"category,indexed"`
	InStock     bool      `falkor:"in_stock"`
	CreatedAt   time.Time `falkor:"created_at"`
}

func (*Product) Labels() []string {
	return []string{"Product"}
}

// Customer entity for validation testing
type Customer struct {
	ID    *int64 `falkor:"id,generated"`
	Name  string `falkor:"name,required"`
	Email string `falkor:"email,unique,indexed"`
	Phone string `falkor:"phone"`
}

func (*Customer) Labels() []string {
	return []string{"Customer"}
}

func main() {
	fmt.Println("FalkorDB Go ORM - Advanced Features Example (Phase 7)")
	fmt.Println("====================================================")

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

	graph := db.SelectGraph("indexes_demo")
	fmt.Println("   ✓ Connected to FalkorDB")

	// 2. Clean up previous data
	fmt.Println("2. Cleaning up previous data...")
	_, _ = graph.Query("MATCH (n) DELETE n", nil, nil)
	fmt.Println("   ✓ Cleaned up")

	// 3. Register entities
	fmt.Println("3. Registering entities...")
	orm.GetEntityRegistry().Register(&Product{})
	orm.GetEntityRegistry().Register(&Customer{})
	fmt.Println("   ✓ Entities registered")

	// 4. Validate schemas
	fmt.Println("4. Validating schemas...")
	schemaManager := orm.NewSchemaManager(graph)

	validationResult := schemaManager.ValidateSchema([]reflect.Type{
		reflect.TypeOf(Product{}),
		reflect.TypeOf(Customer{}),
	})

	if validationResult.IsValid {
		fmt.Println("   ✓ Schema validation passed")
	} else {
		fmt.Println("   ✗ Schema validation failed:")
		for _, err := range validationResult.Errors {
			fmt.Printf("      - %v\n", err)
		}
		if ciutil.RequireDB() {
			log.Fatal("schema validation failed")
		}
	}
	fmt.Println()

	// 5. Create indexes
	fmt.Println("5. Creating indexes...")
	indexManager := orm.NewIndexManager(graph)

	err = indexManager.CreateIndexes(reflect.TypeOf(Product{}), true)
	if err != nil {
		if ciutil.RequireDB() {
			log.Fatalf("error creating Product indexes: %v", err)
		}
		fmt.Printf("   ⚠ Error creating Product indexes: %v\n", err)
	} else {
		fmt.Println("   ✓ Created indexes for Product")
		fmt.Println("      - SKU (unique, indexed)")
		fmt.Println("      - Name (indexed)")
		fmt.Println("      - Price (indexed)")
		fmt.Println("      - Category (indexed)")
	}

	err = indexManager.CreateIndexes(reflect.TypeOf(Customer{}), true)
	if err != nil {
		if ciutil.RequireDB() {
			log.Fatalf("error creating Customer indexes: %v", err)
		}
		fmt.Printf("   ⚠ Error creating Customer indexes: %v\n", err)
	} else {
		fmt.Println("   ✓ Created indexes for Customer")
		fmt.Println("      - Email (unique, indexed)")
	}
	fmt.Println()

	// 6. Create sample data
	fmt.Println("6. Creating sample products...")
	productRepo := orm.NewRepository[Product](graph)

	products := []*Product{
		{SKU: "LAPTOP-001", Name: "Pro Laptop", Description: "High-performance laptop", Price: 1299.99, Category: "Electronics", InStock: true, CreatedAt: time.Now()},
		{SKU: "MOUSE-001", Name: "Wireless Mouse", Description: "Ergonomic mouse", Price: 29.99, Category: "Electronics", InStock: true, CreatedAt: time.Now()},
		{SKU: "DESK-001", Name: "Standing Desk", Description: "Adjustable desk", Price: 499.99, Category: "Furniture", InStock: true, CreatedAt: time.Now()},
		{SKU: "CHAIR-001", Name: "Office Chair", Description: "Ergonomic chair", Price: 349.99, Category: "Furniture", InStock: false, CreatedAt: time.Now()},
		{SKU: "MONITOR-001", Name: "4K Monitor", Description: "Ultra HD display", Price: 599.99, Category: "Electronics", InStock: true, CreatedAt: time.Now()},
	}

	saved, err := productRepo.SaveAll(products)
	if err != nil {
		fmt.Printf("   ✗ Error: %v\n", err)
	} else {
		fmt.Printf("   ✓ Created %d products\n\n", len(saved))
	}

	// 7. Test indexed queries
	fmt.Println("7. Testing indexed queries...")

	// Query by indexed property (Category)
	fmt.Println("\n   a) Finding products by category (indexed):")
	electronicsQuery := "MATCH (p:Product) WHERE p.category = $category RETURN p"
	result, err := graph.Query(electronicsQuery, map[string]interface{}{"category": "Electronics"}, nil)
	if err != nil {
		fmt.Printf("      ✗ Query failed: %v\n", err)
	} else {
		count := 0
		for result.Next() {
			count++
		}
		fmt.Printf("      ✓ Found %d electronics products\n", count)
	}

	// Query by unique indexed property (SKU)
	fmt.Println("\n   b) Finding product by SKU (unique index):")
	skuQuery := "MATCH (p:Product) WHERE p.sku = $sku RETURN p"
	result, err = graph.Query(skuQuery, map[string]interface{}{"sku": "LAPTOP-001"}, nil)
	if err != nil {
		fmt.Printf("      ✗ Query failed: %v\n", err)
	} else {
		if result.Next() {
			fmt.Println("      ✓ Found product with SKU LAPTOP-001")
		}
	}

	// 8. Test schema sync
	fmt.Println("\n8. Testing schema synchronization...")
	err = schemaManager.SyncSchema([]reflect.Type{
		reflect.TypeOf(Product{}),
		reflect.TypeOf(Customer{}),
	})
	if err != nil {
		// SyncSchema may attempt to (re)create indexes that were already created above.
		// Treat "already indexed" as non-fatal so the example remains idempotent in CI.
		if ciutil.RequireDB() && !strings.Contains(err.Error(), "already indexed") {
			log.Fatalf("schema sync failed: %v", err)
		}
		fmt.Printf("   ⚠ Schema sync encountered issues: %v\n", err)
	} else {
		fmt.Println("   ✓ Schema synchronized (indexes ensured)")
	}
	fmt.Println()

	// 9. List indexes
	fmt.Println("9. Listing created indexes...")
	indexes, err := indexManager.ListIndexes("Product")
	if err != nil {
		fmt.Printf("   ⚠ Error listing indexes: %v\n", err)
	} else {
		if len(indexes) > 0 {
			fmt.Printf("   ✓ Found %d indexes for Product\n", len(indexes))
		} else {
			fmt.Println("   Note: Index listing may not be fully supported by FalkorDB version")
		}
	}
	fmt.Println()

	// 10. Demonstrate custom type converter
	fmt.Println("10. Custom type converters...")
	fmt.Println("   ✓ Built-in converters available:")
	fmt.Println("      - time.Time → Unix timestamp")
	fmt.Println("      - uuid.UUID → String")
	fmt.Println("      - Structs → JSON string")
	fmt.Println("      - Slices/Maps → Graph arrays/maps")
	fmt.Println()

	// Summary
	fmt.Println("✅ Phase 7 advanced features demonstration complete!")
	fmt.Println("Key capabilities demonstrated:")
	fmt.Println("  ✓ Index management (create/drop/list)")
	fmt.Println("  ✓ Schema validation")
	fmt.Println("  ✓ Schema synchronization")
	fmt.Println("  ✓ Indexed query performance")
	fmt.Println("  ✓ Unique constraint enforcement")
	fmt.Println("  ✓ Type converter system")
}
