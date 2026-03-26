package module

import (
	"bytes"
	hash "crypto/sha1"
	"encoding/base64"
	"os/exec"
	"testing"

	"github.com/tetratelabs/wazero"
)

// minimalWasm is a minimal valid empty WebAssembly module binary (magic + version).
var minimalWasm = []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}

const validWAT = `
(module
  (func $getAnswer (result i32)
    i32.const 42)

  (func (export "getAnswer") (result i32)
    call $getAnswer)
)`

const invalidWAT = "(invalid)"

// watHash computes the base64-encoded SHA1 hash of a WAT string, matching
// the algorithm used inside Module.compile so the precomputed values align.
func watHash(wat string) string {
	h := hash.New()
	h.Write([]byte(wat))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// TestCompileUsesCache verifies that compile() short-circuits and leaves the
// cached wasm bytes untouched when the WAT hash has not changed.
func TestCompileUsesCache(t *testing.T) {
	wat := validWAT
	m := &Module{
		wat:  wat,
		wasm: minimalWasm,
		hash: watHash(wat), // pre-set to match current WAT → cache hit
	}

	if err := m.compile(t.Context()); err != nil {
		t.Fatalf("expected no error on cache hit, got: %v", err)
	}

	if !bytes.Equal(m.wasm, minimalWasm) {
		t.Error("expected wasm bytes to be unchanged on a compile cache hit")
	}
}

// TestCompileInvalidatesCache verifies that compile() attempts to recompile
// when the stored hash does not match the current WAT content.
func TestCompileInvalidatesCache(t *testing.T) {
	ctx := t.Context()

	// stale-hash does not match SHA1(invalidWAT), so recompile is triggered.
	m := &Module{
		wat:  invalidWAT,
		wasm: minimalWasm,
		hash: "stale-hash",
	}

	if err := m.compile(ctx); err == nil {
		t.Error("expected error when recompiling invalid WAT, got nil")
	}
}

// TestWat2WasmFailure verifies that NewModule returns an error when given
// syntactically invalid WebAssembly text.
func TestWat2WasmFailure(t *testing.T) {
	ctx := t.Context()

	if _, err := NewModule(ctx, invalidWAT); err == nil {
		t.Error("expected error for invalid WAT input, got nil")
	}
}

// TestRunStatusCompileError verifies that Run sets the module status to
// StatusErrored when the compilation step fails.
func TestRunStatusCompileError(t *testing.T) {
	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// stale-hash forces a recompile of invalidWAT, which must fail.
	m := &Module{
		wat:  invalidWAT,
		hash: "stale-hash",
	}

	if err := m.Run(ctx, r); err == nil {
		t.Error("expected error when running a module with invalid WAT")
	}

	if m.status != StatusErrored {
		t.Errorf("expected StatusErrored after compile failure, got %v", m.status)
	}
}

// TestRunStatusInstantiateError verifies that Run sets the module status to
// StatusErrored when the wazero runtime fails to instantiate the wasm bytes.
func TestRunStatusInstantiateError(t *testing.T) {
	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	wat := validWAT
	// The bytes below do not start with the WASM magic number (\x00asm), so
	// wazero will reject them during Instantiate, triggering the error path.
	m := &Module{
		wat:  wat,
		wasm: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		hash: watHash(wat),
	}

	if err := m.Run(ctx, r); err == nil {
		t.Error("expected error when instantiating invalid wasm bytes")
	}

	if m.status != StatusErrored {
		t.Errorf("expected StatusErrored after instantiation failure, got %v", m.status)
	}
}

// TestRunStatusCompleted verifies the full success path: Run transitions the
// status to StatusCompleted after successfully instantiating the module.
func TestRunStatusCompleted(t *testing.T) {
	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	wat := validWAT
	// Use a pre-loaded minimal valid wasm so no wat2wasm invocation is needed.
	m := &Module{
		wat:  wat,
		wasm: minimalWasm,
		hash: watHash(wat),
	}

	if err := m.Run(ctx, r); err != nil {
		t.Fatalf("expected no error on successful run, got: %v", err)
	}

	if m.status != StatusCompleted {
		t.Errorf("expected StatusCompleted, got %v", m.status)
	}
}

// TestRunWithRealRuntime tests the end-to-end path: compiling real WAT with
// wat2wasm and then instantiating the resulting wasm in a wazero runtime.
// This test is skipped if the wat2wasm binary is not present on PATH.
func TestRunWithRealRuntime(t *testing.T) {
	if _, err := exec.LookPath("wat2wasm"); err != nil {
		t.Skip("wat2wasm not found in PATH, skipping real compilation test")
	}

	ctx := t.Context()

	m, err := NewModule(ctx, validWAT)
	if err != nil {
		t.Fatalf("NewModule failed: %v", err)
	}

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	if err := m.Run(ctx, r); err != nil {
		t.Fatalf("expected no error running compiled module, got: %v", err)
	}

	if m.GetStatus() != StatusCompleted {
		t.Errorf("expected StatusCompleted, got %v", m.GetStatus())
	}
}
