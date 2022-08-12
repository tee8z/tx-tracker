package utils

import (
	"sync"
)

type Set[T comparable] struct {
	mutex sync.RWMutex
	dic   map[T]string
}

func NewSet[T comparable]() *Set[T] {
	set := &Set[T]{}
	set.dic = make(map[T]string)
	return set
}

func (s *Set[T]) Add(value T, timestamp string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.dic[value] = timestamp
}

func (s *Set[T]) Remove(value T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.dic, value)
}

func (s *Set[T]) Keys() []T {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	keys := make([]T, 0)
	for key := range s.dic {
		keys = append(keys, key)
	}
	return keys
}

func (s *Set[T]) Get(value T) *string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	val, contains := s.dic[value]
	if contains {
		return &val
	}
	return nil
}

func (s *Set[T]) Contains(value T) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	_, c := s.dic[value]
	return c
}
