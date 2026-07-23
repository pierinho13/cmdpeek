package tui

import (
	"testing"

	"github.com/pierinho13/cmdpeek/internal/catalog"
)

func TestNewAppliesInitialQuery(t *testing.T) {
	t.Parallel()

	commands := []catalog.Command{
		{
			Name:        "switch-kubernetes-context",
			Title:       "Switch Kubernetes context",
			Description: "Select and use a kubeconfig context",
			Labels:      []string{"kubernetes", "context"},
		},
		{
			Name:        "show-kubernetes-context",
			Title:       "Show Kubernetes context",
			Description: "Print the active kubeconfig context",
			Labels:      []string{"kubernetes", "context"},
		},
		{
			Name:   "aws-login",
			Title:  "Log in to AWS",
			Labels: []string{"aws", "sso"},
		},
	}

	model := New(commands, "kubernetes context")

	if len(model.filtered) != 2 {
		t.Fatalf("expected 2 filtered commands, got %d", len(model.filtered))
	}

	if model.filterInput.Value() != "kubernetes context" {
		t.Fatalf(
			"expected initial query to be preserved, got %q",
			model.filterInput.Value(),
		)
	}
}

func TestInitialSelectionUsesExactCommandName(t *testing.T) {
	t.Parallel()

	commands := []catalog.Command{
		{
			Name:   "switch-kubernetes-context",
			Title:  "Switch Kubernetes context",
			Labels: []string{"kubernetes", "context"},
		},
		{
			Name:   "show-kubernetes-context",
			Title:  "Show Kubernetes context",
			Labels: []string{"switch-kubernetes-context"},
		},
	}

	model := New(commands, "switch-kubernetes-context")

	selected, ok := model.initialSelection("switch-kubernetes-context")
	if !ok {
		t.Fatal("expected an initial selection")
	}

	if selected.Name != "switch-kubernetes-context" {
		t.Fatalf("expected exact name match, got %q", selected.Name)
	}
}

func TestInitialSelectionUsesSingleSearchResult(t *testing.T) {
	t.Parallel()

	commands := []catalog.Command{
		{
			Name:        "switch-kubernetes-context",
			Title:       "Switch Kubernetes context",
			Description: "Select and use a kubeconfig context",
			Labels:      []string{"kubernetes", "context"},
		},
		{
			Name:   "aws-login",
			Title:  "Log in to AWS",
			Labels: []string{"aws"},
		},
	}

	model := New(commands, "kubeconfig")

	selected, ok := model.initialSelection("kubeconfig")
	if !ok {
		t.Fatal("expected the single filtered command to be selected")
	}

	if selected.Name != "switch-kubernetes-context" {
		t.Fatalf(
			"expected switch-kubernetes-context, got %q",
			selected.Name,
		)
	}
}

func TestInitialSelectionDoesNotSelectMultipleResults(t *testing.T) {
	t.Parallel()

	commands := []catalog.Command{
		{
			Name:   "switch-kubernetes-context",
			Title:  "Switch Kubernetes context",
			Labels: []string{"kubernetes"},
		},
		{
			Name:   "show-kubernetes-context",
			Title:  "Show Kubernetes context",
			Labels: []string{"kubernetes"},
		},
	}

	model := New(commands, "kubernetes")

	if _, ok := model.initialSelection("kubernetes"); ok {
		t.Fatal("did not expect automatic selection for multiple results")
	}
}
