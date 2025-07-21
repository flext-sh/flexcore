// Package patterns provides a SQL query builder pattern
package patterns

import (
	"fmt"
	"strings"
)

// QueryBuilder demonstrates the builder pattern for SQL queries
type QueryBuilder struct {
	table   string
	columns []string
	where   []WhereClause
	orderBy []OrderClause
	limit   int
	offset  int
	joins   []JoinClause
	groupBy []string
	having  []HavingClause
}

type WhereClause struct {
	Column   string
	Operator string
	Value    interface{}
}

type OrderClause struct {
	Column string
	Desc   bool
}

type JoinClause struct {
	Type      string // INNER, LEFT, RIGHT, FULL
	Table     string
	Condition string
}

type HavingClause struct {
	Condition string
	Value     interface{}
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:   table,
		columns: []string{"*"},
		where:   make([]WhereClause, 0),
		orderBy: make([]OrderClause, 0),
		joins:   make([]JoinClause, 0),
		groupBy: make([]string, 0),
		having:  make([]HavingClause, 0),
	}
}

func (q *QueryBuilder) Select(columns ...string) *QueryBuilder {
	q.columns = columns
	return q
}

func (q *QueryBuilder) Where(column, operator string, value interface{}) *QueryBuilder {
	q.where = append(q.where, WhereClause{column, operator, value})
	return q
}

func (q *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	q.where = append(q.where, WhereClause{column, "IN", values})
	return q
}

func (q *QueryBuilder) OrderBy(column string, desc bool) *QueryBuilder {
	q.orderBy = append(q.orderBy, OrderClause{column, desc})
	return q
}

func (q *QueryBuilder) Limit(limit int) *QueryBuilder {
	q.limit = limit
	return q
}

func (q *QueryBuilder) Offset(offset int) *QueryBuilder {
	q.offset = offset
	return q
}

func (q *QueryBuilder) Join(table, condition string) *QueryBuilder {
	q.joins = append(q.joins, JoinClause{"INNER", table, condition})
	return q
}

func (q *QueryBuilder) LeftJoin(table, condition string) *QueryBuilder {
	q.joins = append(q.joins, JoinClause{"LEFT", table, condition})
	return q
}

func (q *QueryBuilder) RightJoin(table, condition string) *QueryBuilder {
	q.joins = append(q.joins, JoinClause{"RIGHT", table, condition})
	return q
}

func (q *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	q.groupBy = append(q.groupBy, columns...)
	return q
}

func (q *QueryBuilder) Having(condition string, value interface{}) *QueryBuilder {
	q.having = append(q.having, HavingClause{condition, value})
	return q
}

func (q *QueryBuilder) Build() (string, []interface{}) {
	// Build SQL query
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(q.columns, ", "), q.table)

	var args []interface{}
	argIndex := 1

	// Add JOINs
	for _, join := range q.joins {
		query += fmt.Sprintf(" %s JOIN %s ON %s", join.Type, join.Table, join.Condition)
	}

	// Add WHERE clauses
	if len(q.where) > 0 {
		whereClauses := make([]string, 0, len(q.where))
		for _, clause := range q.where {
			if clause.Operator == "IN" {
				values, ok := clause.Value.([]interface{})
				if !ok {
					// Log error and skip this clause
					continue
				}
				placeholders := make([]string, len(values))
				for i, v := range values {
					placeholders[i] = fmt.Sprintf("$%d", argIndex)
					args = append(args, v)
					argIndex++
				}
				whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)",
					clause.Column, strings.Join(placeholders, ", ")))
			} else {
				whereClauses = append(whereClauses, fmt.Sprintf("%s %s $%d",
					clause.Column, clause.Operator, argIndex))
				args = append(args, clause.Value)
				argIndex++
			}
		}
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add GROUP BY
	if len(q.groupBy) > 0 {
		query += " GROUP BY " + strings.Join(q.groupBy, ", ")
	}

	// Add HAVING
	if len(q.having) > 0 {
		havingClauses := make([]string, len(q.having))
		for i, clause := range q.having {
			havingClauses[i] = fmt.Sprintf("%s $%d", clause.Condition, argIndex)
			args = append(args, clause.Value)
			argIndex++
		}
		query += " HAVING " + strings.Join(havingClauses, " AND ")
	}

	// Add ORDER BY
	if len(q.orderBy) > 0 {
		orderClauses := make([]string, len(q.orderBy))
		for i, clause := range q.orderBy {
			dir := "ASC"
			if clause.Desc {
				dir = "DESC"
			}
			orderClauses[i] = fmt.Sprintf("%s %s", clause.Column, dir)
		}
		query += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	// Add LIMIT and OFFSET
	if q.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.limit)
	}
	if q.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", q.offset)
	}

	return query, args
}

// Clone creates a copy of the query builder
func (q *QueryBuilder) Clone() *QueryBuilder {
	clone := &QueryBuilder{
		table:   q.table,
		columns: make([]string, len(q.columns)),
		where:   make([]WhereClause, len(q.where)),
		orderBy: make([]OrderClause, len(q.orderBy)),
		limit:   q.limit,
		offset:  q.offset,
		joins:   make([]JoinClause, len(q.joins)),
		groupBy: make([]string, len(q.groupBy)),
		having:  make([]HavingClause, len(q.having)),
	}

	copy(clone.columns, q.columns)
	copy(clone.where, q.where)
	copy(clone.orderBy, q.orderBy)
	copy(clone.joins, q.joins)
	copy(clone.groupBy, q.groupBy)
	copy(clone.having, q.having)

	return clone
}
