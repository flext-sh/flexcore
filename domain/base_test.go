// Package domain tests
package domain_test

import (
	"testing"
	"time"

	"github.com/flext/flexcore/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewEntity(t *testing.T) {
	id := "test-id-123"
	entity := domain.NewEntity(id)
	
	assert.Equal(t, id, entity.ID)
	assert.NotZero(t, entity.CreatedAt)
	assert.NotZero(t, entity.UpdatedAt)
	assert.Equal(t, int64(1), entity.Version)
	assert.Equal(t, entity.CreatedAt, entity.UpdatedAt)
}

func TestEntityTouch(t *testing.T) {
	entity := domain.NewEntity("test-id")
	originalUpdatedAt := entity.UpdatedAt
	originalVersion := entity.Version
	
	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)
	
	entity.Touch()
	
	assert.Equal(t, "test-id", entity.ID)
	assert.NotEqual(t, originalUpdatedAt, entity.UpdatedAt)
	assert.Equal(t, originalVersion+1, entity.Version)
	assert.True(t, entity.UpdatedAt.After(originalUpdatedAt))
}

func TestEntityEquals(t *testing.T) {
	entity1 := domain.NewEntity("id-1")
	entity2 := domain.NewEntity("id-2")
	entity3 := domain.NewEntity("id-1")
	
	assert.True(t, entity1.Equals(entity3))
	assert.False(t, entity1.Equals(entity2))
}

func TestNewAggregateRoot(t *testing.T) {
	id := "aggregate-id"
	ar := domain.NewAggregateRoot(id)
	
	assert.Equal(t, id, ar.Entity.ID)
	assert.NotNil(t, ar.DomainEvents())
	assert.Empty(t, ar.DomainEvents())
}

func TestAggregateRootRaiseEvent(t *testing.T) {
	ar := domain.NewAggregateRoot("test-id")
	
	event1 := &testDomainEvent{id: "event-1"}
	event2 := &testDomainEvent{id: "event-2"}
	
	ar.RaiseEvent(event1)
	ar.RaiseEvent(event2)
	
	events := ar.DomainEvents()
	assert.Len(t, events, 2)
	assert.Equal(t, event1, events[0])
	assert.Equal(t, event2, events[1])
}

func TestAggregateRootClearEvents(t *testing.T) {
	ar := domain.NewAggregateRoot("test-id")
	
	ar.RaiseEvent(&testDomainEvent{id: "event-1"})
	assert.Len(t, ar.DomainEvents(), 1)
	
	ar.ClearEvents()
	assert.Empty(t, ar.DomainEvents())
}

func TestBaseDomainEvent(t *testing.T) {
	event := domain.NewBaseDomainEvent("TestEvent", "aggregate-123")
	
	assert.NotEmpty(t, event.EventID())
	assert.Equal(t, "TestEvent", event.EventType())
	assert.Equal(t, "aggregate-123", event.AggregateID())
	assert.NotZero(t, event.OccurredAt())
	assert.True(t, event.OccurredAt().Before(time.Now().Add(time.Second)))
}

func TestBaseSpecification(t *testing.T) {
	// Create specifications
	greaterThan10 := domain.NewSpecification(func(n int) bool {
		return n > 10
	})
	
	lessThan100 := domain.NewSpecification(func(n int) bool {
		return n < 100
	})
	
	// Test individual specifications
	assert.True(t, greaterThan10.IsSatisfiedBy(20))
	assert.False(t, greaterThan10.IsSatisfiedBy(5))
	assert.True(t, lessThan100.IsSatisfiedBy(50))
	assert.False(t, lessThan100.IsSatisfiedBy(150))
	
	// Test And
	between10And100 := greaterThan10.And(lessThan100)
	assert.True(t, between10And100.IsSatisfiedBy(50))
	assert.False(t, between10And100.IsSatisfiedBy(5))
	assert.False(t, between10And100.IsSatisfiedBy(150))
	
	// Test Or
	notBetween10And100 := greaterThan10.Or(lessThan100)
	assert.True(t, notBetween10And100.IsSatisfiedBy(5))
	assert.True(t, notBetween10And100.IsSatisfiedBy(50))
	assert.True(t, notBetween10And100.IsSatisfiedBy(150))
	
	// Test Not
	notGreaterThan10 := greaterThan10.Not()
	assert.False(t, notGreaterThan10.IsSatisfiedBy(20))
	assert.True(t, notGreaterThan10.IsSatisfiedBy(5))
}

func TestBaseDomainService(t *testing.T) {
	service := domain.NewDomainService("UserService")
	assert.Equal(t, "UserService", service.Name())
}

// Test helpers

type testDomainEvent struct {
	id          string
	timestamp   time.Time
	aggregateId string
}

func (e *testDomainEvent) EventID() string {
	return e.id
}

func (e *testDomainEvent) EventType() string {
	return "TestEvent"
}

func (e *testDomainEvent) OccurredAt() time.Time {
	if e.timestamp.IsZero() {
		return time.Now()
	}
	return e.timestamp
}

func (e *testDomainEvent) AggregateID() string {
	return e.aggregateId
}

// Benchmarks

func BenchmarkNewEntity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = domain.NewEntity("test-id")
	}
}

func BenchmarkEntityTouch(b *testing.B) {
	entity := domain.NewEntity("test-id")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entity.Touch()
	}
}

func BenchmarkAggregateRootRaiseEvent(b *testing.B) {
	ar := domain.NewAggregateRoot("test-id")
	event := &testDomainEvent{id: "event-1"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ar.RaiseEvent(event)
		if i%100 == 0 {
			ar.ClearEvents() // Prevent unbounded growth
		}
	}
}

func BenchmarkSpecificationAnd(b *testing.B) {
	spec1 := domain.NewSpecification(func(n int) bool { return n > 10 })
	spec2 := domain.NewSpecification(func(n int) bool { return n < 100 })
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combined := spec1.And(spec2)
		_ = combined.IsSatisfiedBy(50)
	}
}