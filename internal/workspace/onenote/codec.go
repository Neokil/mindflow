// Package onenote isolates the bounded native OneNote file codec from storage and HTTP.
package onenote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mindflow/internal/workspace"
	"os/exec"
	"sync"
	"time"
)

const (
	maxFileBytes      = 64 << 20
	maxNotesBytes     = 16 << 10
	maxNoteLength     = 1000
	conversionTimeout = time.Minute
)

// Codec runs one explicit, trusted native executable without a shell or database access.
type Codec struct {
	path       string
	processing sync.Mutex
}

// NewCodec fixes the operator-configured executable path; files never choose commands.
func NewCodec(path string) *Codec { return &Codec{path: path} }

// Convert translates a bounded native file or validated mindflow transfer archive.
func (c *Codec) Convert(ctx context.Context, mode string, input []byte) ([]byte, []string, error) {
	switch mode {
	case "read-one", "read-onepkg", "write-one", "write-onepkg":
	default:
		return nil, nil, invalid("Unsupported OneNote conversion.")
	}
	if len(input) > maxFileBytes {
		return nil, nil, invalid("Choose a OneNote transfer up to 64 MiB.")
	}
	if !c.processing.TryLock() {
		return nil, nil, &workspace.Error{Code: "metadata_conflict", Message: "Another OneNote conversion is running. Retry shortly."}
	}
	defer c.processing.Unlock()
	if c.path == "" {
		return nil, nil, invalid("This server build does not include the native OneNote codec.")
	}
	ctx, cancel := context.WithTimeout(ctx, conversionTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, c.path, mode) //nolint:gosec // G204: executable is operator configuration; mode is allowlisted; file bytes only reach stdin.
	command.Env = []string{"DOTNET_EnableDiagnostics=0", "DOTNET_GCHeapHardLimit=0x20000000"}
	command.Stdin = bytes.NewReader(input)
	output := limitedBuffer{limit: maxFileBytes}
	notes := limitedBuffer{limit: maxNotesBytes}
	command.Stdout = &output
	command.Stderr = &notes
	err := command.Run()
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, invalid("OneNote conversion timed out or was cancelled. Try a smaller file.")
		}
		if _, ok := errors.AsType[*exec.Error](err); ok {
			return nil, nil, fmt.Errorf("start OneNote codec: %w", err)
		}

		return nil, nil, invalid("The OneNote document is corrupt, encrypted, unsupported, or exceeds the conversion limits.")
	}
	var warnings []string
	err = json.Unmarshal(notes.Bytes(), &warnings)
	if err != nil || len(warnings) > 30 {
		return nil, nil, invalid("The OneNote codec returned an invalid conversion report.")
	}
	for _, note := range warnings {
		if len(note) > maxNoteLength {
			return nil, nil, invalid("The OneNote conversion report exceeds its limit.")
		}
	}

	return output.Bytes(), warnings, nil
}

func invalid(message string) error {
	return &workspace.Error{Code: "validation_failed", Message: message}
}

var errOutputLimit = errors.New("native codec output exceeds its limit")

type limitedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.buffer.Len() {
		return 0, errOutputLimit
	}
	count, err := b.buffer.Write(data)
	if err != nil {
		return count, fmt.Errorf("buffer native conversion: %w", err)
	}

	return count, nil
}

func (b *limitedBuffer) Bytes() []byte { return b.buffer.Bytes() }
