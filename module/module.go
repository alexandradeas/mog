package module

import (
	"bytes"
	"context"
	hash "crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
)

type Module struct {
	runtime   wazero.Runtime
	ID        uuid.UUID
	status    Status
	StartTime time.Time
	EndTime   time.Time
	Source    string
	compiled  wazero.CompiledModule

	// The hash of the source used in the last successful compilation
	hash string
}

func wat2Wasm(ctx context.Context, wat string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "wat2wasm-*.wasm")
	if err != nil {
		return nil, errors.Cause(errors.WithStack(err))
	}
	defer os.Remove(tmpFile.Name())

	// Close before wat2wasm writes to it by path
	if err := tmpFile.Close(); err != nil {
		return nil, errors.Cause(errors.WithStack(err))
	}

	// run compilation and write to tmp file
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "wat2wasm", "-o", tmpFile.Name(), "-")
	cmd.Stdin = bytes.NewBufferString(wat)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.Cause(errors.WithStack(fmt.Errorf("wat2wasm: %w: %s", err, stderr.String())))
	}

	wasm, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, errors.Cause(errors.WithStack(err))
	}

	return wasm, nil
}

// NewModule will create a new module
//
// This will run an initial compilation of the source code so that it is "warm" for use in a runtime
func NewModule(ctx context.Context, source string) (Module, error) {
	// create module ID
	id, err := uuid.NewUUID()
	if err != nil {
		return Module{}, err
	}

	m := Module{
		runtime: wazero.NewRuntime(ctx),
		ID:      id,
		status:  StatusIdle,
		Source:  source,
	}

	err = m.Compile(ctx)

	return m, err
}

func (m *Module) Run(ctx context.Context, r wazero.Runtime) error {
	m.status = StatusIdle

	err := m.Compile(ctx)
	if err != nil {
		m.status = StatusErrored
		return err
	}

	m.status = StatusRunning

	config := wazero.NewModuleConfig()

	module, err := r.InstantiateModule(ctx, m.compiled, config)
	if err != nil {
		m.status = StatusErrored
		return err
	}

	defer module.Close(ctx)

	return nil
}

func (m *Module) Compile(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	hasher := hash.New()
	_, err := hasher.Write([]byte(m.Source))
	err = errors.Cause(errors.WithStack(err))
	if err != nil {
		return err
	}

	newHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	// shortcircuit if the compilation has already happened
	if m.hash == newHash {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// compile wat format to wasm
	wasm, err := wat2Wasm(ctx, m.Source)
	err = errors.Cause(errors.WithStack(err))
	if err != nil {
		return errors.Cause(errors.WithStack(err))
	}

	logger.Debug().Object("Module", m).Msg("Compiled wasm")

	// compile wasm to wazero module
	compiled, err := m.runtime.CompileModule(ctx, wasm)
	err = errors.Cause(errors.WithStack(err))

	if err != nil {
		return err
	}

	m.compiled = compiled
	m.hash = newHash

	return nil
}

func (m *Module) MarshalZerologObject(e *zerolog.Event) {
	defaultTime := time.Time{}
	e.
		Str("ID", m.ID.String()).
		Str("Status", statusMap[m.status])

	if m.StartTime != defaultTime {
		e.Time("Start Time", m.StartTime)
	}
	if m.EndTime != defaultTime {
		e.Time("End Time", m.EndTime)

	}
}
