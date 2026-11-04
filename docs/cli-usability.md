# CLI behavior and workflow decisions

This iteration keeps the existing dotfile storage, Git history, Cobra commands,
and shared/local shell manifests. It improves those paths instead of adding a
second interface for agents or replacing the storage format.

The decisions below combine an architecture review, an independent critique,
research into other command-line tools, and separate human and automation
reviews. Reviewers constructed and ran examples with disposable homes, local
Git repositories, fake credentials and harmless shell executables. PowerShell
history informed workflow selection; history is evidence of commands used, not
of their outcome. No personal history or credentials are stored here.

## Included changes

| Problem | Decision and reason |
| --- | --- |
| Different commands require different interactive flags | Missing selections open a picker when both streams are terminals. Explicit complete commands run directly. CI, pipes, JSON and `--no-interactive` suppress prompts. Contradictory modes fail before mutation. |
| Empty remotes and repeated setup | Accept reachable empty repositories and local init without a URL. Preserve real auth/network errors. An explicit changed remote cannot silently succeed; init aliases also create configuration correctly. |
| Destructive shell cleanup | Preserve local manifests, local overrides and unknown/private files. Never recursively purge a shell directory as a side effect of removing its last shared feature. |
| Unsafe history compaction | Reject changed repository/branch/HEAD/worktree state and compare-and-swap the branch reference. A missing remote branch skips compaction, while authentication/network errors remain errors. |
| Credentials and unattended SSH | Enforce the exact HTTPS host boundary on every request, including redirects. Reject an explicit conflicting SSH BatchMode setting instead of unexpectedly prompting. |
| Partial feature batches | Validate selected manifests and configuration before writing. Existing installations without requested settings changes are no-ops. Preserve and commit successes on unexpected execution failures, then return failure with a summary. |
| Silent installed-option changes | Validate supplied options even on reruns. Reject changes through install-only `feature add` and explain the saved-manifest/template-refresh route. `feature info` documents option names, types, defaults and enums. |
| Commit failures after writes | Validate commit identity and unrelated staged changes before automatic mutations. `add/remove --no-commit` deliberately remain available for callers managing their own staging. |
| Confusing back/cancel behavior | Enter keeps the displayed previous/default value, Ctrl+X clears an optional answer, Ctrl+B goes back, Ctrl+S skips one feature, and Esc/Ctrl+C stops remaining work. Completed changes remain explicit. |
| False apply success | Only the actual managed file identity counts as already applied. Unrelated files and wrong/broken symlinks are conflicts. Known conflicts stop execution before writes; unexpected runtime failures return nonzero. |
| Removal leaves a dangling local link | Keeping a managed symlink materializes its content and permissions as an independent local file before deleting its repository source. Other local files and links remain untouched. |
| Profile damage and stale hooks | Reject malformed/ambiguous markers without rewriting. Update exactly one stale managed block while preserving outside bytes. Do not guess a PowerShell profile after discovery fails. |
| Broken shell runtime loading | Eager features rerun when required; lazy loaders remove wrappers before loading and preserve real function-based commands. Fix Fish variable lifetime and Zsh prompt-config quoting. |
| Doctor claims a false repair | Explicit `--fix` can restore an absent known catalog implementation using saved options. It discloses that this is the catalog version, not recovery of customized content. Custom or existing files and symlinks are not replaced. |
| Poor script inspection | `list --json` and `doctor --json` emit one versioned JSON value; errors go to stderr. Missing repositories and failed health checks return nonzero. `doctor --offline` identifies skipped remote connectivity instead of claiming it passed. |
| No safe preview | `apply --dry-run` shares the actual action validation and creates no files, configuration, index changes, hooks or downloads. A bounded PowerShell `-NoProfile` profile-path query is allowed; user profiles/features never execute. |
| Configuration surprises | Read-only commands do not create configuration. Malformed/unreadable configuration fails with its path and remains unchanged. Unknown config keys return failure; help lists readable/writable keys. |
| Feature configured but dependency absent | Doctor warns when enabled Oh My Posh lacks its executable. Applying new/updated hooks explains that a new shell session is needed. Configuration is not described as dependency installation. |
| Dependency vulnerabilities | Update Go modules and development packages; use a patched Go toolchain. Run tests, builds and vulnerability checks after migration. |

## Deliberate deferrals

| Proposal | Reason |
| --- | --- |
| Storage/link-policy migration or recursive adoption | Requires separate migration, editor-save, ACL and cross-filesystem tests. The existing format can be made safer without changing ownership assumptions. |
| General transaction framework or saved executable plans | Cheap validation and explicit partial completion address observed failures. A broad rollback claim would be misleading across filesystem, Git and profile operations. |
| Separate agent command tree, jq/template engine or all-command JSON | Existing deterministic commands plus bounded JSON inspection cover the tested workflows with fewer concepts. |
| Changing compaction's existing `auto/ask/off` default | Preserve configured behavior. Fix history safety immediately; changing which commits are automatically rewritten is a separate compatibility decision. |
| Automatic dependency installer or changing existing module bootstrap defaults | Installation differs by platform and some existing features already bootstrap modules. This needs an explicit compatibility review; do not silently introduce an installer framework. |
| New interactive configuration editor | Do not silently replace customized feature scripts. For now, expose option metadata and accurately describe editing saved options followed by an explicit template refresh. |
| Automatically overwriting apply conflicts | Existing local content must remain safe. Preview explains conflicts and lets the caller deliberately move or adopt the file. |

## Examples for people

Inspect a feature before setting up a repository:

```powershell
oh-my-dot feature info oh-my-posh
```

Before commands that create commits, configure your Git identity if it is not
already set. Choose the name and email you want attached to your commits:

```powershell
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
oh-my-dot init
# Enter a remote URL, or leave it blank for local-only management.
oh-my-dot add
# Select files; already supplied file arguments bypass the picker.
oh-my-dot feature add
# Choose shells and features. Enter skips an unset optional path.
oh-my-dot apply --dry-run
oh-my-dot apply
oh-my-dot doctor --offline
```

The preview does not load features. After applying hooks, open a new shell.
Install prerequisites such as Oh My Posh separately when doctor reports them.
Some existing PowerShell module features have their own bootstrap behavior;
inspect feature scripts before enabling them in unattended environments.

If an existing local file conflicts with apply, inspect it and move it to a
backup deliberately. Apply will not silently replace it. Re-running apply after
resolving the conflict is safe. `remove --keep-linked` stops managing a file
while preserving usable local content.

## Examples for scripts and coding agents

Authentication and Git identity should already be configured. Use a disposable
home when trying these examples. `DOTFILES_REMOTE` is a repository you control:

```sh
export CI=true
export OH_MY_DOT_CONFIG="$HOME/.config/oh-my-dot/config.json"
oh-my-dot init "$DOTFILES_REMOTE" --folder "$HOME/dotfiles" --no-interactive
oh-my-dot feature add git-prompt --shell bash --no-interactive
oh-my-dot apply --dry-run --no-interactive
oh-my-dot apply --no-interactive
```

Use `--no-shell` on both apply commands if the job should manage only dotfiles.
A bare installed-feature add is a no-op. Do not pass different `--option` values
and assume an existing feature changed. `feature update` refreshes templates
using saved options; tool `update` updates the executable.

Inspect without parsing colorized prose:

```sh
oh-my-dot list --json > files.json
status=0
oh-my-dot doctor --offline --json > health.json || status=$?
# Parse health.json even when a health error produced a nonzero status.
exit "$status"
```

Both reports have `schemaVersion: 1`. List has a sorted `files` array of `name`
and optional configured `destination` records. Doctor has `checks`, `errors`
and `warnings`; each check has a name, status, optional shell/message and
fixability. Offline remote checks are reported as skipped warnings. Health
errors return nonzero; warnings alone do not. Normal execution errors are on
stderr, not appended to JSON stdout. No background sync checks run for JSON,
offline, dry-run or noninteractive invocations.

For an already-managed file, inspect the inventory before retrying `add`:
file adoption deliberately rejects an existing registration rather than choosing
another name or replacing content. Use `--no-commit` only when intentionally
managing the index and commit yourself.

## Research sources

These sources informed design choices; the behavior above is verified against
this repository rather than assumed from other tools:

- [GitHub CLI guided repository creation](https://cli.github.com/manual/gh_repo_create)
- [GitHub CLI structured output](https://cli.github.com/manual/gh_help_formatting)
- [GitHub CLI environment controls](https://cli.github.com/manual/gh_help_environment)
- [chezmoi apply and dry-run](https://www.chezmoi.io/reference/commands/apply/)
- [yadm explicit bootstrap](https://yadm.io/docs/bootstrap)
- [Go security release history](https://go.dev/doc/devel/release)
