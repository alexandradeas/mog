package engine_test

import (
	"testing"

	. "alexandradeas.co.uk/mog/engine"
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

func TestEngineInitialize(t *testing.T) {
	engine := Engine{}

	errors := engine.Initialize(t.Context(), []string{everything})

	if len(errors) != 0 {
		t.Errorf("Failed to initialize engine: %v", errors)
	}
}
