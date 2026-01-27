package main

import (
	"fmt"
	"log"
	"reflect"

	falkordb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/falkordb/falkordb-go-orm/examples/internal/ciutil"
	"github.com/falkordb/falkordb-go-orm/orm"
	"github.com/falkordb/falkordb-go-orm/security"
)

// Employee entity with sensitive data
type Employee struct {
	ID         *int64  `falkor:"id,generated"`
	Name       string  `falkor:"name,required"`
	Email      string  `falkor:"email,unique"`
	Position   string  `falkor:"position"`
	Salary     float64 `falkor:"salary"` // Sensitive
	SSN        string  `falkor:"ssn"`    // Very sensitive
	Department string  `falkor:"department"`
}

func (*Employee) Labels() []string {
	return []string{"Employee"}
}

func main() {
	fmt.Println("FalkorDB Go ORM - RBAC Security Example (Phase 6)")
	fmt.Println("=================================================")

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

	graph := db.SelectGraph("security_demo")
	fmt.Println("   ✓ Connected to FalkorDB")

	// 2. Clean up previous data
	fmt.Println("2. Cleaning up previous data...")
	_, _ = graph.Query("MATCH (n) DELETE n", nil, nil)
	fmt.Println("   ✓ Cleaned up")

	// 3. Register entities
	fmt.Println("3. Registering entities...")
	orm.GetEntityRegistry().Register(&Employee{})
	orm.GetEntityRegistry().Register(&security.User{})
	orm.GetEntityRegistry().Register(&security.Role{})
	orm.GetEntityRegistry().Register(&security.Privilege{})
	fmt.Println("   ✓ Entities registered")

	// 4. Configure security metadata for Employee
	fmt.Println("4. Configuring security metadata...")
	security.RegisterSecurityMetadata(
		reflect.TypeOf(Employee{}),
		security.NewSecurityMetadata().
			WithReadRoles("reader", "hr", "admin").
			WithWriteRoles("hr", "admin").
			WithCreateRoles("hr", "admin").
			WithDeleteRoles("admin").
			DenyReadProperty("salary", "reader").    // Readers can't see salary
			DenyReadProperty("ssn", "reader", "hr"). // Only admins can see SSN
			DenyWriteProperty("salary", "reader"),   // Readers can't modify salary
	)
	fmt.Println("   ✓ Security metadata configured")
	fmt.Println("      - Read roles: reader, hr, admin")
	fmt.Println("      - Write roles: hr, admin")
	fmt.Println("      - Salary hidden from: reader")
	fmt.Println("      - SSN hidden from: reader, hr")

	// 5. Create RBAC manager
	fmt.Println("5. Setting up RBAC manager...")
	manager := security.NewRBACManager(graph)
	fmt.Println("   ✓ RBAC manager created")

	// 6. Create roles
	fmt.Println("6. Creating roles...")
	if _, err := manager.CreateRole("reader", "Can read basic employee info", false); err != nil {
		log.Fatal(err)
	}
	if _, err := manager.CreateRole("hr", "HR staff with salary access", false); err != nil {
		log.Fatal(err)
	}
	if _, err := manager.CreateRole("admin", "Full access", true); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Created 3 roles (reader, hr, admin)\n\n")

	// 7. Create users
	fmt.Println("7. Creating users...")
	alice, err := manager.CreateUser("alice", "alice@company.com")
	if err != nil {
		log.Fatal(err)
	}
	bob, err := manager.CreateUser("bob", "bob@company.com")
	if err != nil {
		log.Fatal(err)
	}
	charlie, err := manager.CreateUser("charlie", "charlie@company.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Created 3 users (alice, bob, charlie)\n\n")

	// 8. Assign roles
	fmt.Println("8. Assigning roles...")
	if err := manager.AssignRole("alice", "reader"); err != nil {
		log.Fatal(err)
	}
	if err := manager.AssignRole("bob", "hr"); err != nil {
		log.Fatal(err)
	}
	if err := manager.AssignRole("charlie", "admin"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ alice: reader")
	fmt.Println("   ✓ bob: hr")
	fmt.Println("   ✓ charlie: admin")

	// 9. Grant privileges
	fmt.Println("9. Granting privileges...")
	if err := manager.GrantPrivilege("reader", security.ActionRead, security.ResourceTypeNode, "Employee"); err != nil {
		log.Fatal(err)
	}
	if err := manager.GrantPrivilege("hr", security.ActionRead, security.ResourceTypeNode, "Employee"); err != nil {
		log.Fatal(err)
	}
	if err := manager.GrantPrivilege("hr", security.ActionWrite, security.ResourceTypeNode, "Employee"); err != nil {
		log.Fatal(err)
	}
	if err := manager.GrantPrivilege("admin", security.ActionRead, security.ResourceTypeNode, "*"); err != nil {
		log.Fatal(err)
	}
	if err := manager.GrantPrivilege("admin", security.ActionWrite, security.ResourceTypeNode, "*"); err != nil {
		log.Fatal(err)
	}
	if err := manager.GrantPrivilege("admin", security.ActionCreate, security.ResourceTypeNode, "*"); err != nil {
		log.Fatal(err)
	}
	if err := manager.GrantPrivilege("admin", security.ActionDelete, security.ResourceTypeNode, "*"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Privileges granted to roles")

	// 10. Create sample employees (as admin)
	fmt.Println("10. Creating sample employees...")
	employeeRepo := orm.NewRepository[Employee](graph)

	employees := []*Employee{
		{Name: "John Doe", Email: "john@company.com", Position: "Engineer", Salary: 85000, SSN: "123-45-6789", Department: "Engineering"},
		{Name: "Jane Smith", Email: "jane@company.com", Position: "Manager", Salary: 105000, SSN: "987-65-4321", Department: "Management"},
		{Name: "Mike Johnson", Email: "mike@company.com", Position: "Developer", Salary: 75000, SSN: "555-12-3456", Department: "Engineering"},
	}

	saved, err := employeeRepo.SaveAll(employees)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Created %d employees\n\n", len(saved))
	ciutil.Check(len(saved) == 3, "expected 3 saved employees, got %d", len(saved))

	// 11. Demonstrate security contexts
	fmt.Println("11. Testing security with different user contexts...")

	// Create security contexts (simplified - using manual role assignment for demo)
	aliceCtx := createTestContext(graph, alice, []string{"reader"})
	bobCtx := createTestContext(graph, bob, []string{"hr"})
	charlieCtx := createTestContext(graph, charlie, []string{"admin"})

	// Test as Alice (reader)
	fmt.Println("    a) Alice (reader) viewing employees:")
	testAsUser(graph, aliceCtx, "Alice")

	// Test as Bob (HR)
	fmt.Println("\n    b) Bob (HR) viewing employees:")
	testAsUser(graph, bobCtx, "Bob")

	// Test as Charlie (admin)
	fmt.Println("\n    c) Charlie (admin) viewing employees:")
	testAsUser(graph, charlieCtx, "Charlie")

	// 12. Test permission denials
	fmt.Println("\n12. Testing permission denials...")

	// Alice tries to create (should fail)
	fmt.Println("    a) Alice (reader) trying to create employee:")
	aliceSecureRepo := security.NewSecureRepository[Employee](graph, aliceCtx)
	newEmp := &Employee{Name: "Test User", Email: "test@company.com", Position: "Tester"}
	_, err = aliceSecureRepo.Save(newEmp)
	if err == security.ErrAccessDenied {
		fmt.Println("       ✓ Correctly denied (no CREATE permission)")
	} else {
		fmt.Printf("       ✗ Should have been denied: %v\n", err)
	}

	// Alice tries to delete (should fail)
	fmt.Println("\n    b) Alice (reader) trying to delete employee:")
	err = aliceSecureRepo.DeleteByID(*saved[0].ID)
	if err == security.ErrAccessDenied {
		fmt.Println("       ✓ Correctly denied (no DELETE permission)")
	} else {
		fmt.Printf("       ✗ Should have been denied: %v\n", err)
	}

	// Bob (HR) can create
	fmt.Println("\n    c) Bob (HR) creating employee:")
	bobSecureRepo := security.NewSecureRepository[Employee](graph, bobCtx)
	hrEmp := &Employee{Name: "HR Created", Email: "hr@company.com", Position: "Clerk", Salary: 50000}
	created, err := bobSecureRepo.Save(hrEmp)
	if err == nil {
		fmt.Printf("       ✓ Successfully created employee (ID: %d)\n", *created.ID)
	} else {
		fmt.Printf("       ✗ Failed: %v\n", err)
	}

	// 13. Summary
	fmt.Println("\n✅ Phase 6 RBAC demonstration complete!")
	fmt.Println("Key capabilities demonstrated:")
	fmt.Println("  ✓ User and role management")
	fmt.Println("  ✓ Privilege assignment (READ, WRITE, CREATE, DELETE)")
	fmt.Println("  ✓ Property-level access control")
	fmt.Println("  ✓ Role-based filtering (salary, SSN)")
	fmt.Println("  ✓ Permission checks (create/delete denied)")
	fmt.Println("  ✓ SecureRepository with automatic filtering")
}

// createTestContext creates a test security context with manual role assignment
func createTestContext(graph *falkordb.Graph, user *security.User, roles []string) *security.SecurityContext {
	// Create context
	ctx := &security.SecurityContext{}

	// Manually set roles for demo (in production, would load from graph)
	for _, role := range roles {
		ctx.AddRole(role)
	}

	// Add privileges based on roles
	if containsRole(roles, "reader") {
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionRead,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "Employee",
			GrantType:     security.GrantTypeGrant,
		})
	}

	if containsRole(roles, "hr") {
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionRead,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "Employee",
			GrantType:     security.GrantTypeGrant,
		})
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionWrite,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "Employee",
			GrantType:     security.GrantTypeGrant,
		})
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionCreate,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "Employee",
			GrantType:     security.GrantTypeGrant,
		})
	}

	if containsRole(roles, "admin") {
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionRead,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "*",
			GrantType:     security.GrantTypeGrant,
		})
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionWrite,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "*",
			GrantType:     security.GrantTypeGrant,
		})
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionCreate,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "*",
			GrantType:     security.GrantTypeGrant,
		})
		ctx.AddPrivilege(&security.Privilege{
			Action:        security.ActionDelete,
			ResourceType:  security.ResourceTypeNode,
			ResourceLabel: "*",
			GrantType:     security.GrantTypeGrant,
		})
	}

	return ctx
}

func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func testAsUser(graph *falkordb.Graph, ctx *security.SecurityContext, userName string) {
	secureRepo := security.NewSecureRepository[Employee](graph, ctx)

	employees, err := secureRepo.FindAll()
	if err != nil {
		fmt.Printf("       ✗ Access denied: %v\n", err)
		return
	}

	fmt.Printf("       ✓ Found %d employees:\n", len(employees))
	for _, emp := range employees {
		fmt.Printf("          - %s (%s)\n", emp.Name, emp.Position)
		fmt.Printf("            Email: %s, Dept: %s\n", emp.Email, emp.Department)

		// Check salary visibility
		if emp.Salary > 0 {
			fmt.Printf("            Salary: $%.2f\n", emp.Salary)
		} else {
			fmt.Printf("            Salary: [HIDDEN]\n")
		}

		// Check SSN visibility
		if emp.SSN != "" {
			fmt.Printf("            SSN: %s\n", emp.SSN)
		} else {
			fmt.Printf("            SSN: [HIDDEN]\n")
		}
	}
}
