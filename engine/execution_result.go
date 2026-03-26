package engine

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ExecutionResult encodes the result from a single module run
//
// Result and Error are not mutually exclusive
type ExecutionResult struct {
	Result   []uint64
	Error    error
	ModuleID uuid.UUID
}

func (r ExecutionResult) MarshalZerologObject(e *zerolog.Event) {
	e.
		Str("module_id", r.ModuleID.String()).
		Uints64("result", r.Result)

	if r.Error != nil {
		e.Str("error", r.Error.Error())
	}
}
