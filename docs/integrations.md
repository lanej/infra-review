# Integration boundaries

Statecraft owns infrastructure reviews, roots, immutable plan sets, findings, and
human approval. Source control supplies code/reviewer evidence; the execution
system supplies command results. Provider SDKs and wire DTOs stay in
`internal/adapters`; `internal/domain`, `internal/ports`, services, protobuf, and
TypeScript contracts use Statecraft types.

Research checked 2026-09-25 against GitHub's REST documentation, go-github v92.0.0,
and Atlantis v0.48.0. The adapters are implemented and tested with local HTTP
servers. The executable still composes the mock `ReviewStore`: no production
credentials, GitHub writes, or Atlantis plan/apply calls are enabled by this PR.

## Client choices and compatibility

| System | Choice | Rationale |
| --- | --- | --- |
| GitHub | `github.com/google/go-github/v92` v92.0.0 | GitHub lists this maintained Go client under third-party libraries. GitHub has no official Go REST SDK in its library catalog. Use the typed, paginated client rather than maintaining another REST implementation. It is not an official Google product either. |
| Atlantis | A narrow `net/http` client with adapter-local DTOs | The upstream documentation and release expose no official standalone Go client. Importing Atlantis's server/controller/model packages would couple Statecraft to its server implementation. The small documented HTTP surface is the better-supported integration contract, though it remains alpha. |

GitHub's [library catalog](https://docs.github.com/en/rest/using-the-rest-api/libraries-for-the-rest-api)
and [go-github release](https://github.com/google/go-github/releases/tag/v92.0.0)
are the client references. v92 requires Go 1.26; this repository raises its minimum
accordingly. The SDK currently defaults to REST API `2022-11-28` and selects
`2026-03-10` for migrated methods; leave version negotiation with the pinned SDK,
rather than forcing the newest documentation version over older response models.
See its [versioned source](https://github.com/google/go-github/blob/v92.0.0/github/github.go)
and [module requirements](https://github.com/google/go-github/blob/v92.0.0/go.mod).

Atlantis's [API documentation](https://www.runatlantis.io/docs/api-endpoints)
explicitly labels the endpoints alpha. Upgrade Atlantis only after checking the
release's request/response shapes against these adapter contract tests. v0.48.0
is the inspected server baseline, not a guarantee of compatibility with every
Atlantis deployment. Drift APIs have a different envelope and are outside these
ports; neither drift remediation nor lock mutation is exposed here.

## GitHub: source-change and review evidence

`SourceControl` is an outbound port. Its implementation accepts a configured
`*github.Client`; authentication, GitHub Enterprise URLs, HTTP timeouts, and token
refresh belong at application composition, outside the domain.

| Port operation | REST operation | Internal mapping |
| --- | --- | --- |
| `GetChange` | `GET /repos/{owner}/{repo}/pulls/{number}` | PR number, title/body, author, draft, head SHA/ref, base ref, and URL become `SourceChange`. A merged PR is `merged`, rather than merely `closed`. |
| `ListChangedFiles` | `GET .../pulls/{number}/files` | Filename/previous filename, change status, additions/deletions/count become `ChangedFile`. These are source files, not infrastructure `Change` objects. |
| `ListReviewDecisions` | `GET .../pulls/{number}/reviews` | Submitted reviews become `ExternalReviewDecision`: external ID, actor, decision, commit, timestamp, body, URL, and provenance. Pending drafts are omitted; dismissed decisions and old commit SHAs remain visible evidence. |
| `PublishDecision` | `POST .../pulls/{number}/reviews` | An explicit commit and supported human decision map to `APPROVE`, `REQUEST_CHANGES`, or `COMMENT`. |
| `PublishStatus` | `POST /repos/{owner}/{repo}/check-runs` | A `StatusReport` becomes a check name, head SHA, status/conclusion, summary, and details URL. This creates a check run, not a legacy commit status or a human approval. |

List operations follow GitHub's pagination links with a page size of 100. The files
endpoint caps results at 3,000; the adapter checks PR file totals when it hits that
limit and errors when completeness cannot be established. It does not silently
turn a truncated list into a complete review. API and transport errors propagate
with operation context; write operations are not retried automatically.

Sources: [pull requests and files](https://docs.github.com/en/rest/pulls/pulls),
[reviews](https://docs.github.com/en/rest/pulls/reviews),
[checks](https://docs.github.com/en/rest/checks/runs), and
[pagination](https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api).

### Identity and approval ownership

Use an installation token for repository reads and Statecraft checks, with the
required repository permissions (`Pull requests: read`, or `Checks: write` for
publishing). Publishing a human review requires the signed-in reviewer's GitHub App
user access token and `Pull requests: write`. A bot's review cannot stand in for
that person's approval. Credential acquisition/refresh and authorization are still
future composition work. See [GitHub App user authentication](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user).

GitHub reviews bind to commits; Statecraft approvals bind to exact plan sets. A
GitHub review does not prove which Terraform/OpenTofu plan the reviewer saw.
Outbound reviews can include an HTML `statecraft-plan-set` marker for audit
correlation, but review bodies are editable, untrusted text. The adapter does not
recover an authoritative `PlanSetID` from that marker.

`SourceReviews.Load` assembles a provider-neutral snapshot. `MergeSourceSnapshot`
updates source metadata and `Review.SourceDecisions` while preserving Statecraft's
`Review.Decisions`, infrastructure roots, changes, and findings. Source history
never becomes a Statecraft approval or refreshes a stale approval. Only an
authenticated, persisted Statecraft decision can establish that binding. Source
history stays internal until it has a distinct public API/UI; it is omitted from
the temporary JSON review response and is not added to protobuf's approval list.

When an existing head changes or becomes unavailable, the review and its roots
become `stale`. Retained changes, findings, and decisions are historical evidence;
refreshing source metadata alone cannot restore readiness. The UI shows stale
evidence explicitly and counts only approval records bound to the displayed
commit and a plan set, excluding stale or unbound records. Other decision kinds
retain their own labels. This presentation count is not plan-set authorization.

## Atlantis: execution commands and notifications

The same adapter implements the `Planner` and `Executor` ports. It authenticates
with `X-Atlantis-Token`; the server must configure `api-secret` to enable command
endpoints. The adapter owns HTTP DTOs, not Atlantis Go server types.

| Statecraft input/output | Atlantis wire shape |
| --- | --- |
| Repository, ref, base branch, source-change number | `Repository` (`owner/repo`), `Ref`, `base_branch`, `PR`; this adapter fixes `Type` to `Github`. |
| Named `RootSelector` | `PlannerRef` maps to a `Projects` entry. |
| Directory/workspace `RootSelector` | A `Paths` entry selects the repository-relative directory and workspace. The default workspace is `default`. |
| `PlanRun` / `ApplyRun` | Legacy top-level `Error`, `Failure`, `ProjectResults`, and plan-discard information; not the newer drift envelope. |
| Per-root plan result | `PlanSuccess.TerraformOutput` becomes textual output; `Error`/`Failure` determine failed status. |
| Per-root apply result | `ApplySuccess` becomes textual output; errors and failures remain structured attempt evidence. |
| Policy result during planning | A distinct `PlanAttempt.Phase` avoids mistaking a policy-check result for another generated plan. |

The adapter requires explicit targets and rejects ambiguous or duplicate selectors.
Directory selectors containing `*`, `?`, or `[` are rejected before normalization
or HTTP: Atlantis can expand these into multiple projects, while one Statecraft
root must select one literal directory/workspace pair or a named project.
Mixed named-project and directory/workspace selection is intentionally rejected to
avoid duplicate execution and version-dependent selection behavior. A caller can
normalize its configuration to named projects or exact directory/workspace pairs.
`RootFromProject` maps an already-decoded configuration project into a root; it does
not fetch repositories or parse `atlantis.yaml`. Root identity is scoped to the
repository, and Statecraft-supplied root IDs survive the response mapping.

Atlantis's command result can contain both successful and failed projects. A
non-null `Error` (including `{}`, the serialized Go error shape) is failure
evidence even when no message survives JSON serialization. HTTP 500 with a valid
project-failure result retains those attempts. Aggregate failure and discarded
plans also remain visible; a caller must inspect run metadata and attempts rather
than treating a nil transport error as execution success. Missing results and
unknown status do not prove completeness. When several named projects share a
directory/workspace pair, select by project name to preserve their distinct roots.
Setup/auth errors and invalid response
shapes are errors, not successful empty runs.

A succeeded policy phase means Atlantis cleared its policy gate, including any
approved exception. It is not Statecraft approval; testing only a nested `Passed`
boolean would misclassify approved exceptions. See the
[policy runner](https://github.com/runatlantis/atlantis/blob/v0.48.0/server/events/project_command_runner.go).

The precise controller behavior and models come from the inspected
[v0.48.0 API controller](https://github.com/runatlantis/atlantis/blob/v0.48.0/server/controllers/api_controller.go)
and [command models](https://github.com/runatlantis/atlantis/blob/v0.48.0/server/events/command/result.go).
Root configuration is described in [repo-level configuration](https://www.runatlantis.io/docs/repo-level-atlantis-yaml).

### Apply is not approval enforcement

The API apply operation runs a fresh plan phase before applying. It does not accept
a Statecraft plan digest or promise to apply the artifact that a human approved.
Therefore `Executor.Apply` is only a command capability: the mock runtime does not
expose it, and production approval/apply must wait for artifact identity,
authorization, and plan-set reconciliation. Do not retry an uncertain apply
response automatically; the infrastructure operation may already have run.

`DecodeApplyWebhook` maps repository, PR/head/base, actor, directory/workspace,
project identity, and success to `ApplyNotification`. Decoding is not sender
authentication or replay protection, and no public webhook endpoint is installed.
A future receiver must authenticate the request before decoding/persisting it. The
legacy HTTP notification uses a configured custom header, not GitHub's inbound
webhook signature scheme. See [Atlantis apply notifications](https://www.runatlantis.io/docs/sending-notifications-via-webhooks).

## Evidence that Statecraft still needs

Textual plan/apply output is not structured resource changes, immutable plan
identity, or a durable history. Neither adapter invents findings, graph edges,
plan-set digests, execution timestamps, or approvals from those fields.

The documented Atlantis command API has no historical plan-JSON retrieval
operation. Production needs an explicit workflow-to-Statecraft evidence ingestion
path: capture structured plan JSON, exact binary-plan/artifact digests, commit and
root identity, logs, and lifecycle timestamps before workspaces disappear. Atlantis
[custom workflows](https://www.runatlantis.io/docs/custom-workflows) expose
`$PLANFILE`/`$SHOWFILE` for this integration. Define and persist that ingestion
contract in a subsequent thread instead of scraping comments or the Atlantis UI.
