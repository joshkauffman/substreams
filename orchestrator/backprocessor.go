package orchestrator

import (
	"context"
	"fmt"
	
	"github.com/streamingfast/substreams/manifest"
	"github.com/streamingfast/substreams/orchestrator/work"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/substreams/v1"
	"github.com/streamingfast/substreams/service/config"
	"github.com/streamingfast/substreams/store"
)

type Backprocessor struct {
	plan       *work.Plan
	scheduler  *Scheduler
	squasher   *MultiSquasher
	runnerPool work.JobRunnerPool
}

func BuildBackprocessor(
	ctx context.Context,
	runtimeConfig config.RuntimeConfig,
	upToBlock uint64,
	graph *manifest.ModuleGraph,
	respFunc func(resp *pbsubstreams.Response) error,
	storeConfigs store.ConfigMap,
	upstreamRequestModules *pbsubstreams.Modules,
) (*Backprocessor, error) {
	plan, err := work.BuildNewPlan(ctx, storeConfigs, runtimeConfig.StoreSnapshotsSaveInterval, runtimeConfig.SubrequestsSplitSize, upToBlock, graph)
	if err != nil {
		return nil, fmt.Errorf("build work plan: %w", err)
	}

	if err := plan.SendInitialProgressMessages(respFunc); err != nil {
		return nil, fmt.Errorf("send initial progress: %w", err)
	}

	scheduler := NewScheduler(plan, respFunc, upstreamRequestModules)
	if err != nil {
		return nil, err
	}

	squasher, err := NewMultiSquasher(ctx, runtimeConfig, plan.ModulesStateMap, storeConfigs, upToBlock, scheduler.OnStoreCompletedUntilBlock)
	if err != nil {
		return nil, err
	}

	scheduler.OnStoreJobTerminated = squasher.Squash

	runnerPool := work.NewJobRunnerPool(ctx, runtimeConfig.ParallelSubrequests, runtimeConfig.WorkerFactory)

	return &Backprocessor{
		plan:       plan,
		scheduler:  scheduler,
		squasher:   squasher,
		runnerPool: runnerPool,
	}, nil
}

// TODO(abourget): WARN: this function should NOT GROW in functionality, or abstraction levels.
func (b *Backprocessor) Run(ctx context.Context) (store.Map, error) {

	// parallelDownloader := NewLinearExecOutputReader()
	// go parallelDownloader.Launch()
	b.squasher.Launch(ctx)

	if err := b.scheduler.Schedule(ctx, b.runnerPool); err != nil {
		return nil, fmt.Errorf("scheduler run: %w", err)
	}

	finalStoreMap, err := b.squasher.Wait(ctx)
	if err != nil {
		return nil, err
	}

	// if err := parallelDownloader.Wait(); err != nil {
	// 	return nil, err
	// }

	return finalStoreMap, nil
}