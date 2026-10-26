package hindsight

import (
	"context"
	"errors"
	"testing"
)

func TestFakeRuntime(t *testing.T) {
	ctx := context.Background()
	fake := &FakeRuntime{Containers: map[string]ContainerState{
		"box": {ID: "1", Status: "running", Running: true},
	}}
	state, err := fake.Inspect(ctx, "box")
	if err != nil || !state.Running || state.Name != "box" {
		t.Fatalf("inspect = %+v %v", state, err)
	}
	if _, err := fake.Inspect(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing = %v", err)
	}
	fake.InspectErr = errors.New("boom")
	fake.InspectErrOn = 4
	if _, err := fake.Inspect(ctx, "box"); err != nil {
		t.Fatalf("early inspect failed: %v", err)
	}
	if _, err := fake.Inspect(ctx, "box"); err == nil || err.Error() != "boom" {
		t.Fatalf("scripted inspect = %v", err)
	}

	if err := fake.Run(ctx, Spec{Name: "new", Env: []EnvVar{{Name: "HINDSIGHT_API_LLM_API_KEY", Value: "sek", Secret: true}}}); err != nil {
		t.Fatal(err)
	}
	if fake.Specs[0].Env[0].Value != "sek" {
		t.Fatal("spec was not cloned")
	}
	fake.RunErr = errors.New("run failed")
	if err := fake.Run(ctx, Spec{Name: "other"}); err == nil {
		t.Fatal("run error was ignored")
	}
	if err := fake.Stop(ctx, "box"); err != nil {
		t.Fatal(err)
	}
	if fake.Containers["box"].Running {
		t.Fatal("container still running")
	}
	fake.StopErr = errors.New("stop failed")
	if err := fake.Stop(ctx, "box"); err == nil {
		t.Fatal("stop error was ignored")
	}
	fake.StopErr = nil
	if err := fake.Stop(ctx, "gone"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stop missing = %v", err)
	}
	if err := fake.Remove(ctx, "box"); err != nil {
		t.Fatal(err)
	}
	if _, ok := fake.Containers["box"]; ok {
		t.Fatal("container remained")
	}
	fake.RemoveErr = errors.New("rm failed")
	fake.Containers["keep"] = ContainerState{Status: "exited"}
	if err := fake.Remove(ctx, "keep"); err == nil {
		t.Fatal("remove error was ignored")
	}
	fake.RemoveErr = nil
	if err := fake.Remove(ctx, "keep"); err != nil {
		t.Fatal(err)
	}
	if err := fake.Remove(ctx, "keep"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("remove missing = %v", err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := fake.Inspect(canceled, "new"); err == nil {
		t.Fatal("canceled inspect")
	}
	if err := fake.Run(canceled, Spec{Name: "new"}); err == nil {
		t.Fatal("canceled run")
	}
	if err := fake.Stop(canceled, "new"); err == nil {
		t.Fatal("canceled stop")
	}
	if err := fake.Remove(canceled, "new"); err == nil {
		t.Fatal("canceled remove")
	}
	for _, call := range fake.Calls {
		if stringsContainsVolume(call) {
			t.Fatalf("volume call %q", call)
		}
	}
}

func stringsContainsVolume(call string) bool {
	return len(call) >= 6 && call[:6] == "volume"
}
