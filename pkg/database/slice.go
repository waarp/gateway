package database

import (
	"fmt"
	"strings"
)

type Slice[T Table] []T

func (*Slice[T]) TableName() string { return (*new(T)).TableName() }
func (*Slice[T]) Elem() string      { return (*new(T)).Appellation() }

func (s *Slice[T]) AfterRead(db ReadAccess) error {
	for _, elem := range *s {
		if hook, ok := any(elem).(ReadCallback); ok {
			if err := hook.AfterRead(db); err != nil {
				return err //nolint:wrapcheck //wrapping adds nothing here
			}
		}
	}

	return nil
}

func (s Slice[T]) String() string {
	builder := &strings.Builder{}

	for i, elem := range s {
		fmt.Fprintf(builder, "Item #%d: %+v\n", i, elem)
	}

	return builder.String()
}

func (s Slice[T]) Preloads() []string {
	if preloader, ok := any(*new(T)).(Preloader); ok {
		return preloader.Preloads()
	}

	return nil
}
