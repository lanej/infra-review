package mock

import (
	"context"
	"errors"

	"github.com/lanej/statecraft/internal/domain"
)

type ReviewStore struct{}

func NewReviewStore() *ReviewStore { return &ReviewStore{} }

func (s *ReviewStore) GetReview(_ context.Context, id string) (domain.Review, error) {
	if id != "pr-1842" {
		return domain.Review{}, errors.New("review not found")
	}
	return domain.Review{
		ID: "pr-1842", Repository: "example/infrastructure", PullRequest: 1842,
		Title: "Increase API capacity and tighten DB access", HeadSHA: "abc1234", State: "ready_for_review",
		Roots: []domain.Root{
			{ID: "api", Name: "azure/prod/api", Status: "planned"},
			{ID: "network", Name: "azure/prod/network", Status: "planned"},
			{ID: "observability", Name: "azure/prod/observability", Status: "planned"},
			{ID: "identity", Name: "azure/prod/identity", Status: "planned"},
		},
		Changes: []domain.Change{
			{ID: "c1", RootID: "api", Address: "azurerm_kubernetes_cluster.api", ResourceType: "azurerm_kubernetes_cluster", Action: "modify", Risk: "medium", Summary: "Node count 3 → 5"},
			{ID: "c2", RootID: "network", Address: "azurerm_network_security_rule.api", ResourceType: "azurerm_network_security_rule", Action: "modify", Risk: "high", Summary: "Network access rules changed"},
			{ID: "c3", RootID: "api", Address: "azurerm_postgresql_flexible_server.main", ResourceType: "azurerm_postgresql_flexible_server", Action: "replace", Risk: "critical", Summary: "Replacement required by configuration changes"},
			{ID: "c4", RootID: "observability", Address: "azurerm_monitor_diagnostic_setting.api", ResourceType: "azurerm_monitor_diagnostic_setting", Action: "create", Risk: "low", Summary: "New diagnostic setting"},
		},
		Findings: []domain.Finding{
			{ID: "f1", Severity: "critical", Category: "destructive", Title: "Production database will be replaced", ResourceAddress: "azurerm_postgresql_flexible_server.main", Blocking: true},
			{ID: "f2", Severity: "high", Category: "network", Title: "API network policy changes", ResourceAddress: "azurerm_network_security_rule.api", Blocking: false},
		},
		Decisions: []domain.ReviewDecision{
			{Actor: "matt", Decision: "approved", PlanSetID: "planset-7", CreatedAt: "2026-09-25T20:10:00Z"},
		},
	}, nil
}
