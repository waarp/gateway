package ordered

import (
	"iter"
	"slices"
)

type mapElem[K comparable, V any] struct {
	Key K
	Val V
}

type Map[K comparable, V any] []*mapElem[K, V]

func (o *Map[K, V]) Get(key K) (V, bool) {
	if o == nil {
		return *new(V), false
	}

	for _, elem := range *o {
		if elem.Key == key {
			return elem.Val, true
		}
	}

	return *new(V), false
}

func (o *Map[K, V]) Add(key K, val V) {
	*o = append(*o, &mapElem[K, V]{Key: key, Val: val})
}

func (o *Map[K, V]) Delete(key K) {
	if o == nil {
		return
	}

	for i, elem := range *o {
		if elem.Key == key {
			*o = slices.Delete(*o, i, i)
		}
	}
}

func (o *Map[K, V]) Keys() []K {
	if o == nil {
		return nil
	}

	keys := make([]K, len(*o))
	for i, elem := range *o {
		keys[i] = elem.Key
	}

	return keys
}

func (o *Map[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, elem := range *o {
			if !yield(elem.Key, elem.Val) {
				return
			}
		}
	}
}
