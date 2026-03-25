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

func (e *Engine) Start(ctx context.Context) {
	var wg sync.WaitGroup

	logger := zerolog.Ctx(ctx)

	// start the modules
	for _, m := range e.modules {
		wg.Go(func() {
			exec(ctx, &m)
		})
	}

	logger.Debug().Msg("All modules have been scheduled :)")

	wg.Wait()
}

func exec(ctx context.Context, m *module.Module) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	r := wazero.NewRuntime(ctx)

	err := m.Run(ctx, r)

	// might be nil
	return err
}
