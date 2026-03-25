package engine

import "testing"

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

	engine.Initialize(t.Context(), []string{everything})

	if len(engine.modules) != 1 {
		t.Error("Failed to initialize")
	}
}
