package main

// Set is a collection of unique values.
type Set[T comparable] struct {
	elements map[T]bool
}

// NewSet creates a new empty set.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{elements: map[T]bool{}}
}

// NewSetFromSlice creates a new set from the value in the slice.
func NewSetFromSlice[T comparable](slice []T) *Set[T] {
	s := NewSet[T]()
	for _, v := range slice {
		s.Add(v)
	}
	return s
}

// Add appends new items to the set if they do not already exist.
func (s *Set[T]) Add(values ...T) {
	for _, v := range values {
		s.elements[v] = true
	}
}

// Delete removes an item from the set.
func (s *Set[T]) Delete(v T) {
	delete(s.elements, v)
}

// Has checks if a value is present in the set.
func (s *Set[T]) Has(v T) bool {
	_, ok := s.elements[v]
	return ok
}

// Clear deletes all items from the set.
func (s *Set[T]) Clear() {
	s.elements = map[T]bool{}
}

// Size returns the number of unique values inside the set.
func (s *Set[T]) Size() int {
	return len(s.elements)
}

// Values returns an array of all values in the set.
func (s *Set[T]) Values() []T {
	values := make([]T, len(s.elements))
	i := 0
	for k := range s.elements {
		values[i] = k
		i++
	}
	return values
}
