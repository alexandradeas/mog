package module

import (
	hash "crypto/sha1"
	"encoding/base64"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

const minimalWAT = "(module)"

var minimalWasm = []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}

const successfulWAT = `
(module
  (func $getAnswer (result i32)
    i32.const 42)

  (func (export "_start") (result i32)
    call $getAnswer)
)`

const invalidWAT = "(invalid)"

const runtimeErrorWAT = `
(module
  (func $unresolvable (import "env" "nonexistent_function"))
  (func (export "_start")
    (call $unresolvable)
  )
)
`

const invalidExportWAT = `
(module
  (func $getAnswer (result i32)
    i32.const 42)

  (func (export "getAnswer") (result i32)
    call $getAnswer)
)`

// watHash matches the hashing algorithm used inside Module.compile.
func watHash(wat string) string {
	h := hash.New()
	h.Write([]byte(wat))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func TestCompile(t *testing.T) {
	for c := range slices.Values([]struct {
		name        string
		module      *Module
		shouldError bool
		hash        string
		compiled    []byte
		cacheHit    bool
	}{
		{
			name: "uses cache when valid",
			module: &Module{
				wat:  minimalWAT,
				wasm: minimalWasm,
				hash: watHash(minimalWAT),
			},
			shouldError: false,
			cacheHit:    true,
			hash:        watHash(minimalWAT),
			compiled:    minimalWasm,
		},
		{
			name: "updates cache on compile",
			module: &Module{
				wat:  minimalWAT,
				wasm: []byte{},
				hash: "stale-hash",
			},
			shouldError: false,
			cacheHit:    false,
			hash:        watHash(minimalWAT),
			compiled:    minimalWasm,
		},
		{
			name: "maintains wasm on compile failure",
			module: &Module{
				wat:  "(invalid)",
				wasm: minimalWasm,
				hash: watHash(minimalWAT),
			},
			shouldError: true,
			cacheHit:    true,
			hash:        watHash(minimalWAT),
			compiled:    minimalWasm,
		},
	}) {
		t.Run(c.name, func(t *testing.T) {
			err := c.module.compile(t.Context())
			if c.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.EqualValues(t, c.compiled, c.module.wasm, "compiled wasm does not match expectation")

			assert.Equal(t, c.hash, c.module.hash)
		})

	}
}

func TestNewModuleFailure(t *testing.T) {
	_, err := NewModule(t.Context(), invalidWAT)
	assert.Error(t, err)
}

func TestRun(t *testing.T) {
	for c := range slices.Values([]struct {
		name        string
		module      *Module
		status      Status
		result      []uint64
		shouldError bool
	}{
		{
			name: "compile error",
			module: &Module{
				wat:  "(invalid)",
				hash: "stale-hash",
			},
			status:      StatusErrored,
			result:      nil,
			shouldError: true,
		},
		{
			name: "instantiate error",
			module: &Module{
				wat:  successfulWAT,
				wasm: []byte{0xFF, 0xFF, 0xFF, 0xFF},
				hash: watHash(successfulWAT),
			},
			shouldError: true,
			status:      StatusErrored,
			result:      []uint64([]uint64(nil)),
		},
		{
			name:        "success",
			shouldError: false,
			status:      StatusCompleted,
			result:      []uint64{42},
			module: &Module{
				wat: successfulWAT,
			},
		},
		{
			name:        "resets result on compile failure",
			shouldError: true,
			status:      StatusErrored,
			result:      nil,
			module: &Module{
				wat:    invalidWAT,
				status: StatusIdle,
				result: []uint64{42},
			},
		},
		{
			name:        "resets result on invalid export error",
			shouldError: true,
			status:      StatusErrored,
			result:      nil,
			module: &Module{
				wat:    invalidExportWAT,
				status: StatusIdle,
				result: []uint64{42},
			},
		},
		{
			name:        "resets result on execution failure",
			shouldError: true,
			status:      StatusErrored,
			result:      nil,
			module: &Module{
				wat:    runtimeErrorWAT,
				status: StatusIdle,
				result: []uint64{42},
			},
		},
	}) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			r := wazero.NewRuntime(ctx)
			defer r.Close(ctx)

			res, err := c.module.Run(ctx, r)
			if c.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.status, c.module.GetStatus())
			assert.Equal(t, c.result, res)
			assert.Equal(t, c.result, c.module.GetResult())
		})
	}
}
