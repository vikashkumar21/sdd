// Package transcript provides bounded Antigravity JSONL transcript summarisation
// and confined source resolution for token telemetry ingestion.
package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	// DefaultMaxBytes caps total transcript bytes read (256 MiB).
	DefaultMaxBytes int64 = 256 << 20
	// DefaultMaxLineBytes caps a single JSONL record (1 MiB).
	DefaultMaxLineBytes = 1 << 20
	// NoLimit leaves the upper step bound open-ended.
	NoLimit = -1
)

// Options configures bounded transcript summarisation over [FromStep, ToStep].
type Options struct {
	FromStep     int
	ToStep       int // NoLimit means open-ended; values < NoLimit are rejected.
	MaxBytes     int64
	MaxLineBytes int
}

// Summary holds summed token counters and step metadata for a transcript window.
type Summary struct {
	Input          int64
	CacheRead      int64
	Output         int64
	Steps          int
	SkippedNoUsage int
	LastStepSeen   int // Highest step_index seen in any record; -1 if no records.
}

type rawRecord struct {
	StepIndex       *int   `json:"step_index"`
	Source          string `json:"source"`
	Type            string `json:"type"`
	InputTokens     *int64 `json:"input_tokens"`
	CacheReadTokens *int64 `json:"cache_read_tokens"`
	OutputTokens    *int64 `json:"output_tokens"`
}

// Summarize reads a JSONL transcript from r and sums token usage for MODEL/PLANNER_RESPONSE
// records whose step_index lies in [opts.FromStep, opts.ToStep].
func Summarize(r io.Reader, opts Options) (Summary, error) {
	if r == nil {
		return Summary{}, errors.New("transcript reader is nil")
	}
	if opts.FromStep < 0 {
		return Summary{}, fmt.Errorf("invalid from-step %d: must be >= 0", opts.FromStep)
	}
	if opts.ToStep < NoLimit {
		return Summary{}, fmt.Errorf("invalid to-step %d: must be >= %d", opts.ToStep, NoLimit)
	}
	if opts.ToStep != NoLimit && opts.FromStep > opts.ToStep {
		return Summary{}, fmt.Errorf("from-step %d exceeds to-step %d", opts.FromStep, opts.ToStep)
	}

	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	maxLine := opts.MaxLineBytes
	if maxLine <= 0 {
		maxLine = DefaultMaxLineBytes
	}

	limited := &io.LimitedReader{R: r, N: maxBytes + 1}
	br := bufio.NewReaderSize(limited, 64*1024)

	sum := Summary{LastStepSeen: -1}
	lineNum := 0

	for {
		line, tooLong, err := readBoundedLine(br, maxLine)
		if tooLong {
			return Summary{}, fmt.Errorf("line %d: exceeds maximum line size of %d bytes", lineNum+1, maxLine)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return Summary{}, fmt.Errorf("line %d: read error", lineNum+1)
		}
		if limited.N == 0 {
			return Summary{}, fmt.Errorf("line %d: transcript exceeds maximum size of %d bytes", lineNum+1, maxBytes)
		}

		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			if errors.Is(err, io.EOF) {
				break
			}
			lineNum++
			return Summary{}, fmt.Errorf("line %d: empty transcript line", lineNum)
		}
		lineNum++

		var rec rawRecord
		if uerr := json.Unmarshal(trimmed, &rec); uerr != nil {
			// Format without echoing raw transcript bytes (R-002, R-007).
			return Summary{}, fmt.Errorf("line %d: invalid transcript record", lineNum)
		}
		if rec.StepIndex == nil || *rec.StepIndex < 0 {
			return Summary{}, fmt.Errorf("line %d: missing or negative step_index", lineNum)
		}
		if (rec.InputTokens != nil && *rec.InputTokens < 0) ||
			(rec.CacheReadTokens != nil && *rec.CacheReadTokens < 0) ||
			(rec.OutputTokens != nil && *rec.OutputTokens < 0) {
			return Summary{}, fmt.Errorf("line %d: negative token count", lineNum)
		}

		step := *rec.StepIndex
		if step > sum.LastStepSeen {
			sum.LastStepSeen = step
		}

		inWindow := step >= opts.FromStep && (opts.ToStep == NoLimit || step <= opts.ToStep)
		if inWindow && rec.Source == "MODEL" && rec.Type == "PLANNER_RESPONSE" {
			nilCount := 0
			if rec.InputTokens == nil {
				nilCount++
			}
			if rec.CacheReadTokens == nil {
				nilCount++
			}
			if rec.OutputTokens == nil {
				nilCount++
			}
			switch nilCount {
			case 3:
				sum.SkippedNoUsage++
			case 0:
				var aerr error
				sum.Input, aerr = addChecked(sum.Input, *rec.InputTokens)
				if aerr != nil {
					return Summary{}, fmt.Errorf("line %d: input_tokens %w", lineNum, aerr)
				}
				sum.CacheRead, aerr = addChecked(sum.CacheRead, *rec.CacheReadTokens)
				if aerr != nil {
					return Summary{}, fmt.Errorf("line %d: cache_read_tokens %w", lineNum, aerr)
				}
				sum.Output, aerr = addChecked(sum.Output, *rec.OutputTokens)
				if aerr != nil {
					return Summary{}, fmt.Errorf("line %d: output_tokens %w", lineNum, aerr)
				}
				sum.Steps++
			default:
				return Summary{}, fmt.Errorf("line %d: partial token usage", lineNum)
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return sum, nil
}

func readBoundedLine(br *bufio.Reader, maxLine int) (line []byte, tooLong bool, err error) {
	var buf []byte
	for {
		frag, rerr := br.ReadSlice('\n')
		if len(buf)+len(frag) > maxLine {
			return nil, true, nil
		}
		buf = append(buf, frag...)
		if !errors.Is(rerr, bufio.ErrBufferFull) {
			return buf, false, rerr
		}
	}
}

func addChecked(cur, delta int64) (int64, error) {
	if math.MaxInt64-cur < delta {
		return 0, errors.New("sum overflows int64")
	}
	return cur + delta, nil
}
