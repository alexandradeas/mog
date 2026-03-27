package engine

import (
	"context"
	"sync"
	"time"

	"alexandradeas.co.uk/mog/module"
	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
)

type Engine struct {
	modules []module.Module
}

func (e *Engine) Initialize(ctx context.Context, srcs []string) []error {
	logger := zerolog.Ctx(ctx)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	var errors []error

	// compile the sources
	for _, src := range srcs {
		wg.Go(func() {
			m, err := module.NewModule(ctx, src)
			if err != nil {
				mutex.Lock()
				errors = append(errors, err)
				mutex.Unlock()
				logger.Error().Err(err).Stack().Msg("Failed to create module")
			} else {
				mutex.Lock()
				e.modules = append(e.modules, m)
				mutex.Unlock()
			}
		})
	}

	wg.Wait()

	return errors

}

func (e *Engine) Start(ctx context.Context) []ExecutionResult {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	logger := zerolog.Ctx(ctx)

	var results []ExecutionResult

	// start the modules
	for i := range e.modules {
		m := &e.modules[i]
		wg.Go(func() {
			result := ExecutionResult{ModuleID: m.ID}

			result.Result, result.Error = exec(ctx, m)

			if result.Error != nil {
				logger.Err(result.Error).Stack().Msg("Module failed")
			}

			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()

			logger.Debug().Object("module", m).Object("result", result).Msg("module exection completed")
		})
	}

	logger.Debug().Msg("All modules have been scheduled :)")

	wg.Wait()
	logger.Debug().Msg("All modules have finished executing")

	return results
}

func exec(ctx context.Context, m *module.Module) ([]uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	return m.Run(ctx, r)
}
