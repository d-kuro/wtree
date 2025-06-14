// Package pipeline provides a generic pipeline for data processing.
package pipeline

// Stage represents a processing stage in the pipeline.
type Stage[In, Out any] func(In) (Out, error)

// Pipeline represents a series of processing stages.
type Pipeline[T any] struct {
	stages []func(any) (any, error)
}

// New creates a new pipeline starting with an initial stage.
func New[In, Out any](stage Stage[In, Out]) *Pipeline[Out] {
	return &Pipeline[Out]{
		stages: []func(any) (any, error){
			func(input any) (any, error) {
				return stage(input.(In))
			},
		},
	}
}

// Then adds a new stage to the pipeline.
func Then[In, Out, Next any](p *Pipeline[Out], stage Stage[Out, Next]) *Pipeline[Next] {
	newStages := make([]func(any) (any, error), len(p.stages)+1)
	copy(newStages, p.stages)
	newStages[len(p.stages)] = func(input any) (any, error) {
		return stage(input.(Out))
	}
	return &Pipeline[Next]{stages: newStages}
}

// Execute runs the pipeline with the given input.
func Execute[In, Out any](p *Pipeline[Out], input In) (Out, error) {
	var current any = input
	var err error

	for _, stage := range p.stages {
		current, err = stage(current)
		if err != nil {
			var zero Out
			return zero, err
		}
	}

	return current.(Out), nil
}

// Parallel executes a stage on multiple inputs concurrently.
func Parallel[In, Out any](stage Stage[In, Out], inputs []In) ([]Out, []error) {
	results := make([]Out, len(inputs))
	errors := make([]error, len(inputs))
	done := make(chan int, len(inputs))

	for i, input := range inputs {
		go func(idx int, in In) {
			result, err := stage(in)
			results[idx] = result
			errors[idx] = err
			done <- idx
		}(i, input)
	}

	for range len(inputs) {
		<-done
	}

	return results, errors
}

// Filter creates a stage that filters values based on a predicate.
func Filter[T any](predicate func(T) bool) Stage[[]T, []T] {
	return func(items []T) ([]T, error) {
		result := make([]T, 0, len(items))
		for _, item := range items {
			if predicate(item) {
				result = append(result, item)
			}
		}
		return result, nil
	}
}

// Map creates a stage that transforms each element in a slice.
func Map[In, Out any](transform func(In) Out) Stage[[]In, []Out] {
	return func(items []In) ([]Out, error) {
		result := make([]Out, len(items))
		for i, item := range items {
			result[i] = transform(item)
		}
		return result, nil
	}
}
