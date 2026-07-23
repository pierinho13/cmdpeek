package configsource

import "testing"

func TestParseGitHubSource(t *testing.T) {
	t.Parallel()

	source, err := parseGitHubSource(
		"company/platform-config:cmdpeek/commands.yaml@main",
	)
	if err != nil {
		t.Fatalf("parseGitHubSource() error = %v", err)
	}

	if source.Owner != "company" ||
		source.Repo != "platform-config" ||
		source.Path != "cmdpeek/commands.yaml" ||
		source.Ref != "main" {
		t.Fatalf("unexpected source: %#v", source)
	}
}

func TestParseGitHubSourceWithoutRef(t *testing.T) {
	t.Parallel()

	source, err := parseGitHubSource(
		"company/platform-config:cmdpeek/commands.yaml",
	)
	if err != nil {
		t.Fatalf("parseGitHubSource() error = %v", err)
	}

	if source.Ref != "" {
		t.Fatalf("expected empty ref, got %q", source.Ref)
	}
}

func TestParseGitHubSourceRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if _, err := parseGitHubSource("platform-config"); err == nil {
		t.Fatal("expected invalid source error")
	}
}
