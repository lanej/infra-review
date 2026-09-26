<p align="center"><img src="./assets/statecraft-lockup.svg" alt="Statecraft" width="720"></p>

**Infrastructure change, understood: Understand, debug, and approve infrastructure changes.**

Statecraft is an experimental workbench for infrastructure changes attached to GitHub pull requests. It is intended to become the primary place to work a change from planning through verification: understand what will happen across multiple infrastructure roots, investigate the dependency graph, assess policy and operational risk, diagnose Atlantis failures, collaborate with reviewers, and approve the exact plan being applied.

> **Status:** early prototype. A TypeScript → Go mock-backed steel thread and initial GitHub/Atlantis adapters now exist, but the production integrations are not wired into the runtime. Durable plan evidence/history, Connect-generated handlers, GitHub App authentication, and authenticated approval flows remain to be implemented.

**[Open the prototype](https://lanej.io/infra-review/)** · **[Product definition](./docs/product.md)** · **[Architecture](./docs/architecture.md)** · **[Integrations](./docs/integrations.md)**

## Change lifecycle

```text
Plan -> Diagnose -> Understand -> Assess -> Discuss -> Approve -> Apply -> Verify
  ^         |            |          |          |                    |       |
  |         +---- fix ---+----------+----------+--------------------+       |
  |                                                                      |
  +--------------------------- Revert <-----------------------------------+
                                  |
                                  +----> Plan -> Assess -> Approve -> Apply -> Verify
```

A revert is another infrastructure change, not a privileged undo operation. It should be planned, assessed, approved, applied, and verified through the same controls as a forward change.

## Jobs to be done

Statecraft is organized around the work required to safely move an infrastructure change through its lifecycle.

| Job | What the workbench should make possible |
| --- | --- |
| **Plan** | See every affected infrastructure root, its planning state, and whether the complete proposal is ready for review. |
| **Diagnose** | Understand why a plan or apply failed, with structured errors linked to the relevant Atlantis logs and resource context. |
| **Understand** | Quickly see what changes across roots, modules, resources, and properties without reconstructing the change from raw plan output. |
| **Assess** | Identify destructive, security, availability, data, topology, and cost risks and trace every finding back to evidence. |
| **Explore impact** | Traverse a large directed resource graph to understand dependencies, blast radius, and cross-root relationships. |
| **Discuss** | Attach questions and concerns to resources, property changes, relationships, findings, or the review itself. |
| **Approve** | See other reviewers and make a human approval or request-changes decision bound to the exact plan set reviewed. |
| **Apply** | Follow an approved change into Atlantis execution without losing the review context. |
| **Verify** | Distinguish a successful command from evidence that the intended resulting infrastructure state was actually realized. |
| **Revert** | Prepare a reverse change, understand its consequences, and send it through the normal safety lifecycle. |
| **Understand history** | Inspect and compare prior plans, attempts, findings, approvals, invalidations, applies, and reverts. |

The detailed statements and product invariants live in [docs/product.md](./docs/product.md).

## Assumed workflow

A GitHub pull request is the unit of work. One pull request may affect many independently planned infrastructure roots.

1. **Discover roots.** Statecraft determines which roots are affected and tracks each independently.
2. **Plan.** Atlantis produces plans and logs. Failed roots stay visible and actionable rather than disappearing into PR comments.
3. **Assemble a PlanSet.** Successful root plans form one immutable proposal identified by the Git commit and root-plan digests.
4. **Review.** The workbench presents a semantic change hierarchy, directed resource graph, findings, evidence, execution history, and reviewer state.
5. **Discuss and revise.** Concerns attach to infrastructure objects. A new commit or plan creates a new proposal and makes affected prior approvals visibly stale.
6. **Approve.** Human decisions happen in Statecraft and bind to the exact PlanSet reviewed. A future GitHub App will synchronize those decisions while preserving reviewer identity.
7. **Apply.** Atlantis remains the execution engine. Plan and apply attempts, errors, and logs stay part of the same change history.
8. **Verify.** The workbench records whether the intended resulting state was observed rather than treating exit code zero as sufficient proof.
9. **Revert when necessary.** A revert references the prior change but produces a new PlanSet and follows the same assessment and approval path.

Automatic approval is deliberately outside the product boundary. External policy or GitHub automation may decide that a human review is unnecessary; Statecraft does not make that decision itself.

## The workbench

The product should feel like one investigative environment rather than a collection of dashboards.

- **Overview** — lifecycle state, root completeness, reviewers, approvals, highest-risk changes, findings, and execution state.
- **Changes** — hierarchical root/module/resource/property changes with semantic before/after evidence.
- **Graph** — a directed graph browser designed for hundreds or thousands of resources, with neighborhood expansion, upstream/downstream traversal, path finding, grouping, filtering, and collapsed unchanged subgraphs.
- **Findings** — policy, security, reliability, destructive-operation, and cost concerns tied directly to resources and evidence.
- **Execution** — plan/apply attempts, extracted failures, searchable Atlantis logs, and verification state.
- **History** — immutable plan versions, approvals, invalidations, execution attempts, comparisons, and revert lineage.

A resource selected from a finding, graph node, change row, or execution error should resolve to the same underlying resource and evidence.

## Architecture

The intended implementation separates a **TypeScript frontend** from a **Go + Connect backend**.

```text
GitHub                           Atlantis
  |                                |
  | PRs, commits, identity         | plans, applies, logs
  | reviews, checks                |
  +---------------+----------------+
                  |
                  v
        +----------------------+
        | Statecraft API  |
        | Go + Connect         |
        +----------+-----------+
                   |
          normalized domain
                   |
                   v
        +----------------------+
        | TypeScript frontend  |
        | review workbench     |
        +----------------------+
```

The backend follows a **hexagonal architecture**. GitHub and Atlantis are the first adapters, not the domain model. Core behavior works in terms of repositories, reviews, roots, plan sets, execution attempts, resources, changes, relationships, findings, discussions, decisions, verification, and reverts.

The frontend should never need to understand raw Atlantis objects or use Terraform/OpenTofu JSON as its application model.

See [docs/architecture.md](./docs/architecture.md) for the initial domain and service boundaries.

## Future: assisted diagnosis and remediation

There is enough evidence in this lifecycle to eventually make diagnosis substantially more useful: failed plans and applies, Atlantis logs, resource relationships, policy findings, Git history, prior successful fixes, and organization-specific skill files.

A future assisted workflow could use that history plus LLM reasoning to:

1. diagnose likely causes;
2. find relevant prior fixes;
3. suggest a concrete remediation;
4. prepare a source change for human review.

That is intentionally out of scope now. The near-term requirement is to retain enough structured evidence and history that this capability can be added later without redesigning the system.

## Current prototype

The deployed prototype is deliberately dependency-free and uses synthetic data:

- `fixtures/state.json` — mock current OpenTofu state;
- `fixtures/plan.json` — mock JSON plan;
- `app.js` — normalizes changes, derives lightweight risk hints, and builds dependency edges.

Run it locally with:

```sh
python3 -m http.server 8080
```

Then open `http://localhost:8080`.

The prototype exists to validate the review interaction model. The implementation now has a mock-backed TypeScript/Go steel thread plus GitHub and Atlantis adapters behind domain ports. The next steps are generated Connect wiring and durable plan/evidence ingestion.

## Design principles

- **Consequences before syntax.** Lead with what the change does; retain raw evidence underneath.
- **Complete or visibly incomplete.** Missing, failed, pending, and stale roots cannot look reviewed.
- **Plans are immutable review objects.** Approval binds to an exact PlanSet.
- **Graphs are for investigation.** Never require a reviewer to comprehend a whole-infrastructure hairball.
- **Failures are first-class.** Planning and application errors belong beside the change they prevented or interrupted.
- **Evidence survives abstraction.** Summaries, findings, and future AI assistance always lead back to source evidence.
- **Integrations stay at the boundary.** GitHub and Atlantis are adapters behind explicit ports.


## Development

The current steel thread separates the TypeScript frontend from a Go backend and
uses a mock adapter behind the domain port.

The Go module is `github.com/lanej/statecraft` and the frontend package is
`statecraft`. Use Go 1.26 or later (required by the pinned GitHub client) and
Node.js 22 or later for development.

```sh
# terminal 1
make api

# terminal 2
cd web
npm ci
npm run dev
```

The browser loads review `pr-1842` through the backend rather than importing
fixture JSON. See [docs/steel-thread.md](./docs/steel-thread.md).

The protobuf/Connect contract lives in `proto/statecraft/v1/review.proto`.
Run `make generate` with Buf installed to generate Go and TypeScript bindings;
the runnable thread retains a temporary JSON bridge until those generated handlers
are committed.
