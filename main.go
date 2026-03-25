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

	mog := engine.Engine{}

	logger.Info().Msg("Creating modules")
	mog.Initialize(ctx, []string{everything})
	logger.Info().Msg("Module creation completed")

	logger.Info().Msg("Running modules...")
	mog.Start(ctx)
	logger.Info().Msg("All modules finished")
}
