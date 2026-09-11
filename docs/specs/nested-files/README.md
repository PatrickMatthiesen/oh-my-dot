# Nested dotfile names

A repository file is identified by its forward-slash path relative to `files/`.
The destination remains separate in `linkings.json`.

For example, adding two dotfiles named `config` automatically chooses distinct
repository paths:

```sh
oh-my-dot add ~/.ssh/config                 # Stored as config
oh-my-dot add ~/.config/git/config         # Stored as git/config
oh-my-dot list
oh-my-dot remove git/config --keep-linked
```

These names assume neither entry was already present. Each successful add
prints its chosen name. An explicit `--as ssh/config` overrides automatic naming
when initially adding a file.

Before removal, the repository and mappings would be:

```text
files/
  config
  git/config
```

```json
{
  "config": "~/.ssh/config",
  "git/config": "~/.config/git/config"
}
```

## Behavior

- Without `--as`, add first tries the filename, then prepends up to three nearest parent directory names when needed. It stops before the user's home directory or filesystem root. Existing entries are never renamed.
- Naming is identical in terminal and noninteractive use, and the chosen key is printed. Copy/move operations use the final destination path for naming.
- Bare `add` opens the file picker automatically when both stdin and stdout are terminals. Batch selections are processed in sorted path order.
- If no automatic name is available, terminal users can enter another name. Without a terminal, the command returns an error suggesting `--as`, before modifying the source or destination.
- `--no-interactive`, `CI`, or `OMDOT_NON_INTERACTIVE` suppress prompts. Explicit `--interactive` overrides the environment but requires terminal stdin and stdout. A terminal indicates input capability, not whether the caller is a person or agent.
- `--as` takes one complete repository key for one file. It does not rename the live file.
- Listing walks nested directories and displays complete keys and destinations.
- Removal accepts a complete key or an unambiguous basename. Use `./config` to explicitly select a flat entry when other entries share its basename.
- `--copy-to` and `--move-to` still choose the live destination. Repository conflicts are checked before either operation; `--force` only controls overwriting that live destination.
- Apply validates all keys and destination conflicts before creating links. Each file uses its explicit destination.
- Directories are organizational. Group selection, profiles, and platform-specific destination alternatives are not implemented. Two entries cannot be active for the same destination.

## Portable names

Store canonical relative paths with `/` separators on every OS. Reject traversal, absolute paths, backslashes, empty components, Windows reserved characters/names, Git metadata components (`.git` and `git~1`, regardless of case), and trailing dots/spaces. New entries cannot differ only by case or collide with an existing file/directory prefix. Repository file paths cannot traverse symlinks, and add accepts regular source files only.

This does not make the destination portable automatically. A Windows destination must be resolved on Windows; foreign absolute destinations are rejected on other systems. Path length limits still depend on the host filesystem. Hard-link creation still requires source and repository to be on the same filesystem, as before.

Older binaries are not safe tools for managing nested repositories: their list/remove commands assume flat names. Use a version with nested-file support on every machine that manages such a repository. Existing portable, regular flat entries remain supported without migration.

## Isolated configuration

Set `OH_MY_DOT_CONFIG` to an absolute JSON configuration file to use an independent repository without changing the default configuration or shell profile:

```sh
OH_MY_DOT_CONFIG=/path/to/other/config.json oh-my-dot list
```

## Follow-ups

Group activation (#7), shell feature presets (#64), explicit target roots per OS, repository-path rename/migration, and transactional rollback for failures after filesystem operations are separate work. Preflight checks prevent known naming and destination conflicts before copy/move operations. They do not provide atomic rollback across source, repository, metadata, and commit writes.
