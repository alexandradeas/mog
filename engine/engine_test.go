package engine_test

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"testing"

	. "alexandradeas.co.uk/mog/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const everything = `
(module
  (func $getAnswer (result i32)
    i32.const 42)

  (func (export "_start") (result i32)
    call $getAnswer)
)`

const willError = `
(module
  (func $infinite_recursion
    (call $infinite_recursion)
  )
  (func (export "run")
    (call $infinite_recursion)
  )
)
`

func TestInitialize(t *testing.T) {
	if _, err := exec.LookPath("wat2wasm"); err != nil {
		t.Skip("wat2wasm not found in PATH")
	}

	for c := range slices.Values([]struct {
		name        string
		source      string
		expectError bool
	}{
		{
			name:        "valid WAT",
			source:      everything,
			expectError: false,
		},
		{
			name:        "invalid WAT",
			source:      "(invalid)",
			expectError: true,
		},
		{
			name:        "WAT with runtime error",
			source:      willError,
			expectError: false,
		},
	}) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := Engine{}
			errs := e.Initialize(t.Context(), []string{c.source})
			if c.expectError {
				require.NotEmpty(t, errs)
			} else {
				require.Empty(t, errs)
			}
		})
	}
}

func TestRun(t *testing.T) {
	if _, err := exec.LookPath("wat2wasm"); err != nil {
		t.Skip("wat2wasm not found in PATH")
	}

	for c := range slices.Values([]struct {
		name string
		in   []string
		out  []ExecutionResult
	}{
		{
			name: "single run",
			in:   []string{everything},
			out: []ExecutionResult{
				{Result: []uint64{42}},
			},
		},
		{
			name: "multiple successful runs",
			in:   []string{everything, everything, everything},
			out: []ExecutionResult{
				{Result: []uint64{42}},
				{Result: []uint64{42}},
				{Result: []uint64{42}},
			},
		},
		{
			name: "error run",
			in:   []string{willError},
			out: []ExecutionResult{
				{Result: []uint64(nil), Error: errors.New("module does not export a _start function")},
			},
		},
	}) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			engine := Engine{}
			engine.Initialize(ctx, c.in)

			result := engine.Start(ctx)
			assert.Len(t, result, len(c.out), "should have the correct number of results")

			for i, expect := range c.out {
				assert.EqualValues(t, expect.Result, result[i].Result)
				if expect.Error != nil {
					assert.EqualError(t, result[i].Error, fmt.Sprintf("%s", expect.Error))
				}
			}
		})

	}
}
