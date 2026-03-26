package main

import (
	"context"
	"os"

	"alexandradeas.co.uk/mog/engine"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

const everything = `
(module
  ;; Define a function that returns the constant value 42
  (func $getAnswer (result i32)
    i32.const 42)

  ;; Export the function as "getAnswer" so it can be called from the host
  (func (export "getAnswer") (result i32)
    call $getAnswer)
)`

func main() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx := logger.WithContext(context.Background())

	exitCode := 0

	mog := engine.Engine{}

	logger.Info().Msg("Creating modules")
	errors := mog.Initialize(ctx, []string{everything})
	for _, err := range errors {
		logger.Err(err).Msg("Failed to initialize module")
		exitCode = 1
	}

	if exitCode != 0 {
		logger.Error().Msg("Module initialization failed")
		os.Exit(exitCode)
	}
	logger.Info().Msg("Module creation completed")

	logger.Info().Msg("Running modules...")
	mog.Start(ctx)
	logger.Info().Msg("Finished")
}
