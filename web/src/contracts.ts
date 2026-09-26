// Provider-neutral contract for the temporary JSON bridge. Mirrors review.proto.
export type Action =
  | "plan"
  | "approve"
  | "request_changes"
  | "request_acceptance"
  | "grant_acceptance"
  | "apply"
  | "verify";
export type Root = {
  id: string;
  name: string;
  status: string;
  applyStatus: string;
  log: string;
};
export type Change = {
  id: string;
  rootId: string;
  name: string;
  address: string;
  resourceType: string;
  action: string;
  risk: string;
  summary: string;
  properties: { name: string; before: string; after: string }[];
  relationships: {
    resourceId: string;
    label: string;
    kind: string;
    evidence: string;
  }[];
  planText: string;
  sourcePath: string;
  sourceText: string;
};
export type Plan = {
  id: string;
  number: number;
  commitSha: string;
  digest: string;
  rootIds: string[];
  createdAt: string;
  changeIds: string[];
};
export type Violation = {
  id: string;
  policyId: string;
  policyVersion: string;
  resourceId: string;
  title: string;
  severity: string;
  blocking: boolean;
  acceptanceAllowed: boolean;
  explanation: string;
  consequence: string;
  unknown: string;
  evidence: string[];
};
export type PolicyEvaluation = {
  id: string;
  planSetId: string;
  policySetId: string;
  status: string;
  coverage: string;
  evaluatedAt: string;
  results: { id: string; version: string; title: string; outcome: string }[];
  violations: Violation[];
};
export type Acceptance = {
  id: string;
  violationId: string;
  planSetId: string;
  evaluationId: string;
  policyVersion: string;
  status: string;
  requestedBy: string;
  authorizedBy: string;
  reason: string;
  evidence: string;
  createdAt: string;
  expiresAt: string;
};
export type Decision = {
  actor: string;
  decision: string;
  planSetId: string;
  commitSha: string;
  createdAt: string;
  externalId: string;
  source: string;
  evaluationId: string;
  message: string;
};
export type ActionDecision = {
  action: Action;
  outcome: string;
  reason: string;
};
export type Review = {
  id: string;
  repository: string;
  pullRequest: number;
  title: string;
  headSha: string;
  state: string;
  demo: boolean;
  version: number;
  scenario: string;
  roots: Root[];
  changes: Change[];
  findings: {
    id: string;
    severity: string;
    category: string;
    title: string;
    resourceAddress: string;
    blocking: boolean;
  }[];
  decisions: Decision[];
  plan: Plan | null;
  policy: PolicyEvaluation | null;
  acceptances: Acceptance[];
  attempts: {
    id: string;
    planSetId: string;
    rootId: string;
    operation: string;
    status: string;
    createdAt: string;
    log: string;
  }[];
  history: { title: string; detail: string; createdAt: string }[];
  planHistory: Plan[];
  actions: ActionDecision[];
  verification: string;
};
export type Command = {
  action: Action;
  expectedVersion: number;
  violationId?: string;
  reason?: string;
  evidence?: string;
};
