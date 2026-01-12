# oh-my-dot Documentation

This directory contains technical documentation and specifications for oh-my-dot.

## Specifications

### [Shell Framework](specs/shell-framework/)

Complete specification for the integrated shell framework feature.

**Status**: 📝 Specification Complete - Ready for Implementation

**Overview**: Seamless shell profile integration with modular feature management. Users can add shell features (aliases, prompts, completions) that are automatically applied alongside their dotfiles.

**Key Features**:
- Unified `omdot apply` command for dotfiles and shell hooks
- Auto-initialization and auto-cleanup
- Multi-shell support (bash, zsh, fish, PowerShell, POSIX)
- Lazy loading strategies (eager, defer, on-command)
- Smart shell selection based on feature compatibility

**Documents**:
1. `00-quick-reference.md` - Start here for quick overview
2. `01-design.md` - Primary specification and complete design
3. `02-decisions.md` - All design decisions finalized
4. `03-implementation-roadmap.md` - Detailed 8-phase roadmap (14-17 weeks)
5. `04-code-examples.md` - Implementation templates and patterns
6. `05-code-validation.md` - Go code verification

**Timeline**:
- MVP: 8 weeks (basic feature management)
- Full: 14-17 weeks (all features including lazy loading)

---

## Contributing Documentation

When adding new features or specifications:

1. Create a new directory under `specs/<feature-name>/`
2. Add a `README.md` with overview and document index
3. Break large specs into focused documents
4. Include code examples and validation
5. Document all design decisions
6. Provide implementation roadmap

## Document Structure

```
docs/
├── README.md                          # This file
└── specs/
    └── <feature-name>/
        ├── README.md                  # Feature overview
        ├── 00-quick-reference.md      # Quick start guide
        ├── 01-design.md               # Main specification
        ├── 02-decisions.md            # Design decisions
        ├── 03-implementation-roadmap.md
        ├── 04-code-examples.md
        └── 05-*.md                    # Additional docs as needed
```

## Quick Links

- [Shell Framework Quick Reference](specs/shell-framework/00-quick-reference.md)
- [Shell Framework Design](specs/shell-framework/01-design.md)
- [Main README](../README.md)