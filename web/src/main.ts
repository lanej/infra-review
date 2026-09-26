import "./style.css";
import type { Action, Command, Review } from "./contracts";
import { escapeHTML, initialUI, renderReview, type View } from "./review";
const app = document.querySelector<HTMLElement>("#app")!;
const ui = initialUI();
let review: Review | null = null;
const sessionKey = "statecraft-demo-session-v1";
class RequestError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}
async function request(path: string, body?: unknown): Promise<Review> {
  const response = await fetch(path, {
    method: body ? "POST" : "GET",
    headers: body ? { "Content-Type": "application/json" } : {},
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(30000),
  });
  const value = await response.json();
  if (!response.ok)
    throw new RequestError(
      value.error || "The demo request failed.",
      response.status,
    );
  return value as Review;
}
function render(focus?: string) {
  if (!review) return;
  const active = app.contains(document.activeElement)
    ? (document.activeElement as HTMLElement)
    : null;
  const selection =
    active instanceof HTMLInputElement
      ? [active.selectionStart, active.selectionEnd]
      : null;
  const restore = active?.id
    ? `#${CSS.escape(active.id)}`
    : active?.dataset.ui
      ? `[data-ui="${CSS.escape(active.dataset.ui)}"]${active.dataset.value !== undefined ? `[data-value="${CSS.escape(active.dataset.value)}"]` : ""}${active.dataset.action ? `[data-action="${CSS.escape(active.dataset.action)}"]` : ""}`
      : undefined;
  const restoreIndex =
    restore && active
      ? Array.from(app.querySelectorAll(restore)).indexOf(active)
      : 0;
  const expanded = new Set(
    Array.from(
      app.querySelectorAll("details[open] summary"),
      (el) => el.textContent,
    ),
  );
  app.innerHTML = renderReview(review, ui);
  for (const detail of app.querySelectorAll("details"))
    if (expanded.has(detail.querySelector("summary")?.textContent ?? ""))
      detail.open = true;
  const next = focus || restore;
  const target = next
    ? app.querySelectorAll<HTMLElement>(next)[
        focus ? 0 : Math.max(0, restoreIndex)
      ]
    : null;
  target?.focus({ preventScroll: true });
  if (
    target instanceof HTMLInputElement &&
    selection?.[0] != null &&
    selection[1] != null
  )
    target.setSelectionRange(selection[0], selection[1]);
}
async function create(scenario: string) {
  if (ui.busy) return;
  ui.busy = true;
  ui.error = "";
  render();
  try {
    review = await request("/api/demos", { scenario });
    Object.assign(ui, initialUI());
    ui.view =
      scenario === "partial"
        ? "execution"
        : scenario === "expired"
          ? "policies"
          : "changes";
    try {
      sessionStorage.setItem(sessionKey, review.id);
    } catch {
      /* Optional storage. */
    }
  } catch (error) {
    ui.error = (error as Error).message;
  } finally {
    ui.busy = false;
    if (review) render();
    else
      app.innerHTML = `<div class="load-error" role="alert"><h1>Could not load the mock workspace</h1><p>${escapeHTML(ui.error)}</p><button class="button" data-ui="retry">Try again</button></div>`;
  }
}
async function act() {
  if (!review || !ui.form || ui.busy) return;
  const form = ui.form,
    cmd: Command = {
      action: form.action,
      expectedVersion: review.version,
      violationId: form.violationId,
      reason: form.reason,
      evidence: form.evidence,
    };
  ui.busy = true;
  ui.error = "";
  render();
  try {
    review = await request(
      `/api/demos/${encodeURIComponent(review.id)}/actions`,
      cmd,
    );
    ui.form = null;
    ui.notice =
      "Demo action completed. No live infrastructure or GitHub review was changed.";
    if (cmd.action === "apply" || cmd.action === "verify")
      ui.view = "execution";
    else if (cmd.action === "request_changes") ui.view = "history";
    else if (cmd.action.includes("acceptance")) ui.view = "policies";
    else if (cmd.action === "plan") {
      ui.view = "changes";
      ui.root = "all";
      ui.query = "";
    }
  } catch (error) {
    ui.error = (error as Error).message;
    if (error instanceof RequestError && [409, 422].includes(error.status)) {
      try {
        review = await request(`/api/demos/${encodeURIComponent(review.id)}`);
      } catch {
        /* Preserve draft and original error. */
      }
    }
  } finally {
    ui.busy = false;
    render(
      ui.form
        ? ".command-panel textarea, .command-panel button[type=submit]"
        : ".heading-actions button",
    );
    app
      .querySelector<HTMLElement>("[role=alert],[role=status]")
      ?.scrollIntoView({ block: "nearest" });
  }
}
app.addEventListener("click", (event) => {
  const target = (event.target as Element).closest<HTMLElement>("[data-ui]");
  if (!target) return;
  const action = target.dataset.ui,
    value = target.dataset.value ?? "";
  if (action === "retry") {
    void create("review");
    return;
  }
  if (!review || ui.busy) return;
  if (action === "reset") {
    void create(review.scenario);
    return;
  }
  if (action === "view") {
    ui.view = value as View;
    ui.notice = "";
  }
  if (action === "select" || action === "inspect") {
    ui.selected = value;
    if (action === "inspect") {
      ui.view = "changes";
      ui.root = "all";
      ui.query = "";
    }
  }
  if (action === "violation") ui.view = "policies";
  if (action === "root") {
    ui.root = value;
    ui.query = "";
  }
  if (action === "detail") ui.detail = value;
  if (action === "cancel") ui.form = null;
  if (action === "command") {
    ui.form = {
      action: target.dataset.action as Action,
      violationId: value,
      reason: "",
      evidence: "",
    };
    ui.error = "";
    ui.notice = "";
  }
  render(
    action === "command"
      ? ".command-panel textarea, .command-panel button[type=submit]"
      : undefined,
  );
  if (action === "command")
    app.querySelector(".command-panel")?.scrollIntoView({ block: "nearest" });
  if (action === "inspect")
    app.querySelector(".inspector")?.scrollIntoView({ block: "nearest" });
});
app.addEventListener("change", (event) => {
  if ((event.target as HTMLElement).id === "demo-scenario")
    void create((event.target as HTMLSelectElement).value);
});
app.addEventListener("input", (event) => {
  const input = event.target as HTMLInputElement;
  if (input.id === "resource-search") {
    ui.query = input.value;
    render("#resource-search");
  }
  if (input.id === "decision-reason" && ui.form) ui.form.reason = input.value;
  if (input.id === "decision-evidence" && ui.form)
    ui.form.evidence = input.value;
});
app.addEventListener("submit", (event) => {
  if ((event.target as HTMLElement).id === "decision-form") {
    event.preventDefault();
    void act();
  }
});
async function start() {
  let id: string | null = null;
  try {
    id = sessionStorage.getItem(sessionKey);
  } catch {
    /* Optional storage. */
  }
  if (id) {
    try {
      review = await request(`/api/demos/${encodeURIComponent(id)}`);
      render();
      return;
    } catch {
      /* Restarted servers lose demo sessions. */
    }
  }
  await create("review");
}
void start();
setInterval(async () => {
  if (
    !review ||
    ui.busy ||
    ui.form ||
    document.hidden ||
    (app.contains(document.activeElement) &&
      document.activeElement?.matches("input,select,textarea"))
  )
    return;
  try {
    const id = review.id,
      fresh = await request(`/api/demos/${encodeURIComponent(id)}`);
    if (
      !ui.busy &&
      !ui.form &&
      review.id === id &&
      fresh.version >= review.version &&
      JSON.stringify(fresh) !== JSON.stringify(review)
    ) {
      review = fresh;
      render();
    }
  } catch {
    /* Commands revalidate state. */
  }
}, 15000);
