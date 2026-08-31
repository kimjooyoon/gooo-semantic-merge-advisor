package advisor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadCase(t *testing.T, name string, authorityName string) BuildInput {
	t.Helper()
	root := filepath.Join("..", "..")
	left, leftBytes, err := LoadJSON[Manifest](filepath.Join(root, "fixtures", "cases", name, "left.json"))
	if err != nil {
		t.Fatal(err)
	}
	right, rightBytes, err := LoadJSON[Manifest](filepath.Join(root, "fixtures", "cases", name, "right.json"))
	if err != nil {
		t.Fatal(err)
	}
	authority, authorityBytes, err := LoadJSON[AuthorityDeclaration](filepath.Join(root, "fixtures", "cases", authorityName))
	if err != nil {
		t.Fatal(err)
	}
	denominator, denominatorBytes, err := LoadJSON[Denominator](filepath.Join(root, "contracts", "semantic-merge-advisor-denominator-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "examples", "semantic-merge-authority", "main.gooo")
	source, err := readTestFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	return BuildInput{
		Left:              left,
		Right:             right,
		Authority:         authority,
		Denominator:       denominator,
		SourcePath:        sourcePath,
		Source:            source,
		LeftDigest:        CanonicalDigest(leftBytes),
		RightDigest:       CanonicalDigest(rightBytes),
		AuthorityDigest:   CanonicalDigest(authorityBytes),
		DenominatorDigest: CanonicalDigest(denominatorBytes),
	}
}

func readTestFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func TestNormalUnionClosesOnlyExactlyOneBindings(t *testing.T) {
	plan, report, err := Build(loadCase(t, "normal", "authority.json"))
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != StateClosed || plan.Decision != DecisionUnion || plan.Improvement != StateUnknown || report.State != StateClosed {
		t.Fatalf("unexpected normal result: state=%s decision=%s report=%s", plan.State, plan.Decision, report.State)
	}
	if plan.SourceTextMerged || len(plan.Items) != 8 || len(plan.AuthorityBindings) != FixedDenominator {
		t.Fatalf("normal result lost non-text or one-to-one evidence: %+v", plan)
	}
	for _, item := range plan.Items {
		if !item.ExactlyOne {
			t.Fatalf("normal item is not exactly one: %+v", item)
		}
	}
}

func TestUnknownPreservesSixCoordinates(t *testing.T) {
	plan, report, err := Build(loadCase(t, "unknown", "authority.json"))
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != StateUnknown || plan.Decision != DecisionUnknown || len(report.Unknowns) != 4 {
		t.Fatalf("unexpected unknown result: state=%s decision=%s unknowns=%d", plan.State, plan.Decision, len(report.Unknowns))
	}
	unknownBytes, err := json.Marshal(report.Unknowns[0].Unknown)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(unknownBytes, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 6 {
		t.Fatalf("unknown record has %d fields, want six", len(fields))
	}
}

func TestRefutedFixtureNamesGeneralizedFailures(t *testing.T) {
	plan, report, err := Build(loadCase(t, "refuted", "authority.json"))
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != StateRefuted || plan.Decision != DecisionRefuted || len(report.Refutations) < 4 {
		t.Fatalf("unexpected refuted result: state=%s decision=%s refutations=%d", plan.State, plan.Decision, len(report.Refutations))
	}
	reasons := make(map[string]bool)
	for _, refutation := range report.Refutations {
		for _, reason := range refutation.Reasons {
			reasons[reason] = true
		}
	}
	for _, reason := range []string{"stale_api", "same_authority_twice", "new_and_previous_file_double_ownership", "multiple_authorities"} {
		if !reasons[reason] {
			t.Fatalf("missing refutation reason %q: %v", reason, reasons)
		}
	}
}

func TestMalformedAndFixedPointFailClosed(t *testing.T) {
	malformed := loadCase(t, "malformed", "authority.json")
	if _, _, err := Build(malformed); err == nil || !strings.Contains(err.Error(), "unsupported manifest schema") {
		t.Fatalf("malformed input did not fail closed: %v", err)
	}
	fixedPoint := loadCase(t, "fixed-point", "fixed-point/authority.json")
	if _, _, err := Build(fixedPoint); err == nil || !strings.Contains(err.Error(), "FIXED_POINT") {
		t.Fatalf("fixed point did not fail closed: %v", err)
	}
}
