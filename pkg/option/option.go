// Package option provides a generic Option type for optional values.
package option

// Option represents an optional value.
type Option[T any] struct {
	value *T
}

// Some creates an Option with a value.
func Some[T any](value T) Option[T] {
	return Option[T]{value: &value}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{}
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool {
	return o.value != nil
}

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool {
	return o.value == nil
}

// Unwrap returns the value or panics if empty.
func (o Option[T]) Unwrap() T {
	if o.value == nil {
		panic("attempted to unwrap None option")
	}
	return *o.value
}

// UnwrapOr returns the value or a default if empty.
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if o.value == nil {
		return defaultValue
	}
	return *o.value
}

// UnwrapOrElse returns the value or calls a function to get a default.
func (o Option[T]) UnwrapOrElse(f func() T) T {
	if o.value == nil {
		return f()
	}
	return *o.value
}

// Map transforms the value if present.
func Map[T, U any](o Option[T], f func(T) U) Option[U] {
	if o.value == nil {
		return None[U]()
	}
	return Some(f(*o.value))
}

// FlatMap transforms the value to another Option if present.
func FlatMap[T, U any](o Option[T], f func(T) Option[U]) Option[U] {
	if o.value == nil {
		return None[U]()
	}
	return f(*o.value)
}

// Filter returns the Option if the predicate is satisfied, otherwise None.
func (o Option[T]) Filter(predicate func(T) bool) Option[T] {
	if o.value != nil && predicate(*o.value) {
		return o
	}
	return None[T]()
}

// Or returns this Option if it has a value, otherwise returns the other Option.
func (o Option[T]) Or(other Option[T]) Option[T] {
	if o.value != nil {
		return o
	}
	return other
}

// OrElse returns this Option if it has a value, otherwise calls the function.
func (o Option[T]) OrElse(f func() Option[T]) Option[T] {
	if o.value != nil {
		return o
	}
	return f()
}
