// Package repository provides generic repository interfaces and implementations.
package repository

import (
	"fmt"
)

// Repository defines a generic repository interface for CRUD operations.
type Repository[T any, ID comparable] interface {
	Find(id ID) (*T, error)
	FindAll() ([]T, error)
	Save(entity *T) error
	Delete(id ID) error
}

// InMemoryRepository provides a generic in-memory repository implementation.
type InMemoryRepository[T any, ID comparable] struct {
	data      map[ID]*T
	getID     func(*T) ID
	nextID    func() ID
	setID     func(*T, ID)
}

// NewInMemoryRepository creates a new in-memory repository.
func NewInMemoryRepository[T any, ID comparable](
	getID func(*T) ID,
	nextID func() ID,
	setID func(*T, ID),
) *InMemoryRepository[T, ID] {
	return &InMemoryRepository[T, ID]{
		data:   make(map[ID]*T),
		getID:  getID,
		nextID: nextID,
		setID:  setID,
	}
}

// Find retrieves an entity by ID.
func (r *InMemoryRepository[T, ID]) Find(id ID) (*T, error) {
	entity, exists := r.data[id]
	if !exists {
		return nil, fmt.Errorf("entity with id %v not found", id)
	}
	return entity, nil
}

// FindAll retrieves all entities.
func (r *InMemoryRepository[T, ID]) FindAll() ([]T, error) {
	result := make([]T, 0, len(r.data))
	for _, entity := range r.data {
		result = append(result, *entity)
	}
	return result, nil
}

// Save stores or updates an entity.
func (r *InMemoryRepository[T, ID]) Save(entity *T) error {
	id := r.getID(entity)
	var zero ID
	if id == zero {
		id = r.nextID()
		r.setID(entity, id)
	}
	r.data[id] = entity
	return nil
}

// Delete removes an entity by ID.
func (r *InMemoryRepository[T, ID]) Delete(id ID) error {
	if _, exists := r.data[id]; !exists {
		return fmt.Errorf("entity with id %v not found", id)
	}
	delete(r.data, id)
	return nil
}

// Filter applies a predicate to filter entities.
func (r *InMemoryRepository[T, ID]) Filter(predicate func(*T) bool) []T {
	result := make([]T, 0)
	for _, entity := range r.data {
		if predicate(entity) {
			result = append(result, *entity)
		}
	}
	return result
}
