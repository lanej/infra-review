# Steel thread

The first implementation thread proves one domain path end-to-end while establishing
the real external adapter boundaries.

## Runnable path

```text
TypeScript UI
    |
    | ReviewService / Connect boundary
    v
Go application service
    |
    | ReviewStore port
    v
Mock in-memory adapter
```

The mock contains one multi-root review with changes, findings, and an existing
approval.

## External adapter shape

PR #5 also establishes, but does not yet wire into the runtime:

```text
SourceControl port -> GitHub adapter
Planner port       -> Atlantis adapter
Executor port      -> Atlantis adapter
Atlantis webhook   -> ApplyNotification
```

These adapters translate external API DTOs into Statecraft domain types. GitHub and
Atlantis packages do not appear in the domain package.

## Acceptance

The runnable mock thread is complete when:

1. the Go process starts with the mock `ReviewStore`;
2. a review can be fetched through the public API boundary;
3. the TypeScript UI renders roots, changes, findings, and existing decisions from
   that API;
4. no frontend fixture imports remain;
5. replacing the mock store does not change application/domain code;
6. GitHub and Atlantis types do not appear in the domain package.

The external-boundary thread is complete when:

1. GitHub pull request metadata/files/reviews map through `SourceControl`;
2. Statecraft review decisions map back to GitHub review events;
3. Statecraft status can be published as a GitHub check;
4. Atlantis plan/apply requests map from domain root selectors;
5. Atlantis per-project results map to structured plan/apply attempts even when
   Atlantis returns HTTP 500 for project errors;
6. Atlantis apply webhook payloads map to provider-neutral lifecycle events.

The adapters are deliberately **not** wired into production execution in this PR.
Plan evidence ingestion, historical logs, plan-set persistence, GitHub App auth,
root discovery, graph traversal, and approval mutation in the UI are subsequent
threads.

## Temporary HTTP bridge

The first runnable commit exposes a JSON `/api/reviews/{id}` bridge while generated
Connect code is wired. The protobuf service is the intended public contract and the
bridge is not a second permanent API.
