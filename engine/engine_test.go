package engine_test

import (
	"os/exec"
	"testing"

	. "alexandradeas.co.uk/mog/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const everything = `
(module
  (func $getAnswer (result i32)
    i32.const 42)

  (func (export "getAnswer") (result i32)
    call $getAnswer)
)`

func TestEngineInitialize(t *testing.T) {
	if _, err := exec.LookPath("wat2wasm"); err != nil {
		t.Skip("wat2wasm not found in PATH")
	}

	e := Engine{}
	errs := e.Initialize(t.Context(), []string{everything})
	require.Empty(t, errs)
}

func TestEngineInitializeInvalidWAT(t *testing.T) {
	if _, err := exec.LookPath("wat2wasm"); err != nil {
		t.Skip("wat2wasm not found in PATH")
	}

	e := Engine{}
	errs := e.Initialize(t.Context(), []string{"(invalid)"})
	assert.NotEmpty(t, errs)
}
