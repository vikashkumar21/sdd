package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"time"

	"github.com/vikashkumar21/sdd/internal/update"
	"github.com/vikashkumar21/sdd/internal/version"
)

func runSelfUpdate(args []string, env IO) int {
	fs := newFlagSet("self-update", env)
	checkOnly := fs.Bool("check", false, "check for updates without downloading or modifying executable")
	dryRun := fs.Bool("dry-run", false, "download and verify update without replacing executable")
	targetVersion := fs.String("version", "", "target a specific version release (e.g. 1.5.0 or v1.5.0)")
	force := fs.Bool("force", false, "force update even if already at latest or requested version")
	yes := fs.Bool("yes", false, "skip confirmation prompt")
	fs.BoolVar(yes, "y", false, "alias for --yes")
	jsonOut := fs.Bool("json", false, "output result in JSON format")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fprintf(env.Err, "sdd self-update: %v\n", err)
		return 2
	}

	baseURL := env.getenv("SDD_UPDATE_BASE_URL")
	execPath := env.getenv("SDD_EXEC_PATH")
	client := env.HTTPClient

	opts := update.Options{
		BaseURL:        baseURL,
		Client:         client,
		CurrentVersion: version.Version,
		TargetVersion:  *targetVersion,
		Force:          *force,
		CheckOnly:      *checkOnly,
		DryRun:         *dryRun,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := update.Check(ctx, opts)
	if err != nil {
		if *jsonOut {
			data, _ := json.MarshalIndent(map[string]any{
				"error": err.Error(),
			}, "", "  ")
			fprintln(env.Err, string(data))
		} else {
			fprintf(env.Err, "sdd self-update: %v\n", err)
		}
		return 1
	}

	if *checkOnly {
		if *jsonOut {
			data, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				fprintf(env.Err, "sdd self-update: %v\n", err)
				return 1
			}
			fprintln(env.Out, string(data))
			return 0
		}

		if res.UpdateAvailable {
			fprintf(env.Out, "Update available: %s -> %s\nRun 'sdd self-update' to update.\n", res.CurrentVersion, res.LatestVersion)
		} else {
			fprintf(env.Out, "sdd is up-to-date (version %s)\n", res.CurrentVersion)
		}
		return 0
	}

	if !res.UpdateAvailable {
		if *jsonOut {
			data, _ := json.MarshalIndent(res, "", "  ")
			fprintln(env.Out, string(data))
			return 0
		}
		if *targetVersion != "" {
			fprintf(env.Out, "sdd is already at requested version %s\n", res.LatestVersion)
		} else {
			fprintf(env.Out, "sdd is up-to-date (version %s)\n", res.CurrentVersion)
		}
		return 0
	}

	if !*yes && env.Interactive {
		action := "update"
		if update.CompareSemver(res.CurrentVersion, res.LatestVersion) > 0 {
			action = "downgrade"
		}
		fprintf(env.Out, "Ready to %s sdd from %s to %s (%s)\n", action, res.CurrentVersion, res.LatestVersion, res.AssetName)
		pr := newPrompter(env.In, env.Out)
		if !pr.confirm("Proceed with update?") {
			fprintf(env.Out, "Update cancelled.\n")
			return 0
		}
	}

	if *dryRun {
		fprintf(env.Out, "Dry run: downloading and verifying update %s -> %s (%s)...\n", res.CurrentVersion, res.LatestVersion, res.AssetName)
		bytes, err := update.DownloadAndVerify(ctx, res, opts)
		if err != nil {
			fprintf(env.Err, "sdd self-update: download/verify failed: %v\n", err)
			return 1
		}
		fprintf(env.Out, "✓ Downloaded and verified %d bytes (SHA256 valid)\n", len(bytes))
		fprintf(env.Out, "Dry run complete: binary not modified.\n")
		if *jsonOut {
			data, _ := json.MarshalIndent(map[string]any{
				"dry_run":          true,
				"current_version":  res.CurrentVersion,
				"latest_version":   res.LatestVersion,
				"asset_name":       res.AssetName,
				"bytes_downloaded": len(bytes),
				"status":           "verified",
			}, "", "  ")
			fprintln(env.Out, string(data))
		}
		return 0
	}

	fprintf(env.Out, "Downloading and verifying update %s -> %s (%s)...\n", res.CurrentVersion, res.LatestVersion, res.AssetName)
	bytes, err := update.DownloadAndVerify(ctx, res, opts)
	if err != nil {
		fprintf(env.Err, "sdd self-update: download/verify failed: %v\n", err)
		return 1
	}

	if err := update.ReplaceExecutable(execPath, bytes); err != nil {
		fprintf(env.Err, "sdd self-update: replace failed: %v\n", err)
		return 1
	}

	fprintf(env.Out, "✓ Successfully updated sdd to %s\n", res.LatestVersion)
	if *jsonOut {
		data, _ := json.MarshalIndent(map[string]any{
			"updated":         true,
			"current_version": res.CurrentVersion,
			"new_version":     res.LatestVersion,
			"asset_name":      res.AssetName,
			"status":          "success",
		}, "", "  ")
		fprintln(env.Out, string(data))
	}
	return 0
}
