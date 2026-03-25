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
	ID        uuid.UUID
	status    Status
	StartTime time.Time
	EndTime   time.Time
	wat       string
	wasm      []byte

	// The hash of the source used to compile the wasm
	hash string
}

func wat2Wasm(ctx context.Context, wat string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "wat2wasm-*.wasm")
	if err != nil {
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}

	return wasm, nil
}

// NewModule will create a new module
//
// This will run an initial compilation of the source WAT code to WASM
func NewModule(ctx context.Context, source string) (Module, error) {
	// create module ID
	id, err := uuid.NewUUID()
	if err != nil {
		return Module{}, err
	}

	wasm, err := wat2Wasm(ctx, source)

	m := Module{
		ID:     id,
		status: StatusIdle,
		wat:    source,
		wasm:   wasm,
	}

	return m, err
}

func (m *Module) Run(ctx context.Context, r wazero.Runtime) error {
	m.status = StatusIdle

	err := m.compile(ctx)
	if err != nil {
		m.status = StatusErrored
		return err
	}

	m.status = StatusRunning

	module, err := r.Instantiate(ctx, m.wasm)
	if err != nil {
		m.status = StatusErrored
		return err
	}

	defer module.Close(ctx)

	m.status = StatusCompleted

	return nil
}

func (m *Module) GetStatus() Status {
	return m.status
}

func (m *Module) compile(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	hasher := hash.New()
	_, err := hasher.Write([]byte(m.wat))
	err = errors.WithStack(err)
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
	wasm, err := wat2Wasm(ctx, m.wat)
	if err != nil {
		return errors.WithStack(err)
	}

	m.wasm = wasm

	m.hash = newHash

	logger.Debug().Object("Module", m).Msg("Compiled wasm")

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
