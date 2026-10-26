package hindsight

import (
	"context"
	"sync"
)

// FakeRuntime is a Runtime that records calls and never invokes Docker.
type FakeRuntime struct {
	mu sync.Mutex
	// Containers is the inspectable set. A missing name returns ErrNotFound.
	Containers map[string]ContainerState
	// RunErr is returned by Run when set.
	RunErr error
	// StopErr is returned by Stop when set.
	StopErr error
	// RemoveErr is returned by Remove when set.
	RemoveErr error
	// InspectErr is returned by the inspect call selected by InspectErrOn.
	InspectErr error
	// InspectErrOn is the 1-based inspect call that returns InspectErr.
	// Zero disables InspectErr.
	InspectErrOn int
	// Specs are the Run requests, including secret env values for assertions.
	Specs []Spec
	// Calls records operations in order, such as "inspect name".
	Calls []string

	inspects int
}

// Inspect returns a stored container or ErrNotFound.
func (f *FakeRuntime) Inspect(ctx context.Context, name string) (ContainerState, error) {
	if err := ctx.Err(); err != nil {
		return ContainerState{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inspects++
	f.Calls = append(f.Calls, "inspect "+name)
	if f.InspectErr != nil && f.InspectErrOn == f.inspects {
		return ContainerState{}, f.InspectErr
	}
	state, ok := f.Containers[name]
	if !ok {
		return ContainerState{}, ErrNotFound
	}
	state.Name = name
	return state, nil
}

// Run records spec and marks the container running unless RunErr is set.
func (f *FakeRuntime) Run(ctx context.Context, spec Spec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, "run "+spec.Name)
	f.Specs = append(f.Specs, cloneSpec(spec))
	if f.RunErr != nil {
		return f.RunErr
	}
	if f.Containers == nil {
		f.Containers = map[string]ContainerState{}
	}
	f.Containers[spec.Name] = ContainerState{ID: "fake", Name: spec.Name, Status: "running", Running: true}
	return nil
}

// Stop marks a container exited.
func (f *FakeRuntime) Stop(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, "stop "+name)
	if f.StopErr != nil {
		return f.StopErr
	}
	state, ok := f.Containers[name]
	if !ok {
		return ErrNotFound
	}
	state.Running = false
	state.Status = "exited"
	f.Containers[name] = state
	return nil
}

// Remove deletes a container record and does not touch volumes.
func (f *FakeRuntime) Remove(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, "remove "+name)
	if f.RemoveErr != nil {
		return f.RemoveErr
	}
	if _, ok := f.Containers[name]; !ok {
		return ErrNotFound
	}
	delete(f.Containers, name)
	return nil
}

func cloneSpec(spec Spec) Spec {
	spec.Env = append([]EnvVar(nil), spec.Env...)
	return spec
}
