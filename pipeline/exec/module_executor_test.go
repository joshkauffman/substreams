package exec

import (
	"context"
	"fmt"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/substreams/v1"
	"github.com/streamingfast/substreams/pipeline/execout"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MockExecOutput struct {
	clockFunc func() *pbsubstreams.Clock

	cacheMap map[string][]byte
}

func (t *MockExecOutput) Clock() *pbsubstreams.Clock {
	return t.clockFunc()
}

func (t *MockExecOutput) Get(name string) ([]byte, bool, error) {
	v, ok := t.cacheMap[name]
	if !ok {
		return nil, false, execout.NotFound
	}
	return v, true, nil
}

func (t *MockExecOutput) Set(name string, value []byte) (err error) {
	t.cacheMap[name] = value
	return nil
}

type MockModuleExecutor struct {
	name string

	RunFunc   func(ctx context.Context, reader execout.ExecutionOutputGetter) (out []byte, moduleOutputData pbsubstreams.ModuleOutputData, err error)
	ApplyFunc func(value []byte) error
	LogsFunc  func() (logs []string, truncated bool)
	StackFunc func() []string
}

func (t *MockModuleExecutor) Name() string {
	return t.name
}

func (t *MockModuleExecutor) String() string {
	return fmt.Sprintf("TestModuleExecutor(%s)", t.name)
}

func (t *MockModuleExecutor) Reset() {}

func (t *MockModuleExecutor) run(ctx context.Context, reader execout.ExecutionOutputGetter) (out []byte, moduleOutputData pbsubstreams.ModuleOutputData, err error) {
	if t.RunFunc != nil {
		return t.RunFunc(ctx, reader)
	}
	return nil, nil, fmt.Errorf("not implemented")
}

func (t *MockModuleExecutor) applyCachedOutput(value []byte) error {
	if t.ApplyFunc != nil {
		return t.ApplyFunc(value)
	}
	return fmt.Errorf("not implemented")
}

func (t *MockModuleExecutor) moduleLogs() (logs []string, truncated bool) {
	if t.LogsFunc != nil {
		return t.LogsFunc()
	}
	return nil, false
}

func (t *MockModuleExecutor) currentExecutionStack() []string {
	if t.StackFunc != nil {
		return t.StackFunc()
	}
	return nil
}

func TestModuleExecutorRunner_Run_HappyPath(t *testing.T) {
	ctx := context.Background()
	executor := &MockModuleExecutor{
		name: "test",
		RunFunc: func(ctx context.Context, reader execout.ExecutionOutputGetter) (out []byte, moduleOutputData pbsubstreams.ModuleOutputData, err error) {
			return []byte("test"), &pbsubstreams.ModuleOutput_MapOutput{}, nil
		},
		LogsFunc: func() (logs []string, truncated bool) {
			return []string{"test"}, false
		},
	}
	output := &MockExecOutput{
		cacheMap: make(map[string][]byte),
	}

	moduleOutput, err := RunModule(ctx, executor, output)
	if err != nil {
		t.Fatal(err)
	}

	assert.NoError(t, err)
	assert.NotEmpty(t, moduleOutput)
}