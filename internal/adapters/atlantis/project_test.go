package atlantis

import "testing"

func TestRootFromProject(t *testing.T) {
	root := RootFromProject(Project{Directory: "prod/api"})
	if root.ID != "prod/api::default" || root.Workspace != "default" {
		t.Fatalf("root = %#v", root)
	}

	named := RootFromProject(Project{Name: "api", Directory: "prod/api", Workspace: "production"})
	if named.ID != "api" || named.PlannerRef != "api" {
		t.Fatalf("named root = %#v", named)
	}
}
