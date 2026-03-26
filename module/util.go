package module

import (
	"bytes"
	"context"
	hash "crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

func wat2Wasm(ctx context.Context, wat string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "wat2wasm-*.wasm")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer os.Remove(tmpFile.Name())

	// Close before wat2wasm writes to it by path
	if err := tmpFile.Close(); err != nil {
		return nil, errors.WithStack(err)
	}

	// run compilation and write to tmp file
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "wat2wasm", "-o", tmpFile.Name(), "-")
	cmd.Stdin = bytes.NewBufferString(wat)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.WithStack(fmt.Errorf("wat2wasm: %w: %s", err, stderr.String()))
	}

	wasm, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return wasm, nil
}

// calculates a hash for the provided string
//
// if an error occurs an empty string is returned, along with an error that includes a stack trace
func calcHash(in string) (string, error) {
	hasher := hash.New()
	_, err := hasher.Write([]byte(in))
	if err != nil {
		return "", errors.WithStack(err)
	}

	return base64.URLEncoding.EncodeToString(hasher.Sum(nil)), nil

}
