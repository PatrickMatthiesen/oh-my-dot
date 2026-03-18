# Feature Proposal: Add a dependency provider layer for feature-managed tool installation

## Summary
This proposal aims to introduce a dependency provider layer that manages the installation of features required by the user in a streamlined manner.

## Motivation
As feature managers evolve, ensuring that dependencies are handled gracefully becomes crucial for user experience and system integrity. This layer provides a consistent approach for users across various systems.

## Goals
- Introduce a flexible dependency management approach.
- Ensure smooth installation and enablement of features.
- Provide users clear visibility into the dependency requirements.

## Non-Goals
- This project will not cover the installation of non-feature-managed tools.
- Exclusion of dependencies related to platform-specific features not covered by the tool management.

## Proposed Design
1. **Prompt UX**: Separate required and optional dependencies.
2. **Default Providers**: The initial providers will be `apt`, `dnf`, `winget`, and `brew`.
3. **Install Denial Behavior**: If a required dependency is denied, the setup for that feature will abort. Denying an optional dependency will allow the feature to continue without that dependency.
4. **Security Considerations**: Ensuring deterministic metadata-driven resolution, no silent trust expansion, exact command previews, and no arbitrary command constructions from user input.
5. **Install Provenance**: A lightweight log will be stored in an `.oh-my-dot` directory, which will be surfaced with the config command.

## Workflow Coverage
- **Feature Add**: Adding a new feature through a standardized approach.
- **Feature Enable**: Enabling features based on available dependencies.
- **Apply**: Applying changes and ensuring dependencies are resolved.
- **Doctor --fix**: A command to verify and fix issues related to dependencies.

## Suggested MVP
- Implement the core functionality for installing tools from the defined providers.
- Ensure a prompt for required and optional dependencies is in place.

## Acceptance Criteria
- Users can successfully add and enable features with required dependencies.
- The system alerts users about optional dependencies and allows overriding.

## Follow-up Sub-Issue
- Add a settings command or extend the interactive config command to allow users to view and change settings, including preferred provider with defaults set to auto and an automatic best-order fallback.