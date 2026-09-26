# Steel thread

The first implementation thread proves one domain path end-to-end before adding GitHub or Atlantis adapters.

## Path

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

The mock contains one multi-root review with changes, findings, and an existing approval.

## Acceptance

The thread is complete when:

1. the Go process starts with the mock `ReviewStore`;
2. a review can be fetched through the public API boundary;
3. the TypeScript UI renders roots, changes, findings, and existing decisions from that API;
4. no frontend fixture imports remain;
5. replacing the mock store does not change application/domain code;
6. GitHub and Atlantis types do not appear in the domain package.

Approval mutation, graph traversal, logs, plan history, persistence, GitHub, and Atlantis are intentionally subsequent threads.

## Temporary HTTP bridge

The first commit may expose a JSON `/api/reviews/{id}` bridge while generated Connect code is wired. The protobuf service is the contract and the bridge is not a second permanent API. This keeps the thread runnable without committing generated code by hand.
