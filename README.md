# infra-review

Prototype for a visual review layer over Atlantis/OpenTofu plans.

## Run

Serve the repository root with any static HTTP server:

~~~sh
python3 -m http.server 8080
~~~

Then open http://localhost:8080.

## Inputs

The UI is generated from:

- `fixtures/state.json` — mock current OpenTofu state
- `fixtures/plan.json` — mock JSON plan
- `app.js` — normalizes changes, derives lightweight risk hints, and builds dependency edges

## Prototype goals

- Make destructive and security-sensitive changes obvious.
- Show before/after values without reading raw plan output.
- Use a focused dependency graph rather than a whole-infrastructure hairball.
- Keep evidence traceable to state/plan inputs.
- Keep Atlantis/GitHub authoritative; this is initially a read-only review surface.

This is deliberately dependency-free. If the interaction model holds up, the next step is replacing the fixture inputs with Atlantis post-plan artifacts and moving the view into EasyUI components.


## Product direction

The prototype is evolving into the primary review, diagnosis, and human approval interface for infrastructure changes attached to GitHub pull requests.

- [Product definition and jobs to be done](./docs/product.md)
- [Architecture and domain model](./docs/architecture.md)
