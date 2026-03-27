package module

import (
	"context"
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

	// the result from the run of this module, initially nil
	result []uint64

	// The hash of the source used to compile the wasm
	hash string
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

	if err != nil {
		return Module{}, err
	}

	hash, err := calcHash(source)

	if err != nil {
		return Module{}, err
	}

	return Module{
		ID:     id,
		status: StatusIdle,
		wat:    source,
		wasm:   wasm,
		hash:   hash,
	}, err
}

func (m *Module) Run(ctx context.Context, r wazero.Runtime, args ...uint64) ([]uint64, error) {
	m.status = StatusIdle

	// reset result to nil
	m.result = nil
	err := m.compile(ctx)
	if err != nil {
		m.status = StatusErrored
		return nil, err
	}

	m.status = StatusRunning

	module, err := r.Instantiate(ctx, m.wasm)
	if err != nil {
		m.status = StatusErrored
		return nil, errors.WithStack(err)
	}

	defer module.Close(ctx)

	f := module.ExportedFunction("_start")

	if f == nil {
		m.status = StatusErrored
		return nil, errors.WithStack(errors.New("module does not export a _start function"))
	}

	res, err := f.Call(ctx, args...)
	if err != nil {
		m.status = StatusErrored
		return nil, errors.WithStack(err)
	}

	m.result = res
	m.status = StatusCompleted

	return res, err
}

func (m *Module) GetResult() []uint64 {
	return m.result
}

func (m *Module) GetStatus() Status {
	return m.status
}

func (m *Module) compile(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	newHash, err := calcHash(m.wat)
	if err != nil {
		return err
	}
	// shortcircuit if the compilation has already happened
	if m.hash == newHash {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
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
		e.Time("start_time", m.StartTime)
	}
	if m.EndTime != defaultTime {
		e.Time("end_time", m.EndTime)

	}
}
