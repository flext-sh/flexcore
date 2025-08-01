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

// BuildContext holds the state during query building - reduces parameter passing
type BuildContext struct {
	args     []interface{}
	argIndex int
}

// ClauseBuilder defines strategy interface for building different SQL clauses
type ClauseBuilder interface {
	buildClause(ctx *BuildContext) string
}

// JoinClauseBuilder builds JOIN clauses - Strategy pattern
type JoinClauseBuilder struct {
	joins []JoinClause
}

func (j *JoinClauseBuilder) buildClause(ctx *BuildContext) string {
	if len(j.joins) == 0 {
		return ""
	}
	var parts []string
	for _, join := range j.joins {
		parts = append(parts, fmt.Sprintf(" %s JOIN %s ON %s", join.Type, join.Table, join.Condition))
	}
	return strings.Join(parts, "")
}

// WhereClauseBuilder builds WHERE clauses - Strategy pattern
type WhereClauseBuilder struct {
	where []WhereClause
}

func (w *WhereClauseBuilder) buildClause(ctx *BuildContext) string {
	if len(w.where) == 0 {
		return ""
	}
	whereClauses := make([]string, 0, len(w.where))
	for _, clause := range w.where {
		if clause.Operator == "IN" {
			if inClause := w.buildInClause(clause, ctx); inClause != "" {
				whereClauses = append(whereClauses, inClause)
			}
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s %s $%d",
				clause.Column, clause.Operator, ctx.argIndex))
			ctx.args = append(ctx.args, clause.Value)
			ctx.argIndex++
		}
	}
	return " WHERE " + strings.Join(whereClauses, " AND ")
}

func (w *WhereClauseBuilder) buildInClause(clause WhereClause, ctx *BuildContext) string {
	values, ok := clause.Value.([]interface{})
	if !ok {
		return "" // Skip invalid IN clause
	}
	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = fmt.Sprintf("$%d", ctx.argIndex)
		ctx.args = append(ctx.args, v)
		ctx.argIndex++
	}
	return fmt.Sprintf("%s IN (%s)", clause.Column, strings.Join(placeholders, ", "))
}

// HavingClauseBuilder builds HAVING clauses - Strategy pattern
type HavingClauseBuilder struct {
	having []HavingClause
}

func (h *HavingClauseBuilder) buildClause(ctx *BuildContext) string {
	if len(h.having) == 0 {
		return ""
	}
	havingClauses := make([]string, len(h.having))
	for i, clause := range h.having {
		havingClauses[i] = fmt.Sprintf("%s $%d", clause.Condition, ctx.argIndex)
		ctx.args = append(ctx.args, clause.Value)
		ctx.argIndex++
	}
	return " HAVING " + strings.Join(havingClauses, " AND ")
}

// OrderByClauseBuilder builds ORDER BY clauses - Strategy pattern
type OrderByClauseBuilder struct {
	orderBy []OrderClause
}

func (o *OrderByClauseBuilder) buildClause(ctx *BuildContext) string {
	if len(o.orderBy) == 0 {
		return ""
	}
	orderClauses := make([]string, len(o.orderBy))
	for i, clause := range o.orderBy {
		dir := "ASC"
		if clause.Desc {
			dir = "DESC"
		}
		orderClauses[i] = fmt.Sprintf("%s %s", clause.Column, dir)
	}
	return " ORDER BY " + strings.Join(orderClauses, ", ")
}

// SimpleClauseBuilder builds simple clauses like GROUP BY - Strategy pattern
type SimpleClauseBuilder struct {
	clauseName string
	columns    []string
}

func (s *SimpleClauseBuilder) buildClause(ctx *BuildContext) string {
	if len(s.columns) == 0 {
		return ""
	}
	return fmt.Sprintf(" %s %s", s.clauseName, strings.Join(s.columns, ", "))
}

// LimitOffsetBuilder builds LIMIT and OFFSET clauses - Strategy pattern
type LimitOffsetBuilder struct {
	limit  int
	offset int
}

func (l *LimitOffsetBuilder) buildClause(ctx *BuildContext) string {
	var parts []string
	if l.limit > 0 {
		parts = append(parts, fmt.Sprintf(" LIMIT %d", l.limit))
	}
	if l.offset > 0 {
		parts = append(parts, fmt.Sprintf(" OFFSET %d", l.offset))
	}
	return strings.Join(parts, "")
}

// Build constructs the SQL query using Strategy pattern - reduced complexity from 28 to ~8
func (q *QueryBuilder) Build() (string, []interface{}) {
	// Initialize build context
	ctx := &BuildContext{
		args:     make([]interface{}, 0),
		argIndex: 1,
	}

	// Build base SELECT statement
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(q.columns, ", "), q.table)

	// Apply clause builders using Strategy pattern - reduces complexity
	builders := []ClauseBuilder{
		&JoinClauseBuilder{joins: q.joins},
		&WhereClauseBuilder{where: q.where},
		&SimpleClauseBuilder{clauseName: "GROUP BY", columns: q.groupBy},
		&HavingClauseBuilder{having: q.having},
		&OrderByClauseBuilder{orderBy: q.orderBy},
		&LimitOffsetBuilder{limit: q.limit, offset: q.offset},
	}

	// Execute all clause builders - single loop reduces complexity
	for _, builder := range builders {
		if clauseSQL := builder.buildClause(ctx); clauseSQL != "" {
			query += clauseSQL
		}
	}

	return query, ctx.args
}

// Clone creates a copy of the query builder - refactored using helper functions
func (q *QueryBuilder) Clone() *QueryBuilder {
	clone := &QueryBuilder{
		table:  q.table,
		limit:  q.limit,
		offset: q.offset,
	}

	// Use helper functions to reduce complexity
	clone.columns = q.cloneStringSlice(q.columns)
	clone.groupBy = q.cloneStringSlice(q.groupBy)
	clone.where = q.cloneWhereSlice(q.where)
	clone.orderBy = q.cloneOrderSlice(q.orderBy)
	clone.joins = q.cloneJoinSlice(q.joins)
	clone.having = q.cloneHavingSlice(q.having)

	return clone
}

// Helper functions for Clone - DRY principle
func (q *QueryBuilder) cloneStringSlice(src []string) []string {
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func (q *QueryBuilder) cloneWhereSlice(src []WhereClause) []WhereClause {
	dst := make([]WhereClause, len(src))
	copy(dst, src)
	return dst
}

func (q *QueryBuilder) cloneOrderSlice(src []OrderClause) []OrderClause {
	dst := make([]OrderClause, len(src))
	copy(dst, src)
	return dst
}

func (q *QueryBuilder) cloneJoinSlice(src []JoinClause) []JoinClause {
	dst := make([]JoinClause, len(src))
	copy(dst, src)
	return dst
}

func (q *QueryBuilder) cloneHavingSlice(src []HavingClause) []HavingClause {
	dst := make([]HavingClause, len(src))
	copy(dst, src)
	return dst
}
