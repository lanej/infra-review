# Integration boundaries

Statecraft treats GitHub and Atlantis as adapters around its own change-review domain.

This document records the external shapes we currently depend on and, equally
important, the data those systems do **not** provide through stable APIs.

## GitHub

GitHub provides source-change identity, code history, reviewers, human review
decisions, and commit-level status surfaces.

### Client

The Go adapter uses `github.com/google/go-github/v72`.

This is a mature community client rather than an official GitHub SDK. The GitHub
REST API remains the contract of record. Version 72 is intentionally pinned because
it supports the repository's Go 1.23 toolchain.

### Port mapping

| Statecraft | GitHub |
| --- | --- |
| `SourceControl.GetChange` | Get pull request |
| `SourceControl.ListChangedFiles` | List pull request files |
| `SourceControl.ListReviewDecisions` | List pull request reviews |
| `SourceControl.PublishDecision` | Create pull request review |
| `SourceControl.PublishStatus` | Create check run |

GitHub-specific objects are translated inside
`internal/adapters/github`. Application and domain packages do not import the
GitHub client.

### Human identity

Statecraft approval should remain attributable to the human who made the decision.

A GitHub App installation token is appropriate for repository reads and Statecraft
check runs. When Statecraft submits a human `APPROVE`, `REQUEST_CHANGES`, or
`COMMENT` review on behalf of the signed-in reviewer, the adapter should be
constructed with that reviewer's GitHub App user access token.

The port deliberately accepts an authenticated adapter rather than token concepts.
Credential selection and token refresh belong in the GitHub adapter/application
composition layer, not in the review domain.

### Plan-set binding

GitHub binds a review to a commit SHA, while Statecraft approvals bind to a
`PlanSet`, which is more specific.

Until Statecraft owns a durable synchronization record, the GitHub adapter writes a
non-rendered marker into the review body:

```html
<!-- statecraft-plan-set:planset-7 -->
```

Statecraft must still persist the decision itself. The marker is synchronization
metadata and an audit aid, not the authoritative database.

## Atlantis

Atlantis provides Terraform/OpenTofu planning and application execution.

### API stability

Atlantis documents `POST /api/plan` and `POST /api/apply` as alpha APIs. Statecraft
therefore owns a narrow HTTP adapter instead of importing Atlantis server packages.

The adapter's wire DTOs mirror only fields Statecraft currently needs.

### Command shape

Statecraft maps:

```text
PlanRequest / ApplyRequest
  repository
  ref
  base branch
  pull request
  roots[]
      project name OR directory/workspace

        |
        v

Atlantis APIRequest
  Repository
  Ref
  base_branch
  Type = Github
  PR
  Projects[] / Paths[]
```

A Statecraft `RootSelector` maps to an Atlantis project name when one exists.
Otherwise the stable identity is the repo-relative directory and Terraform
workspace.

Atlantis project identity is therefore an adapter input, not Statecraft's definition
of a root.

### Result shape

Legacy plan/apply responses contain per-project results:

```text
ProjectResult
  ProjectName
  RepoRelDir
  Workspace
  Error
  Failure
  PlanSuccess.TerraformOutput OR ApplySuccess
```

These become Statecraft `PlanAttempt` and `ApplyAttempt` values.

Atlantis currently returns HTTP 500 when a command result contains project errors,
while still returning useful `ProjectResults`. The adapter preserves those results
as failed attempts instead of treating them as a transport failure.

### Apply behavior

Atlantis's API apply endpoint performs a plan phase before applying. Statecraft
should therefore not infer that an `Apply` request is a pure application of an
already-captured immutable plan. The eventual execution model must reconcile the
plan actually used by Atlantis with the `PlanSet` approved in Statecraft.

That is a critical integration invariant before production approval/apply is enabled.

### Apply webhooks

Atlantis HTTP apply webhooks provide repository, pull request, actor, project,
directory/workspace, head/base identity, and success. The webhook adapter maps that
payload to `domain.ApplyNotification`.

This is useful lifecycle evidence, but it is not a replacement for persisted apply
attempts and logs.

## What Atlantis does not give Statecraft

The documented plan/apply command API is not a complete review read model.

Statecraft still needs durable access to:

- structured plan JSON;
- exact plan/artifact identity and digest;
- historical plan attempts;
- full execution logs;
- policy evidence;
- plan/apply timestamps and lifecycle events;
- evidence needed to reconstruct a review after Atlantis workspaces are cleaned up.

Atlantis custom workflows expose plan-related files, including `$PLANFILE` and
`$SHOWFILE`, and can produce JSON plan output. The likely production integration is
therefore an explicit evidence-ingestion path from the Atlantis workflow into
Statecraft rather than scraping Atlantis's web UI.

The exact ingestion protocol is a subsequent steel thread.

## Boundary summary

```text
                 Statecraft domain
              /                    \
     SourceControl                 Planner / Executor
          |                              |
       GitHub                          Atlantis
          |                              |
 PRs / reviews / checks       plan/apply command API
                                        |
                              workflow evidence + webhooks
                                        |
                                        v
                             Statecraft evidence store
```

GitHub answers who/what code is changing and carries human review decisions.
Atlantis executes infrastructure workflows.
Statecraft owns the durable infrastructure-specific review model joining those
facts together.
