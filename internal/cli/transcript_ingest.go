package cli

import (
	"encoding/json"
	"flag"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vikashkumar21/sdd/internal/state"
	"github.com/vikashkumar21/sdd/internal/transcript"
)

var roleNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)

// TranscriptIngestResult is returned when ingesting transcript tokens with --json.
type TranscriptIngestResult struct {
	TaskID         string               `json:"task_id"`
	Source         string               `json:"source"`
	Role           string               `json:"role"`
	FromStep       int                  `json:"from_step"`
	ToStep         int                  `json:"to_step"`
	Steps          int                  `json:"steps"`
	SkippedNoUsage int                  `json:"skipped_no_usage"`
	LastStepSeen   int                  `json:"last_step_seen"`
	DryRun         bool                 `json:"dry_run"`
	NothingNew     bool                 `json:"nothing_new"`
	Saved          bool                 `json:"saved"`
	Path           string               `json:"path,omitempty"`
	UpdatedAt      string               `json:"updated_at,omitempty"`
	Delta          state.TokenTelemetry `json:"delta"`
	Totals         state.TokenTelemetry `json:"totals"`
}

type transcriptIngestOpts struct {
	cmdName        string
	taskID         string
	root           string
	conversationID string
	transcriptFile string
	transcriptRoot string
	fromStep       int
	fromSet        bool
	toStep         int
	toSet          bool
	role           string
	dryRun         bool
	jsonOut        bool
}

func validateTranscriptFlags(cmdName string, visited map[string]bool, conv, file string, fromStep, toStep int, role string, env IO) (useTranscript bool, exitCode int) {
	hasConv := visited["conversation"]
	hasFile := visited["transcript"]
	hasRoot := visited["transcript-root"]
	hasFrom := visited["from-step"]
	hasTo := visited["to-step"]
	hasRole := visited["role"]
	hasDry := visited["dry-run"]
	hasManual := visited["input"] || visited["cache"] || visited["output"] || visited["stdin"]

	if !hasConv && !hasFile {
		if hasRoot || hasFrom || hasTo || hasRole || hasDry {
			fprintf(env.Err, "sdd %s: --from-step, --to-step, --role, --dry-run, and --transcript-root require --conversation or --transcript\n", cmdName)
			return false, 2
		}
		return false, 0
	}

	if hasConv && hasFile {
		fprintf(env.Err, "sdd %s: --conversation and --transcript are mutually exclusive\n", cmdName)
		return true, 2
	}
	if hasManual {
		fprintf(env.Err, "sdd %s: transcript flags cannot be combined with --input, --cache, --output, or --stdin\n", cmdName)
		return true, 2
	}
	if hasRoot && !hasConv {
		fprintf(env.Err, "sdd %s: --transcript-root requires --conversation\n", cmdName)
		return true, 2
	}
	if hasConv && strings.TrimSpace(conv) == "" {
		fprintf(env.Err, "sdd %s: --conversation must not be empty\n", cmdName)
		return true, 2
	}
	if hasFile && strings.TrimSpace(file) == "" {
		fprintf(env.Err, "sdd %s: --transcript must not be empty\n", cmdName)
		return true, 2
	}
	if hasFrom && fromStep < 0 {
		fprintf(env.Err, "sdd %s: --from-step must be >= 0\n", cmdName)
		return true, 2
	}
	if hasTo && toStep < 0 {
		fprintf(env.Err, "sdd %s: --to-step must be >= 0\n", cmdName)
		return true, 2
	}
	if !roleNameRe.MatchString(role) {
		fprintf(env.Err, "sdd %s: invalid --role %q (must match ^[a-z][a-z0-9-]{0,39}$)\n", cmdName, role)
		return true, 2
	}
	return true, 0
}

func collectVisitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func ingestTranscript(opts transcriptIngestOpts, env IO) int {
	var sourceKey string
	var rc io.ReadCloser
	var err error

	if opts.conversationID != "" {
		if err := transcript.ValidateConversationID(opts.conversationID); err != nil {
			fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
			return 1
		}
		tRoot := opts.transcriptRoot
		if strings.TrimSpace(tRoot) == "" {
			tRoot, err = transcript.DefaultRoot(env.home())
			if err != nil {
				fprintf(env.Err, "sdd %s: resolve transcript root: %v\n", opts.cmdName, err)
				return 1
			}
		}
		sourceKey = opts.conversationID
		rc, err = transcript.OpenConversation(tRoot, opts.conversationID)
	} else {
		absFile, aerr := filepath.Abs(opts.transcriptFile)
		if aerr != nil {
			fprintf(env.Err, "sdd %s: resolve transcript path: %v\n", opts.cmdName, aerr)
			return 1
		}
		sourceKey = transcript.SourceKeyForPath(absFile)
		rc, err = transcript.OpenFile(absFile)
	}
	if err != nil {
		fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
		return 1
	}
	defer func() { _ = rc.Close() }()

	stateMap, hasState, err := state.ReadState(opts.root, opts.taskID)
	if err != nil {
		fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
		return 1
	}
	existingTokens, _ := state.ExtractTokens(stateMap)

	fromStep := opts.fromStep
	if hasState {
		if err := state.ValidateIngestState(stateMap); err != nil {
			fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
			return 1
		}
		if prevStep, prevRole, ok := state.CursorFor(stateMap, sourceKey); ok {
			if prevRole != "" && prevRole != opts.role {
				fprintf(env.Err, "sdd %s: role %q does not match stored role %q for source %s\n", opts.cmdName, opts.role, prevRole, sourceKey)
				return 1
			}
			if !opts.fromSet {
				fromStep = prevStep + 1
			} else if fromStep <= prevStep {
				fprintf(env.Err, "sdd %s: --from-step %d overlaps already-ingested cursor %d for source %s\n", opts.cmdName, fromStep, prevStep, sourceKey)
				return 1
			}
		} else if !opts.fromSet {
			fprintf(env.Err, "sdd %s: --from-step is required on first ingestion of source %s\n", opts.cmdName, sourceKey)
			return 1
		}
	} else if !opts.fromSet {
		fprintf(env.Err, "sdd %s: --from-step is required on first ingestion of source %s\n", opts.cmdName, sourceKey)
		return 1
	}

	toStep := transcript.NoLimit
	if opts.toSet {
		toStep = opts.toStep
		if fromStep > toStep {
			fprintf(env.Err, "sdd %s: --from-step %d exceeds --to-step %d\n", opts.cmdName, fromStep, toStep)
			return 1
		}
	}

	sum, err := transcript.Summarize(rc, transcript.Options{
		FromStep: fromStep,
		ToStep:   toStep,
	})
	if err != nil {
		fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
		return 1
	}

	effectiveTo := sum.LastStepSeen
	if toStep != transcript.NoLimit && (effectiveTo < 0 || toStep < effectiveTo) {
		effectiveTo = toStep
	}
	if sum.Steps == 0 {
		out := TranscriptIngestResult{
			TaskID:         opts.taskID,
			Source:         sourceKey,
			Role:           opts.role,
			FromStep:       fromStep,
			ToStep:         effectiveTo,
			Steps:          0,
			SkippedNoUsage: sum.SkippedNoUsage,
			LastStepSeen:   sum.LastStepSeen,
			DryRun:         opts.dryRun,
			NothingNew:     true,
			Saved:          false,
			Totals:         existingTokens,
		}
		return printIngestResult(opts.cmdName, out, opts.jsonOut, env)
	}

	delta, err := state.AccumulateTokens(state.TokenTelemetry{}, sum.Input, sum.CacheRead, sum.Output)
	if err != nil {
		fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
		return 1
	}

	if opts.dryRun {
		projTotals, err := state.AccumulateTokens(existingTokens, sum.Input, sum.CacheRead, sum.Output)
		if err != nil {
			fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
			return 1
		}
		out := TranscriptIngestResult{
			TaskID:         opts.taskID,
			Source:         sourceKey,
			Role:           opts.role,
			FromStep:       fromStep,
			ToStep:         effectiveTo,
			Steps:          sum.Steps,
			SkippedNoUsage: sum.SkippedNoUsage,
			LastStepSeen:   sum.LastStepSeen,
			DryRun:         true,
			NothingNew:     false,
			Saved:          false,
			Delta:          delta,
			Totals:         projTotals,
		}
		return printIngestResult(opts.cmdName, out, opts.jsonOut, env)
	}

	res, totals, err := state.RecordIngest(opts.root, opts.taskID, state.Ingest{
		Source:   sourceKey,
		Role:     opts.role,
		FromStep: fromStep,
		ToStep:   effectiveTo,
		Steps:    sum.Steps,
		Delta:    delta,
	}, env.now())
	if err != nil {
		fprintf(env.Err, "sdd %s: %v\n", opts.cmdName, err)
		return 1
	}

	out := TranscriptIngestResult{
		TaskID:         res.TaskID,
		Source:         sourceKey,
		Role:           opts.role,
		FromStep:       fromStep,
		ToStep:         effectiveTo,
		Steps:          sum.Steps,
		SkippedNoUsage: sum.SkippedNoUsage,
		LastStepSeen:   sum.LastStepSeen,
		DryRun:         false,
		NothingNew:     false,
		Saved:          res.Saved,
		Path:           res.Path,
		UpdatedAt:      res.UpdatedAt,
		Delta:          delta,
		Totals:         totals,
	}
	return printIngestResult(opts.cmdName, out, opts.jsonOut, env)
}

func printIngestResult(cmdName string, out TranscriptIngestResult, jsonOut bool, env IO) int {
	if jsonOut {
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd %s: encode json: %v\n", cmdName, err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}
	if out.NothingNew {
		fprintf(env.Out, "Nothing new for %s (source=%s, role=%s, from_step=%d, last_step_seen=%d, skipped_no_usage=%d)\n",
			out.TaskID, out.Source, out.Role, out.FromStep, out.LastStepSeen, out.SkippedNoUsage)
		return 0
	}
	prefix := "Ingested"
	if out.DryRun {
		prefix = "[dry-run] Would ingest"
	}
	fprintf(env.Out, "%s %s (%s, role=%s, steps=%d..%d, counted=%d, skipped=%d): input=+%d, cache=+%d, output=+%d (totals: input=%d, cache=%d, output=%d, total=%d)\n",
		prefix, out.TaskID, out.Source, out.Role, out.FromStep, out.ToStep, out.Steps, out.SkippedNoUsage,
		out.Delta.InputTokens, out.Delta.CacheReadTokens, out.Delta.OutputTokens,
		out.Totals.InputTokens, out.Totals.CacheReadTokens, out.Totals.OutputTokens, out.Totals.TotalTokens)
	return 0
}
