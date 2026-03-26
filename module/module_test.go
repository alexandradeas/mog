package module

import (
	"bytes"
	hash "crypto/sha1"
	"encoding/base64"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

var minimalWasm = []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}

const validWAT = `
(module
  (func $getAnswer (result i32)
    i32.const 42)

  (func (export "getAnswer") (result i32)
    call $getAnswer)
)`

const invalidWAT = "(invalid)"

// watHash matches the hashing algorithm used inside Module.compile.
func watHash(wat string) string {
	h := hash.New()
	h.Write([]byte(wat))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func TestCompileUsesCache(t *testing.T) {
	wat := validWAT
	m := &Module{
		wat:  wat,
		wasm: minimalWasm,
		hash: watHash(wat),
	}

	err := m.compile(t.Context())
	require.NoError(t, err)
	assert.True(t, bytes.Equal(m.wasm, minimalWasm), "wasm bytes should be unchanged on cache hit")
}

func TestCompileInvalidatesCache(t *testing.T) {
	m := &Module{
		wat:  invalidWAT,
		wasm: minimalWasm,
		hash: "stale-hash",
	}

	err := m.compile(t.Context())
	assert.Error(t, err)
}

func TestWat2WasmFailure(t *testing.T) {
	_, err := NewModule(t.Context(), invalidWAT)
	assert.Error(t, err)
}

func TestRunStatusCompileError(t *testing.T) {
	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	m := &Module{
		wat:  invalidWAT,
		hash: "stale-hash",
	}

	err := m.Run(ctx, r)
	assert.Error(t, err)
	assert.Equal(t, StatusErrored, m.status)
}

func TestRunStatusInstantiateError(t *testing.T) {
	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	wat := validWAT
	m := &Module{
		wat:  wat,
		wasm: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		hash: watHash(wat),
	}

	err := m.Run(ctx, r)
	assert.Error(t, err)
	assert.Equal(t, StatusErrored, m.status)
}

func TestRunStatusCompleted(t *testing.T) {
	ctx := t.Context()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	wat := validWAT
	m := &Module{
		wat:  wat,
		wasm: minimalWasm,
		hash: watHash(wat),
	}

	err := m.Run(ctx, r)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, m.status)
}

func TestRunWithRealRuntime(t *testing.T) {
	if _, err := exec.LookPath("wat2wasm"); err != nil {
		t.Skip("wat2wasm not found in PATH")
	}

	ctx := t.Context()

	m, err := NewModule(ctx, validWAT)
	require.NoError(t, err)

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	err = m.Run(ctx, r)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, m.GetStatus())
}
