---
name: shell-feature-authoring
description: 'Create or edit oh-my-dot shell feature catalog entries and templates. Use for adding new shell features, updating existing feature scripts, choosing shell-specific vs posix fallback templates, wiring feature options, and validating shell feature behavior.'
argument-hint: 'Describe the feature name, whether this is a new feature or an edit, target shells, and whether options or load-strategy changes are needed.'
---

# Shell Feature Authoring

## When to Use

- Add a new built-in shell feature to the catalog.
- Edit an existing feature template under `internal/catalog/features/`.
- Add or change feature options in `internal/catalog/catalog.go`.
- Extend a feature to another shell or convert duplicated bash/zsh logic into a `posix.sh` fallback.
- Review whether a prompt, environment, alias, completion, or tool feature should be `eager` or `on-command`.

## What This Skill Produces

This skill guides the agent to make repo-source changes for shell features, not ad hoc edits to generated files in a user's `omd-shells/<shell>/features/` directory. The source of truth is the catalog metadata plus embedded templates in `internal/catalog/features/`.

## Decision Points

### 1. Is this a new feature or an existing one?

- Existing feature: start with the matching directory in `internal/catalog/features/<feature>/` and nearby catalog metadata.
- New feature: add metadata to `internal/catalog/catalog.go`, then add template files for the supported shells.

### 2. Does the feature need shell-specific implementations?

- Use `posix.sh` when bash/zsh/posix can share the same implementation, only `bash.sh` or `zsh.sh` only when POSIX fallback is not sufficient.
- Add `fish.fish` for fish syntax.
- Add `powershell.ps1` for PowerShell syntax.

### 3. Does the feature need user-configurable options?

- If no: keep the template static and avoid adding option machinery.
- If yes: define `Options` metadata in `internal/catalog/catalog.go` and render values through the catalog template system.

### 4. Does the feature need a feature-level guard?

- Add a guard only for expensive or one-time setup.
- Do not guard prompt/environment reapplication just to avoid re-sourcing. Eager features may need to run again when shell rc files are re-sourced.

## Procedure

### 1. Anchor on the owning source

- Read `internal/catalog/catalog.go` for the feature metadata.
- Read `internal/catalog/template.go` to confirm template file names, fallback rules, and available rendering helpers.
- Read the existing feature templates in `internal/catalog/features/<feature>/` before changing behavior.

### 2. Update catalog metadata first

For a new or changed feature, verify these fields in `internal/catalog/catalog.go`:

- `Name`
- `Description`
- `Category`
- `DefaultStrategy`
- `DefaultCommands` for `on-command` features
- `SupportedShells`
- `Options` when interactive configuration is required

Rules:

- `SupportedShells` must match the templates you actually provide.
- `on-command` features should declare meaningful `DefaultCommands`.
- Prefer minimal options; only add prompts for real user choices.

### 3. Author the template files

Template directory layout:

- `internal/catalog/features/<feature>/posix.sh`
- `internal/catalog/features/<feature>/bash.sh`
- `internal/catalog/features/<feature>/zsh.sh`
- `internal/catalog/features/<feature>/fish.fish`
- `internal/catalog/features/<feature>/powershell.ps1`

Authoring rules:

- Do not add shebangs to feature template files.
- Write shell-native syntax for the target shell.
- Keep the template focused on feature behavior; init scripts handle shell bootstrapping and sourcing.
- If bash/zsh/sh share logic, prefer `posix.sh` over copy-pasted variants.
- If the feature uses options, prefer the generic template helpers exposed by `RenderFeatureTemplate`, such as `hasOption` and `option`.
- Only use named template context fields such as `.FeatureName`, `.ShellName`, or feature-specific fields after confirming they exist in `internal/catalog/template.go`.
- Do not rely on template fields that do not exist in `internal/catalog/template.go`. Extend the template context in Go first if the template needs more data.

### 4. Respect load-strategy semantics

- `eager`: for aliases, prompt setup, environment setup, and other startup behavior that should be available immediately.
- `on-command`: for completions or heavy tooling that can load lazily when a command is first used.
- `defer`: only use if the surrounding implementation supports the intended behavior.

When editing eager features:

- Prompt and environment features should usually re-apply on shell rc re-source.
- Add a feature-local loaded flag only when repeated execution is incorrect or unnecessarily expensive.

### 5. Wire options only when necessary

If the feature needs configuration:

- Declare `Options` metadata in `internal/catalog/catalog.go` with the smallest correct type.
- Prefer enums for constrained choices.
- Use file/path options only when the feature really depends on user-supplied paths.
- Ensure required options have a clear prompt path and that non-interactive behavior remains sensible.

Relevant option types already supported:

- `string`
- `int`
- `bool`
- `enum`
- `file`
- `path`

### 6. Validate the change at the closest layer

Run the narrowest checks that match the slice you changed:

- Template or fallback changes: `go test ./internal/catalog`
- Catalog metadata or feature selection changes: `go test ./internal/featurecmd`
- Manifest or add/remove flow changes: `go test ./internal/shell ./internal/manifest`
- Option prompt or validation changes: `go test ./internal/options ./internal/validation`

If you changed Go files, run `gofmt -w` on the touched files before the test run.

## Completion Checklist

- Metadata and templates describe the same shell support and behavior.
- `SupportedShells` aligns with the shell-specific templates or `posix.sh` fallback available under `internal/catalog/features/<feature>/`.
- `DefaultCommands` is set for `on-command` features.
- No feature template includes a shebang.
- POSIX fallback is used deliberately instead of duplicating bash/zsh code.
- Any new option has appropriate type, defaults, and validation expectations.
- The nearest focused tests pass.

## Repo References

- `internal/catalog/catalog.go`
- `internal/catalog/template.go`
- `internal/catalog/template_test.go`
- `internal/shell/operations.go`
- `docs/specs/shell-framework/README.md`
- `docs/specs/feature-options/README.md`
- `AGENTS.md`

## Notes For Ambiguous Requests

If the target cannot be inferred from the user's request or the existing catalog, clarify one point before broad changes:

- Are they changing an existing catalog template, adding a new built-in feature, or modifying generated feature output in a specific repo?

If the user asks for a new built-in feature and does not specify shells, default to the smallest supported shell set that the feature can genuinely maintain well.
