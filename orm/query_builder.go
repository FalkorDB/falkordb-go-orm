package orm

import (
	"fmt"
	"strings"
)

// QueryBuilder generates Cypher queries for various operations
type QueryBuilder struct{}

// NewQueryBuilder creates a new query builder instance
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// BuildMatchByID generates a MATCH query to find an entity by ID
func (b *QueryBuilder) BuildMatchByID(metadata *EntityMetadata, id interface{}) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) WHERE id(n) = $id RETURN n", labelsStr)
	params := map[string]interface{}{"id": id}
	return cypher, params
}

// BuildMatchAll generates a MATCH query to find all entities
func (b *QueryBuilder) BuildMatchAll(metadata *EntityMetadata) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) RETURN n", labelsStr)
	params := map[string]interface{}{}
	return cypher, params
}

// BuildCount generates a COUNT query
func (b *QueryBuilder) BuildCount(metadata *EntityMetadata) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as count", labelsStr)
	params := map[string]interface{}{}
	return cypher, params
}

// BuildExistsByID generates an EXISTS query by ID
func (b *QueryBuilder) BuildExistsByID(metadata *EntityMetadata, id interface{}) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) WHERE id(n) = $id RETURN count(n) > 0 as exists", labelsStr)
	params := map[string]interface{}{"id": id}
	return cypher, params
}

// BuildDeleteByID generates a DELETE query by ID
func (b *QueryBuilder) BuildDeleteByID(metadata *EntityMetadata, id interface{}) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) WHERE id(n) = $id DELETE n", labelsStr)
	params := map[string]interface{}{"id": id}
	return cypher, params
}

// BuildDeleteAll generates a DELETE query for all entities
func (b *QueryBuilder) BuildDeleteAll(metadata *EntityMetadata) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) DELETE n", labelsStr)
	params := map[string]interface{}{}
	return cypher, params
}

// BuildMatchByIDs generates a MATCH query to find entities by multiple IDs
func (b *QueryBuilder) BuildMatchByIDs(metadata *EntityMetadata, ids []interface{}) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) WHERE id(n) IN $ids RETURN n", labelsStr)
	params := map[string]interface{}{"ids": ids}
	return cypher, params
}

// BuildMatchWithCondition generates a MATCH query with a WHERE clause
func (b *QueryBuilder) BuildMatchWithCondition(metadata *EntityMetadata, property string, operator string, value interface{}) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	cypher := fmt.Sprintf("MATCH (n:%s) WHERE n.%s %s $value RETURN n", labelsStr, property, operator)
	params := map[string]interface{}{"value": value}
	return cypher, params
}

// BuildMatchWithConditions generates a MATCH query with multiple WHERE conditions
func (b *QueryBuilder) BuildMatchWithConditions(metadata *EntityMetadata, conditions []WhereCondition, logicalOp string) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	params := make(map[string]interface{})

	whereClauses := make([]string, 0, len(conditions))
	for i, cond := range conditions {
		paramName := fmt.Sprintf("param_%d", i)
		clause := fmt.Sprintf("n.%s %s $%s", cond.Property, cond.Operator, paramName)
		whereClauses = append(whereClauses, clause)
		params[paramName] = cond.Value
	}

	whereStr := strings.Join(whereClauses, fmt.Sprintf(" %s ", logicalOp))
	cypher := fmt.Sprintf("MATCH (n:%s) WHERE %s RETURN n", labelsStr, whereStr)
	return cypher, params
}

// BuildFromQuerySpec builds a Cypher query from a QuerySpec
func (qb *QueryBuilder) BuildFromQuerySpec(spec *QuerySpec, metadata *EntityMetadata, args []interface{}) (string, map[string]interface{}) {
	params := make(map[string]interface{})
	labels := strings.Join(metadata.Labels, ":")
	paramIndex := 0

	// Build WHERE clause
	var whereClauses []string
	for _, condition := range spec.Conditions {
		clause, condParams := qb.buildCondition(condition, &paramIndex, args)
		for k, v := range condParams {
			params[k] = v
		}
		whereClauses = append(whereClauses, clause)
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		separator := " AND "
		if spec.LogicalOperator == LogicalOr {
			separator = " OR "
		}
		whereClause = " WHERE " + strings.Join(whereClauses, separator)
	}

	// Build ORDER BY clause
	orderByClause := ""
	if len(spec.Ordering) > 0 {
		var orderParts []string
		for _, order := range spec.Ordering {
			orderParts = append(orderParts, fmt.Sprintf("n.%s %s", order.PropertyName, order.Direction))
		}
		orderByClause = " ORDER BY " + strings.Join(orderParts, ", ")
	}

	// Build query based on operation
	var cypher string
	switch spec.Operation {
	case OpFind:
		cypher = fmt.Sprintf("MATCH (n:%s)%s RETURN n%s", labels, whereClause, orderByClause)
	case OpCount:
		cypher = fmt.Sprintf("MATCH (n:%s)%s RETURN count(n)", labels, whereClause)
	case OpExists:
		cypher = fmt.Sprintf("MATCH (n:%s)%s RETURN count(n) > 0", labels, whereClause)
	case OpDelete:
		cypher = fmt.Sprintf("MATCH (n:%s)%s DELETE n", labels, whereClause)
	}

	if spec.Limit != nil && spec.Operation == OpFind {
		cypher += fmt.Sprintf(" LIMIT %d", *spec.Limit)
	}

	return cypher, params
}

// buildCondition builds a WHERE condition from a Condition spec
func (qb *QueryBuilder) buildCondition(condition Condition, paramIndex *int, args []interface{}) (string, map[string]interface{}) {
	params := make(map[string]interface{})
	propName := condition.PropertyName

	switch condition.Operator {
	case OpEqual:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s = $%s", propName, paramName), params

	case OpNotEqual:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s <> $%s", propName, paramName), params

	case OpGreaterThan:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s > $%s", propName, paramName), params

	case OpGreaterThanOrEqual:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s >= $%s", propName, paramName), params

	case OpLessThan:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s < $%s", propName, paramName), params

	case OpLessThanOrEqual:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s <= $%s", propName, paramName), params

	case OpBetween:
		paramName1 := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName1] = args[*paramIndex]
		*paramIndex++
		paramName2 := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName2] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s >= $%s AND n.%s <= $%s", propName, paramName1, propName, paramName2), params

	case OpIn:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s IN $%s", propName, paramName), params

	case OpNotIn:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("NOT n.%s IN $%s", propName, paramName), params

	case OpLike:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s =~ $%s", propName, paramName), params

	case OpNotLike:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("NOT n.%s =~ $%s", propName, paramName), params

	case OpContains:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s CONTAINS $%s", propName, paramName), params

	case OpNotContains:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("NOT n.%s CONTAINS $%s", propName, paramName), params

	case OpStartsWith:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s STARTS WITH $%s", propName, paramName), params

	case OpEndsWith:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s ENDS WITH $%s", propName, paramName), params

	case OpIsNull:
		return fmt.Sprintf("n.%s IS NULL", propName), params

	case OpIsNotNull:
		return fmt.Sprintf("n.%s IS NOT NULL", propName), params

	default:
		paramName := fmt.Sprintf("param_%d", *paramIndex)
		params[paramName] = args[*paramIndex]
		*paramIndex++
		return fmt.Sprintf("n.%s = $%s", propName, paramName), params
	}
}

// BuildMatchWithPagination generates a MATCH query with SKIP and LIMIT
func (b *QueryBuilder) BuildMatchWithPagination(metadata *EntityMetadata, page, size int, sortBy, direction string) (string, map[string]interface{}) {
	labelsStr := metadata.PrimaryLabel
	skip := page * size

	var cypher string
	if sortBy != "" {
		dir := "ASC"
		if strings.ToUpper(direction) == "DESC" {
			dir = "DESC"
		}
		cypher = fmt.Sprintf("MATCH (n:%s) RETURN n ORDER BY n.%s %s SKIP %d LIMIT %d",
			labelsStr, sortBy, dir, skip, size)
	} else {
		cypher = fmt.Sprintf("MATCH (n:%s) RETURN n SKIP %d LIMIT %d",
			labelsStr, skip, size)
	}

	params := map[string]interface{}{}
	return cypher, params
}

// WhereCondition represents a WHERE clause condition for QueryBuilderChain
type WhereCondition struct {
	Property string
	Operator string
	Value    interface{}
}

// QueryBuilderChain provides a fluent API for building queries
type QueryBuilderChain struct {
	metadata   *EntityMetadata
	conditions []WhereCondition
	logicalOp  string
	orderBy    []OrderBy
	limit      int
	skip       int
}

// OrderBy represents an ORDER BY clause
type OrderBy struct {
	Property  string
	Direction string // ASC or DESC
}

// NewQueryBuilderChain creates a new query builder chain
func NewQueryBuilderChain(metadata *EntityMetadata) *QueryBuilderChain {
	return &QueryBuilderChain{
		metadata:   metadata,
		conditions: make([]WhereCondition, 0),
		logicalOp:  "AND",
		orderBy:    make([]OrderBy, 0),
	}
}

// Where adds a condition to the query
func (q *QueryBuilderChain) Where(property, operator string, value interface{}) *QueryBuilderChain {
	q.conditions = append(q.conditions, WhereCondition{
		Property: property,
		Operator: operator,
		Value:    value,
	})
	return q
}

// And sets the logical operator to AND (default)
func (q *QueryBuilderChain) And() *QueryBuilderChain {
	q.logicalOp = "AND"
	return q
}

// Or sets the logical operator to OR
func (q *QueryBuilderChain) Or() *QueryBuilderChain {
	q.logicalOp = "OR"
	return q
}

// OrderByAsc adds an ascending order clause
func (q *QueryBuilderChain) OrderByAsc(property string) *QueryBuilderChain {
	q.orderBy = append(q.orderBy, OrderBy{Property: property, Direction: "ASC"})
	return q
}

// OrderByDesc adds a descending order clause
func (q *QueryBuilderChain) OrderByDesc(property string) *QueryBuilderChain {
	q.orderBy = append(q.orderBy, OrderBy{Property: property, Direction: "DESC"})
	return q
}

// Limit sets the result limit
func (q *QueryBuilderChain) Limit(limit int) *QueryBuilderChain {
	q.limit = limit
	return q
}

// Skip sets the number of results to skip
func (q *QueryBuilderChain) Skip(skip int) *QueryBuilderChain {
	q.skip = skip
	return q
}

// Build generates the final Cypher query
func (q *QueryBuilderChain) Build() (string, map[string]interface{}) {
	labelsStr := q.metadata.PrimaryLabel
	params := make(map[string]interface{})

	// Start with MATCH
	parts := []string{fmt.Sprintf("MATCH (n:%s)", labelsStr)}

	// Add WHERE clause if conditions exist
	if len(q.conditions) > 0 {
		whereClauses := make([]string, 0, len(q.conditions))
		for i, cond := range q.conditions {
			paramName := fmt.Sprintf("param_%d", i)
			clause := fmt.Sprintf("n.%s %s $%s", cond.Property, cond.Operator, paramName)
			whereClauses = append(whereClauses, clause)
			params[paramName] = cond.Value
		}
		whereStr := strings.Join(whereClauses, fmt.Sprintf(" %s ", q.logicalOp))
		parts = append(parts, "WHERE", whereStr)
	}

	// Add RETURN
	parts = append(parts, "RETURN n")

	// Add ORDER BY if specified
	if len(q.orderBy) > 0 {
		orderClauses := make([]string, 0, len(q.orderBy))
		for _, order := range q.orderBy {
			orderClauses = append(orderClauses, fmt.Sprintf("n.%s %s", order.Property, order.Direction))
		}
		parts = append(parts, "ORDER BY", strings.Join(orderClauses, ", "))
	}

	// Add SKIP if specified
	if q.skip > 0 {
		parts = append(parts, fmt.Sprintf("SKIP %d", q.skip))
	}

	// Add LIMIT if specified
	if q.limit > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT %d", q.limit))
	}

	cypher := strings.Join(parts, " ")
	return cypher, params
}

// BuildRelationshipQuery builds a query to load related entities
func (b *QueryBuilder) BuildRelationshipQuery(relMeta *RelationshipMetadata, sourceID int64) (string, map[string]interface{}) {
	params := map[string]interface{}{"source_id": sourceID}

	// Build relationship pattern based on direction
	var pattern string
	switch relMeta.Direction {
	case DirectionOutgoing:
		pattern = fmt.Sprintf("(source)-[:%s]->(target)", relMeta.RelationshipType)
	case DirectionIncoming:
		pattern = fmt.Sprintf("(source)<-[:%s]-(target)", relMeta.RelationshipType)
	case DirectionBoth:
		pattern = fmt.Sprintf("(source)-[:%s]-(target)", relMeta.RelationshipType)
	default:
		pattern = fmt.Sprintf("(source)-[:%s]->(target)", relMeta.RelationshipType)
	}

	cypher := fmt.Sprintf("MATCH %s WHERE id(source) = $source_id RETURN target", pattern)
	return cypher, params
}

// BuildCreateRelationship builds a query to create a relationship
func (b *QueryBuilder) BuildCreateRelationship(sourceID, targetID int64, relMeta *RelationshipMetadata) (string, map[string]interface{}) {
	params := map[string]interface{}{
		"source_id": sourceID,
		"target_id": targetID,
	}

	// Build relationship pattern based on direction
	var pattern string
	switch relMeta.Direction {
	case DirectionOutgoing:
		pattern = fmt.Sprintf("(source)-[:%s]->(target)", relMeta.RelationshipType)
	case DirectionIncoming:
		pattern = fmt.Sprintf("(source)<-[:%s]-(target)", relMeta.RelationshipType)
	case DirectionBoth:
		pattern = fmt.Sprintf("(source)-[:%s]-(target)", relMeta.RelationshipType)
	default:
		pattern = fmt.Sprintf("(source)-[:%s]->(target)", relMeta.RelationshipType)
	}

	cypher := fmt.Sprintf("MATCH (source) WHERE id(source) = $source_id MATCH (target) WHERE id(target) = $target_id CREATE %s", pattern)
	return cypher, params
}

// BuildDeleteRelationships builds a query to delete relationships
func (b *QueryBuilder) BuildDeleteRelationships(sourceID int64, relMeta *RelationshipMetadata) (string, map[string]interface{}) {
	params := map[string]interface{}{"source_id": sourceID}

	// Build relationship pattern based on direction
	var pattern string
	switch relMeta.Direction {
	case DirectionOutgoing:
		pattern = fmt.Sprintf("(source)-[r:%s]->()", relMeta.RelationshipType)
	case DirectionIncoming:
		pattern = fmt.Sprintf("(source)<-[r:%s]-()", relMeta.RelationshipType)
	case DirectionBoth:
		pattern = fmt.Sprintf("(source)-[r:%s]-()", relMeta.RelationshipType)
	default:
		pattern = fmt.Sprintf("(source)-[r:%s]->()", relMeta.RelationshipType)
	}

	cypher := fmt.Sprintf("MATCH %s WHERE id(source) = $source_id DELETE r", pattern)
	return cypher, params
}

// BuildEagerLoading builds a query with OPTIONAL MATCH for eager loading relationships
func (b *QueryBuilder) BuildEagerLoading(metadata *EntityMetadata, id interface{}, fetch []string) (string, map[string]interface{}) {
	params := map[string]interface{}{"id": id}
	labelsStr := strings.Join(metadata.Labels, ":")

	// Start with base MATCH
	parts := []string{
		fmt.Sprintf("MATCH (n:%s) WHERE id(n) = $id", labelsStr),
	}

	// Add OPTIONAL MATCH for each requested relationship
	returns := []string{"n"}
	for _, relName := range fetch {
		relMeta := metadata.GetRelationshipByName(relName)
		if relMeta == nil {
			continue
		}

		// Build relationship pattern
		var pattern string
		switch relMeta.Direction {
		case DirectionOutgoing:
			pattern = fmt.Sprintf("(n)-[:%s]->(rel_%s)", relMeta.RelationshipType, relName)
		case DirectionIncoming:
			pattern = fmt.Sprintf("(n)<-[:%s]-(rel_%s)", relMeta.RelationshipType, relName)
		case DirectionBoth:
			pattern = fmt.Sprintf("(n)-[:%s]-(rel_%s)", relMeta.RelationshipType, relName)
		default:
			pattern = fmt.Sprintf("(n)-[:%s]->(rel_%s)", relMeta.RelationshipType, relName)
		}

		parts = append(parts, fmt.Sprintf("OPTIONAL MATCH %s", pattern))

		if relMeta.IsList {
			returns = append(returns, fmt.Sprintf("collect(distinct rel_%s) as %s", relName, relName))
		} else {
			returns = append(returns, fmt.Sprintf("rel_%s as %s", relName, relName))
		}
	}

	parts = append(parts, "RETURN "+strings.Join(returns, ", "))
	cypher := strings.Join(parts, " ")
	return cypher, params
}
