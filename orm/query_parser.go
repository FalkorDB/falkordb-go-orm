package orm

import (
	"fmt"
	"strings"
	"sync"
)

// Operation represents the type of query operation
type Operation int

const (
	OpFind Operation = iota
	OpCount
	OpExists
	OpDelete
)

func (o Operation) String() string {
	switch o {
	case OpFind:
		return "FIND"
	case OpCount:
		return "COUNT"
	case OpExists:
		return "EXISTS"
	case OpDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// Operator represents comparison operators in queries
type Operator int

const (
	OpEqual Operator = iota
	OpNotEqual
	OpGreaterThan
	OpGreaterThanOrEqual
	OpLessThan
	OpLessThanOrEqual
	OpBetween
	OpIn
	OpNotIn
	OpLike
	OpNotLike
	OpContains
	OpNotContains
	OpStartsWith
	OpEndsWith
	OpIsNull
	OpIsNotNull
)

func (o Operator) String() string {
	switch o {
	case OpEqual:
		return "="
	case OpNotEqual:
		return "<>"
	case OpGreaterThan:
		return ">"
	case OpGreaterThanOrEqual:
		return ">="
	case OpLessThan:
		return "<"
	case OpLessThanOrEqual:
		return "<="
	case OpBetween:
		return "BETWEEN"
	case OpIn:
		return "IN"
	case OpNotIn:
		return "NOT IN"
	case OpLike:
		return "LIKE"
	case OpNotLike:
		return "NOT LIKE"
	case OpContains:
		return "CONTAINS"
	case OpNotContains:
		return "NOT CONTAINS"
	case OpStartsWith:
		return "STARTS WITH"
	case OpEndsWith:
		return "ENDS WITH"
	case OpIsNull:
		return "IS NULL"
	case OpIsNotNull:
		return "IS NOT NULL"
	default:
		return "UNKNOWN"
	}
}

// ToCypher converts operator to Cypher syntax
func (o Operator) ToCypher() string {
	switch o {
	case OpContains:
		return "CONTAINS"
	case OpNotContains:
		return "NOT ... CONTAINS" // Handled specially
	case OpStartsWith:
		return "STARTS WITH"
	case OpEndsWith:
		return "ENDS WITH"
	default:
		return o.String()
	}
}

// ParamCount returns how many parameters this operator requires
func (o Operator) ParamCount() int {
	switch o {
	case OpBetween:
		return 2
	case OpIsNull, OpIsNotNull:
		return 0
	default:
		return 1
	}
}

// LogicalOperator represents AND/OR between conditions
type LogicalOperator int

const (
	LogicalAnd LogicalOperator = iota
	LogicalOr
)

func (l LogicalOperator) String() string {
	if l == LogicalOr {
		return "OR"
	}
	return "AND"
}

// SortDirection represents sort direction
type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

func (s SortDirection) String() string {
	if s == SortDesc {
		return "DESC"
	}
	return "ASC"
}

// Condition represents a single query condition
type Condition struct {
	PropertyName string
	Operator     Operator
}

// OrderClause represents an ORDER BY clause
type OrderClause struct {
	PropertyName string
	Direction    SortDirection
}

// QuerySpec represents a parsed query specification
type QuerySpec struct {
	Operation       Operation
	Conditions      []Condition
	LogicalOperator LogicalOperator
	Ordering        []OrderClause
	Limit           *int
}

// QueryParser parses method names into QuerySpec
type QueryParser struct {
	cache sync.Map // method name -> *QuerySpec
}

// NewQueryParser creates a new query parser
func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

// Parse parses a method name into a QuerySpec
// Supported patterns:
//   - FindByName
//   - FindByAgeGreaterThan
//   - FindByNameAndAge
//   - FindByNameOrAge
//   - CountByAge
//   - ExistsByEmail
//   - DeleteByID
func (p *QueryParser) Parse(methodName string) (*QuerySpec, error) {
	// Check cache first
	if cached, ok := p.cache.Load(methodName); ok {
		return cached.(*QuerySpec), nil
	}

	spec, err := p.parse(methodName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	p.cache.Store(methodName, spec)
	return spec, nil
}

func (p *QueryParser) parse(methodName string) (*QuerySpec, error) {
	spec := &QuerySpec{
		LogicalOperator: LogicalAnd, // Default to AND
	}

	// Determine operation type
	var remainder string
	switch {
	case strings.HasPrefix(methodName, "FindBy"):
		spec.Operation = OpFind
		remainder = strings.TrimPrefix(methodName, "FindBy")
	case strings.HasPrefix(methodName, "CountBy"):
		spec.Operation = OpCount
		remainder = strings.TrimPrefix(methodName, "CountBy")
	case strings.HasPrefix(methodName, "ExistsBy"):
		spec.Operation = OpExists
		remainder = strings.TrimPrefix(methodName, "ExistsBy")
	case strings.HasPrefix(methodName, "DeleteBy"):
		spec.Operation = OpDelete
		remainder = strings.TrimPrefix(methodName, "DeleteBy")
	default:
		return nil, fmt.Errorf("unsupported method name pattern: %s", methodName)
	}

	if remainder == "" {
		return nil, fmt.Errorf("invalid method name: %s (missing property specification)", methodName)
	}

	// Check for OrderBy clause
	if idx := strings.Index(remainder, "OrderBy"); idx != -1 {
		orderPart := remainder[idx+7:] // Skip "OrderBy"
		remainder = remainder[:idx]

		// Parse ordering
		orderSpec, err := p.parseOrdering(orderPart)
		if err != nil {
			return nil, fmt.Errorf("invalid ordering in %s: %v", methodName, err)
		}
		spec.Ordering = orderSpec
	}

	// Parse conditions (properties and operators)
	// Split by And/Or - but only if they are logical operators, not part of operator names
	var conditionParts []string

	// Try to determine if And/Or is a logical operator or part of an operator name
	// We do this by trying to parse the whole thing as a single condition first
	singleCondition, err := p.parseCondition(remainder)
	if err == nil && strings.Contains(remainder, string(singleCondition.PropertyName)) {
		// Successfully parsed as single condition - use it
		conditionParts = []string{remainder}
	} else if strings.Contains(remainder, "And") && !strings.Contains(remainder, "LessThanAnd") && !strings.Contains(remainder, "GreaterThanAnd") {
		// Contains And but not as part of an operator name
		spec.LogicalOperator = LogicalAnd
		conditionParts = splitByLogicalOperator(remainder, "And")
	} else if strings.Contains(remainder, "Or") && !strings.Contains(remainder, "OrEqual") {
		// Contains Or but not as part of "OrEqual"
		spec.LogicalOperator = LogicalOr
		conditionParts = splitByLogicalOperator(remainder, "Or")
	} else {
		// Single condition
		conditionParts = []string{remainder}
	}

	// Parse each condition
	for _, part := range conditionParts {
		condition, err := p.parseCondition(part)
		if err != nil {
			return nil, fmt.Errorf("invalid condition '%s' in %s: %v", part, methodName, err)
		}
		spec.Conditions = append(spec.Conditions, condition)
	}

	return spec, nil
}

func (p *QueryParser) parseCondition(part string) (Condition, error) {
	condition := Condition{
		Operator: OpEqual, // Default operator
	}

	// Try to match operators (longest first)
	operatorMap := []struct {
		suffix   string
		operator Operator
	}{
		{"GreaterThanOrEqual", OpGreaterThanOrEqual},
		{"LessThanOrEqual", OpLessThanOrEqual},
		{"GreaterThan", OpGreaterThan},
		{"LessThan", OpLessThan},
		{"NotEqual", OpNotEqual},
		{"Between", OpBetween},
		{"NotIn", OpNotIn},
		{"In", OpIn},
		{"NotLike", OpNotLike},
		{"Like", OpLike},
		{"NotContains", OpNotContains},
		{"Contains", OpContains},
		{"StartsWith", OpStartsWith},
		{"EndsWith", OpEndsWith},
		{"IsNotNull", OpIsNotNull},
		{"IsNull", OpIsNull},
	}

	for _, op := range operatorMap {
		if strings.HasSuffix(part, op.suffix) {
			condition.Operator = op.operator
			condition.PropertyName = strings.TrimSuffix(part, op.suffix)
			if condition.PropertyName == "" {
				return condition, fmt.Errorf("missing property name before operator %s", op.suffix)
			}
			// Convert to lowercase first letter for property name
			condition.PropertyName = toLowerFirst(condition.PropertyName)
			return condition, nil
		}
	}

	// No operator suffix found, use property name as-is with Equal operator
	condition.PropertyName = toLowerFirst(part)
	return condition, nil
}

func (p *QueryParser) parseOrdering(orderPart string) ([]OrderClause, error) {
	if orderPart == "" {
		return nil, fmt.Errorf("empty ordering specification")
	}

	// Split by And for multiple order clauses
	parts := splitByLogicalOperator(orderPart, "And")
	var ordering []OrderClause

	for _, part := range parts {
		direction := SortAsc
		propName := part

		// Check for Desc/Asc suffix
		if strings.HasSuffix(part, "Desc") {
			direction = SortDesc
			propName = strings.TrimSuffix(part, "Desc")
		} else if strings.HasSuffix(part, "Asc") {
			propName = strings.TrimSuffix(part, "Asc")
		}

		if propName == "" {
			return nil, fmt.Errorf("empty property name in ordering")
		}

		ordering = append(ordering, OrderClause{
			PropertyName: toLowerFirst(propName),
			Direction:    direction,
		})
	}

	return ordering, nil
}

// splitByLogicalOperator splits a string by a logical operator (And/Or) but only at word boundaries
// and not when it's part of an operator name (like LessThanOrEqual)
func splitByLogicalOperator(s string, keyword string) []string {
	var parts []string
	current := ""
	i := 0

	for i < len(s) {
		if strings.HasPrefix(s[i:], keyword) {
			// Check if it's at a word boundary (next char is uppercase or end)
			// AND not preceded by "Than" (for LessThanAnd, GreaterThanAnd patterns)
			isWordBoundary := i+len(keyword) >= len(s) || isUpper(rune(s[i+len(keyword)]))
			isPartOfOperator := (keyword == "And" && strings.HasSuffix(current, "Than")) ||
				(keyword == "Or" && (i+len(keyword) < len(s) && s[i+len(keyword):i+len(keyword)+5] == "Equal"))

			if isWordBoundary && !isPartOfOperator {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
				i += len(keyword)
				continue
			}
		}
		current += string(s[i])
		i++
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// toLowerFirst converts the first character to lowercase
func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// isUpper checks if a rune is uppercase
func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// ClearCache clears the query spec cache
func (p *QueryParser) ClearCache() {
	p.cache.Range(func(key, value interface{}) bool {
		p.cache.Delete(key)
		return true
	})
}
