package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/state"
)

func runTaskState(args []string, env IO) int {
	if len(args) == 0 {
		fprintln(env.Err, "usage: sdd task state <command> [flags]\n\ncommands:\n  show <T-xxx>          show task state snapshot and event journal (--json supported)\n  save <T-xxx>          atomically save task state snapshot (--state or --stdin, --json)\n  append-event <T-xxx>  append lifecycle event to event journal (--event or --stdin, --json)\n  purge <T-xxx>         completely remove task state directory (--force, --json)")
		return 2
	}
	switch args[0] {
	case "show":
		return runTaskStateShow(args[1:], env)
	case "save":
		return runTaskStateSave(args[1:], env)
	case "append-event":
		return runTaskStateAppendEvent(args[1:], env)
	case "purge":
		return runTaskStatePurge(args[1:], env)
	default:
		fprintf(env.Err, "sdd task state: unknown command %q\n\nusage: sdd task state <show|save|append-event|purge> [flags]\n", args[0])
		return 2
	}
}

func runTaskStateShow(args []string, env IO) int {
	fs := newFlagSet("task state show", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task state show <T-xxx> [--json] [--dir <project>]")
		return 2
	}

	normID, err := state.ValidateTaskID(pos[0])
	if err != nil {
		fprintf(env.Err, "sdd task state show: %v\n", err)
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task state show: %v\n", err)
		return 1
	}

	res, err := state.Show(root, normID, nil)
	if err != nil {
		fprintf(env.Err, "sdd task state show: %v\n", err)
		return 1
	}

	if *jsonFlag {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd task state show: encode json: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	if !res.Found {
		fprintf(env.Err, "sdd task state: no state or events found for %s\n", normID)
		return 1
	}

	// Human-readable output
	fprintf(env.Out, "Task State: %s\n", res.TaskID)
	if res.WorktreeActive {
		fprintf(env.Out, "Worktree: Active (%s)\n", res.WorktreePath)
	} else {
		fprintln(env.Out, "Worktree: Inactive")
	}

	if res.State != nil {
		fprintln(env.Out, "\nState Snapshot:")
		keys := make([]string, 0, len(res.State))
		for k := range res.State {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fprintf(env.Out, "  %s: %v\n", k, res.State[k])
		}
	}

	if len(res.Events) > 0 {
		fprintf(env.Out, "\nEvents (%d):\n", len(res.Events))
		for _, evt := range res.Events {
			if evtMap, ok := evt.(map[string]any); ok {
				ts, _ := evtMap["timestamp"].(string)
				kind, _ := evtMap["kind"].(string)
				details, _ := evtMap["details"].(string)
				if ts != "" || kind != "" {
					fprintf(env.Out, "  [%s] %s: %s\n", ts, kind, details)
					continue
				}
			}
			evtJSON, _ := json.Marshal(evt)
			fprintf(env.Out, "  %s\n", string(evtJSON))
		}
	}

	return 0
}

func runTaskStateSave(args []string, env IO) int {
	fs := newFlagSet("task state save", env)
	dir := fs.String("dir", ".", "project root")
	stateFlag := fs.String("state", "", "state JSON payload")
	stdinFlag := fs.Bool("stdin", false, "read state JSON payload from standard input")
	jsonFlag := fs.Bool("json", false, "output structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task state save <T-xxx> [--state <json> | --stdin] [--json] [--dir <project>]")
		return 2
	}

	normID, err := state.ValidateTaskID(pos[0])
	if err != nil {
		fprintf(env.Err, "sdd task state save: %v\n", err)
		return 1
	}

	var payload []byte
	if *stdinFlag {
		data, err := state.ReadBounded(env.In, state.MaxStdinStateBytes)
		if err != nil {
			fprintf(env.Err, "sdd task state save: read stdin: %v\n", err)
			return 1
		}
		payload = data
	} else if strings.TrimSpace(*stateFlag) != "" {
		payload = []byte(*stateFlag)
	} else {
		fprintln(env.Err, "sdd task state save: either --state <json> or --stdin is required")
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task state save: %v\n", err)
		return 1
	}

	res, err := state.SaveState(root, normID, payload, env.now())
	if err != nil {
		fprintf(env.Err, "sdd task state save: %v\n", err)
		return 1
	}

	if *jsonFlag {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd task state save: encode json: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	fprintf(env.Out, "Saved task state for %s to %s\n", res.TaskID, res.Path)
	return 0
}

func runTaskStateAppendEvent(args []string, env IO) int {
	fs := newFlagSet("task state append-event", env)
	dir := fs.String("dir", ".", "project root")
	eventFlag := fs.String("event", "", "event JSON payload")
	stdinFlag := fs.Bool("stdin", false, "read event JSON payload from standard input")
	jsonFlag := fs.Bool("json", false, "output structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task state append-event <T-xxx> [--event <json> | --stdin] [--json] [--dir <project>]")
		return 2
	}

	normID, err := state.ValidateTaskID(pos[0])
	if err != nil {
		fprintf(env.Err, "sdd task state append-event: %v\n", err)
		return 1
	}

	var payload []byte
	if *stdinFlag {
		data, err := state.ReadBounded(env.In, state.MaxStdinEventBytes)
		if err != nil {
			fprintf(env.Err, "sdd task state append-event: read stdin: %v\n", err)
			return 1
		}
		payload = data
	} else if strings.TrimSpace(*eventFlag) != "" {
		payload = []byte(*eventFlag)
	} else {
		fprintln(env.Err, "sdd task state append-event: either --event <json> or --stdin is required")
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task state append-event: %v\n", err)
		return 1
	}

	res, err := state.AppendEvent(root, normID, payload)
	if err != nil {
		fprintf(env.Err, "sdd task state append-event: %v\n", err)
		return 1
	}

	if *jsonFlag {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd task state append-event: encode json: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	fprintf(env.Out, "Appended event to .sdd/state/%s/events.jsonl (total events: %d)\n", res.TaskID, res.EventCount)
	return 0
}

func runTaskStatePurge(args []string, env IO) int {
	fs := newFlagSet("task state purge", env)
	dir := fs.String("dir", ".", "project root")
	force := fs.Bool("force", false, "purge without confirmation")
	jsonFlag := fs.Bool("json", false, "output structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task state purge <T-xxx> [--force] [--json] [--dir <project>]")
		return 2
	}

	normID, err := state.ValidateTaskID(pos[0])
	if err != nil {
		fprintf(env.Err, "sdd task state purge: %v\n", err)
		return 1
	}

	if !*force && !*jsonFlag {
		if env.Interactive {
			p := newPrompter(env.In, env.Out)
			if !p.confirm(fmt.Sprintf("Purge all state and event history for %s?", normID)) {
				fprintln(env.Out, "Purge cancelled.")
				return 0
			}
		} else {
			fprintln(env.Err, "sdd task state purge: --force required in non-interactive environment")
			return 1
		}
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task state purge: %v\n", err)
		return 1
	}

	res, err := state.Purge(root, normID)
	if err != nil {
		fprintf(env.Err, "sdd task state purge: %v\n", err)
		return 1
	}

	if *jsonFlag {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd task state purge: encode json: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	fprintf(env.Out, "Purged state directory for %s (%s)\n", res.TaskID, res.Path)
	return 0
}
