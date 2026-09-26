# Working on Statecraft

Statecraft is the infrastructure review workbench for a source change: plan,
diagnose, understand consequences, assess policies, discuss, approve, apply, verify,
and prepare a reviewed revert. GitHub supplies source identity; Atlantis executes
plans and applies; OPA/Rego will evaluate policies behind an adapter. Human
infrastructure decisions and their evidence belong in Statecraft.

## Start here

1. Inspect the current branch, working tree, and associated PR before changing
   files. Preserve existing work; do not assume a referenced PR has merged.
2. Read [the handoff](docs/handoff.md) for the implemented baseline, code map,
   runnable scenarios, and limitations.
3. Read [the roadmap](docs/roadmap.md) for feature status, dependencies, completion
   criteria, and the recommended next task. Follow the user's current scope when
   it differs from the suggested ordering.
4. Read [product intent](docs/product.md) and [the design contract](DESIGN.md)
   before changing user workflows. Read [architecture](docs/architecture.md),
   [integration boundaries](docs/integrations.md), and [policy semantics](docs/policies.md)
   for changes in those areas.

Keep this file short and operational. Product requirements belong in the product
and design docs; status and sequencing belong in the roadmap and handoff. Update
those documents when behavior, boundaries, or the next useful step changes.

## What is running

- Work in `web/` for the current TypeScript/Vite interface. Root-level `index.html`,
  `app.js`, `styles.css`, and `fixtures/` are the older static prototype.
- `cmd/statecraft` composes an isolated mock workflow. It does not invoke the live
  GitHub/Atlantis adapters. Keep the simulation explicit and separately composed.
- The runtime uses a temporary JSON bridge. Protobuf and Buf configuration exist;
  Connect handlers/clients are not generated or wired yet. Until that migration,
  keep Go JSON, protobuf, and `web/src/contracts.ts` aligned.
- Demo identities, plan digests, policies, evidence, execution, and verification
  are synthetic. Do not present them as production authorization or real results.

## Invariants to preserve

- Keep provider SDKs, wire DTOs, Rego queries, and raw engine documents inside
  adapters. Domain, ports, services, and frontend contracts use Statecraft models.
  Prefer official clients when available; document a supported alternative when
  none exists. Verify authoritative sources when changing external integrations.
- An approval binds to an exact proposal and assessment, including commit and
  complete root scope. A GitHub review or editable body marker is source evidence,
  not proof of Statecraft approval. New evidence can invalidate old decisions.
- Policy coverage, policy outcomes, violation acceptance, human approval, and
  action eligibility are distinct. Acceptance preserves the violation, is scoped
  and time bounded, and requires authorization. Missing required evaluation never
  becomes a pass. A policy saying approval is unnecessary must not create a fake
  human approval record.
- The backend revalidates scope, freshness, current requirements, and permissions
  on each mutation. UI availability is explanatory, not enforcement. Preserve
  atomic version checks and the distinction between a failed and committed action.
- Partial execution and uncertain outcomes need reconciliation. Do not blindly
  retry an apply. Success of an API call, success of a root, completion of all
  roots, resulting-state agreement, and operational health are different facts.
- The current Atlantis apply API can replan. Do not wire it to an "apply approved
  plan" promise until exact-artifact enforcement and other execution paths are
  addressed. See the [documented limitation](docs/integrations.md#apply-is-not-approval-enforcement).
- Preserve evidence and decision history. A revert is a new proposal through the
  same controls, not a privileged undo command.

## Development and verification

Prerequisites: Go 1.26+, Node.js 22.6+, npm; Buf for protobuf work. Use the versions
in `go.mod` and `web/package-lock.json` when checking compatibility.

```sh
# Repository root, terminal 1
make api

# Repository root, terminal 2
npm --prefix web ci
npm --prefix web run dev
```

The API defaults to `127.0.0.1:8081`; use Vite's displayed browser URL. If occupied,
set `STATECRAFT_PORT` for the API and the matching `STATECRAFT_API_URL` for Vite.
Do not stop an unrelated process to free a port. Sessions disappear on API restart.

Choose checks for the affected boundary:

```sh
go test ./...
# Use the race detector for store/concurrency changes.
go test -race ./...
npm --prefix web test
npm --prefix web run build
# For protobuf changes:
buf lint
buf breaking --against '.git#branch=main'
```

Use the actual target/base ref for the breaking comparison if it differs from
local `main`. `make generate` invokes Buf, but generated runtime wiring is still a
roadmap task. Do not hand-author generated files. Existing CI runs Go tests and
frontend tests/build; it does not provide browser or production-integration proof.

For workflow changes, exercise the relevant scenarios in the running UI, including
the blocked/recovery path. Check keyboard focus, narrow layout, loading/error
behavior, and the path from a summary to supporting evidence. Browser tests and
screenshots must target `web/`, not the static prototype. If edits do not appear,
restart the dev server and verify the new result before capturing evidence.

Docs-only changes need link, path, command, and consistency checks; do not run
unrelated test suites solely to populate a PR description.

## Pull requests and handoff

Keep descriptions concise: explain the final change and why, show the result, and
add only context the diff and checks do not already provide. Let tests and automated
checks report their own results; do not duplicate cases, coverage, counts, or CI
results. Mention manual validation, external evidence, or verification gaps only
when useful. Omit unused sections, boilerplate checklists, and implementation history.

UI PRs for this project include screenshots of the actual current running result.
Use durable URLs that render inline on GitHub and verify they load. Include the
relevant failure/blocked state when it explains a workflow change. Documentation-only
updates do not need new screenshots. Never use a design mockup as implementation
proof or make changes to the UI just to stage evidence.

Before handing off, record what works, what remains simulated or incomplete, the
next bounded task, and any unresolved design decisions in the appropriate docs.
Keep session-specific ports, temporary paths, credentials, and transient CI/PR
status out of permanent documentation. Do not merge a PR unless the user has
requested that action in the applicable task.
