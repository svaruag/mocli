# ADR-0004: OneDrive Command Surface and Limitations

- Status: accepted
- Date: 2026-02-19

## Context

Mocli MVP shipped with mail/calendar/tasks. Agent workflows also require file operations in Microsoft ecosystem, so OneDrive support is needed.

The requested scope includes:

- list/search/upload/download files
- organize folders
- manage permissions and sharing
- list shared drives/items
- manage comments

## Decision

Add `drive` command group with this implemented surface:

- Files/folders: `ls`, `search`, `get`, `upload`, `download`, `mkdir`, `rename`, `move`, `delete`
- Permissions/sharing: `permissions`, `share`, `unshare`
- Discovery: `drives`, `shared`

And explicitly keep these limitations for now:

- `drive comments`, `drive comment add`, `drive comment delete` return `not_implemented`.
- Rationale: Microsoft Graph v1.0 does not provide general file-comments endpoints for drive items.
- `drive shared` relies on Graph `sharedWithMe`, which is deprecated by Microsoft and may degrade.

## Consequences

Positive:

- Delivers core OneDrive automation capability without bloating architecture.
- Keeps UX close to `gogcli` drive naming (`share`/`unshare`).
- Preserves JSON-first and deterministic exit-code contracts.

Negative:

- Comments workflow is intentionally unavailable until Graph support path is clear.
- Shared-with-me reliability depends on a deprecated Graph endpoint.

## Follow-ups

- Add resumable uploads for files larger than simple-upload limits.
- Revisit comments if Graph introduces stable file-comment APIs.
- Revisit `shared` command if Microsoft replaces `sharedWithMe`.
