// Package result provides a generic Result type for error handling.
package result

// Result represents either a successful value or an error.
type Result[T any] struct {
	value *T
	err   error
}

// Ok creates a successful Result with the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: &value}
}

// Err creates a failed Result with the given error.
func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// IsOk returns true if the Result contains a value.
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr returns true if the Result contains an error.
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Value returns the value and a boolean indicating success.
func (r Result[T]) Value() (T, bool) {
	if r.value != nil {
		return *r.value, true
	}
	var zero T
	return zero, false
}

// Error returns the error, if any.
func (r Result[T]) Error() error {
	return r.err
}

// Unwrap returns the value or panics if there's an error.
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return *r.value
}

// UnwrapOr returns the value or a default if there's an error.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.err != nil {
		return defaultValue
	}
	return *r.value
}

// Map transforms the value using the provided function if Result is Ok.
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return Ok(f(*r.value))
}

// FlatMap transforms the value using a function that returns a Result.
func FlatMap[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return f(*r.value)
}

// MapErr transforms the error using the provided function if Result is Err.
func MapErr[T any](r Result[T], f func(error) error) Result[T] {
	if r.err != nil {
		return Err[T](f(r.err))
	}
	return r
}
