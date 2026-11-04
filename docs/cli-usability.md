# CLI behavior and usage decisions

The CLI uses the same operations for terminal prompts and explicit script inputs.
Shared configuration lives in Git. Device-only declarations, generated variants
and caches stay private. Existing file contents and customized feature scripts
require an explicit choice before replacement.

The design considered novice PowerShell setup, changing an existing theme,
repeated server provisioning, and inspection by coding agents. Architecture
review, independent criticism, CLI research and sanitized PowerShell history
informed these cases. Repeated history entries establish commands used, not
failures or success. No personal history or credentials are stored here.

## Included changes

| Problem | Decision and reason |
| --- | --- |
| Different commands require different interactive flags | Missing selections open a picker when both streams are terminals. Explicit complete commands run directly. CI, pipes, JSON and `--no-interactive` suppress prompts. Contradictory modes fail before mutation. |
| Empty remotes and repeated setup | Accept reachable empty repositories and local init without a URL. Preserve real auth/network errors. An explicit changed remote cannot silently succeed; init aliases also create configuration correctly. |
| Shared removal with a private override | Refuse before writes and name the private removal command. Remove the local override explicitly, then retry shared removal, rather than silently converting partial local settings into a standalone feature. |
| Destructive shell cleanup | Preserve local manifests, local overrides and unknown/private files. Never recursively purge a shell directory as a side effect of removing its last shared feature. |
| Unsafe history compaction | Reject changed repository/branch/HEAD/worktree state and compare-and-swap the branch reference. A missing remote branch skips compaction, while authentication/network errors remain errors. |
| Credentials and unattended SSH | Enforce the exact HTTPS host boundary on every request, including redirects. Reject an explicit conflicting SSH BatchMode setting instead of unexpectedly prompting. |
| Partial feature batches | Validate selected manifests and configuration before writing. Existing installations without requested settings changes are no-ops. Preserve and commit successes on unexpected execution failures, then return failure with a summary. |
| Changing an installed feature requires internal edits | `feature configure` changes typed options and regenerates owned templates. It preserves customized scripts unless `--force` explicitly permits replacement. `feature add` remains a retry-safe installation operation. |
| Private settings enter shared output | Public init uses base declarations only. An ignored private init replaces base loading when local settings exist; private implementations and theme caches have separate ignored paths. Automatic staging independently excludes device-only paths and rejects private paths already tracked by Git. |
| Atomic editor saves detach a managed file | Re-adding an unchanged registration is a no-op. `add FILE --update` explicitly captures the edited live file and restores the hard link, while refusing conflicting stored edits. Doctor reports detached content. |
| Commit failures after writes | Validate commit identity and unrelated staged changes before automatic mutations. `add/remove --no-commit` deliberately remain available for callers managing their own staging. |
| Confusing back/cancel behavior | Enter keeps the displayed previous/default value, Ctrl+X clears an optional answer, Ctrl+B goes back, Ctrl+S skips one feature, and Esc/Ctrl+C stops remaining work. Completed changes remain explicit. |
| False apply success | Only the actual managed file identity counts as already applied. Unrelated files and wrong/broken symlinks are conflicts. Known conflicts stop execution before writes; unexpected runtime failures return nonzero. |
| Removal leaves a dangling local link | Keeping a managed symlink materializes its content and permissions as an independent local file before deleting its repository source. Other local files and links remain untouched. |
| Profile damage and stale hooks | Reject malformed/ambiguous markers without rewriting. Update exactly one stale managed block while preserving outside bytes. Do not guess a PowerShell profile after discovery fails. |
| Broken shell runtime loading | Eager features rerun when required; lazy loaders remove wrappers before loading and preserve real function-based commands. Fix Fish variable lifetime and Zsh prompt-config quoting. |
| Doctor claims a false repair | Explicit `--fix` can restore an absent known catalog implementation using saved options. It discloses that this is the catalog version, not recovery of customized content. Custom or existing files and symlinks are not replaced. |
| Poor script inspection | `feature list/info --json` expose effective options, shared/local provenance and implementation paths. `list --json` and `doctor --json` emit one versioned JSON value; errors go to stderr. Missing repositories and failed health checks return nonzero. `doctor --offline` identifies skipped remote connectivity instead of claiming it passed. |
| No safe preview | `apply --dry-run` shares action validation and does not modify the repository, configuration, managed targets, index or profiles. Bounded PowerShell discovery and module queries run without loading user profiles or features; PowerShell itself may update its [startup caches](https://learn.microsoft.com/en-us/powershell/scripting/dev-cross-plat/performance/startup-performance). |
| Configuration surprises | Read-only commands do not create configuration. Malformed/unreadable configuration fails with its path and remains unchanged. Unknown config keys return failure; help lists readable/writable keys. |
| Feature configured but dependency absent | Add/configure, apply and doctor report missing Oh My Posh or PowerShell module prerequisites with installation instructions. New/refreshed posh-git and Terminal-Icons templates import existing modules; they do not install packages during shell startup. Apply explains activation in a new session. |
| Dependency vulnerabilities | Update Go modules and development packages; use a patched Go toolchain. Run tests, builds and vulnerability checks after migration. |

| Silent history rewriting for new users | The default is `off` when no value was configured. Existing explicit `auto/ask/off` values remain effective. Users can deliberately enable compaction with `config set push-compact auto`. |

## Deliberate deferrals

| Proposal | Reason |
| --- | --- |
| Storage/link-policy migration or recursive adoption | Requires separate migration, editor-save, ACL and cross-filesystem tests. The existing format can be made safer without changing ownership assumptions. |
| General transaction framework or saved executable plans | Cheap validation and explicit partial completion address observed failures. A broad rollback claim would be misleading across filesystem, Git and profile operations. |
| Separate agent command tree, jq/template engine or all-command JSON | Existing deterministic commands plus bounded JSON inspection cover the tested workflows with fewer concepts. |
| Automatic dependency installer | Installation differs by platform. Report the missing prerequisite during setup and leave installation explicit. Already-generated user scripts keep their contents until an explicit template refresh. |
| Separate machine recipe schema | A declarative machine document could simplify larger bootstraps, but a second configuration truth would complicate existing manifests. Reconsider if explicit commands remain repetitive after these lifecycle changes. |
| Automatically overwriting apply conflicts | Existing local content must remain safe. Preview explains conflicts and lets the caller deliberately move or adopt the file. |

## Examples for people

Inspect a feature before setting up a repository:

```powershell
oh-my-dot feature info oh-my-posh
```

Interactive init offers repository-local Git identity setup if needed. Scripts
should configure their intended commit identity in advance:

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
Previously generated module scripts may still contain their earlier bootstrap
behavior. Refresh them explicitly with `feature update` to adopt the current
import-only templates; customized implementations require `--force`.

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
An installed-feature add with unchanged settings is a no-op. Use configure to
change an existing feature and update to refresh its catalog implementation:

```sh
oh-my-dot feature configure oh-my-posh --shell bash --option theme=pure --no-interactive
oh-my-dot feature configure oh-my-posh --shell bash --local --option theme=paradox --no-interactive
oh-my-dot feature info oh-my-posh --shell bash --json
oh-my-dot feature update oh-my-posh --shell bash --local --no-interactive
oh-my-dot feature remove oh-my-posh --shell bash --local --no-interactive
```

Removing a local override restores shared settings. To remove both shared and
private configuration, remove the local override first, then the shared feature. Local-only features also
support add, enable/disable, configure, update and remove with `--local`. These
operations create no commit. Local option values equal to the shared value
inherit that value; a local enable/disable explicitly pins its state. Tool
`update` updates the executable, while `feature update` refreshes templates.

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

For an already-managed file, a repeated `add FILE` reports whether it is still
the managed file. After an editor replaces the file atomically, inspect the
divergence and run `add FILE --update` to choose the live copy. Stored edits
that also diverge from HEAD must be resolved first; `--force` does not bypass
this check. Use `--no-commit` only when intentionally managing the index and
commit yourself. Hard-link capture requires the same filesystem as the existing
file storage.

`config set` without arguments opens a terminal picker. Scripts supply both
key and value. A blank remote URL removes origin; it does not authenticate or
create a remote repository.

## Architecture choice

Three designs were compared. Keeping the current mixed shared/private output
would preserve hooks but continue leaking device settings. The implemented
choice keeps the public entry path and generates a private effective entry in
the same shell directory. The public entry dispatches before loading any base
features, so eager side effects run once. Apply regenerates this output after
shared manifest changes, including changes received through pull. Preview
validates the same runtime preparation without creating output or ignore rules.

Compiling everything outside the repository would isolate outputs further,
but requires migrating hooks and shell-root/helper assumptions. A portable
machine document plus recorded deployment baselines would improve whole-machine
review and two-sided file synchronization, but needs a single source of truth
and a storage migration. Both remain real alternatives if these explicit
operations prove insufficient. Neither is necessary to remove private values
from tracked output or capture an atomic editor save safely.

## Research sources

These sources informed design choices; the behavior above is verified against
this repository rather than assumed from other tools:

- [GitHub CLI guided repository creation](https://cli.github.com/manual/gh_repo_create)
- [GitHub CLI structured output](https://cli.github.com/manual/gh_help_formatting)
- [GitHub CLI environment controls](https://cli.github.com/manual/gh_help_environment)
- [chezmoi apply and dry-run](https://www.chezmoi.io/reference/commands/apply/)
- [yadm explicit bootstrap](https://yadm.io/docs/bootstrap)
- [Go security release history](https://go.dev/doc/devel/release)

- [Terraform preview before apply](https://developer.hashicorp.com/terraform/cli/commands/plan)
- [uv explicit Python download controls](https://docs.astral.sh/uv/guides/install-python/)
