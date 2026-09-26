package atlantis

import "github.com/lanej/statecraft/internal/domain"

// Project mirrors only the stable repo-level Atlantis project fields that
// Statecraft needs for root identity. It is intentionally not the Atlantis
// server's internal Go type.
type Project struct {
	Name      string
	Directory string
	Workspace string
}

func RootFromProject(project Project) domain.RootSelector {
	workspace := project.Workspace
	if workspace == "" {
		workspace = "default"
	}
	return domain.RootSelector{
		ID:          stableProjectID(project.Name, project.Directory, workspace),
		PlannerRef: project.Name,
		Directory:   project.Directory,
		Workspace:   workspace,
	}
}

func stableProjectID(name, directory, workspace string) string {
	if name != "" {
		return name
	}
	return directory + "::" + workspace
}
