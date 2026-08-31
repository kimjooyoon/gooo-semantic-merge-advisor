package advisor

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type BuildInput struct {
	Left              Manifest
	Right             Manifest
	Authority         AuthorityDeclaration
	Denominator       Denominator
	SourcePath        string
	Source            []byte
	LeftDigest        string
	RightDigest       string
	AuthorityDigest   string
	DenominatorDigest string
}

func ValidateManifest(manifest Manifest) error {
	if manifest.Schema != ManifestSchema {
		return fmt.Errorf("unsupported manifest schema %q", manifest.Schema)
	}
	if strings.TrimSpace(manifest.TreeID) == "" || strings.TrimSpace(manifest.TreeOID) == "" {
		return errors.New("manifest tree_id and tree_oid are required")
	}
	if !manifest.Immutable {
		return fmt.Errorf("manifest %q is not immutable", manifest.TreeID)
	}
	if manifest.RepositoryWrites != 0 {
		return fmt.Errorf("manifest %q requests %d repository writes", manifest.TreeID, manifest.RepositoryWrites)
	}
	seenPaths := make(map[string]bool, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if strings.TrimSpace(entry.Path) == "" || strings.TrimSpace(entry.Blob) == "" {
			return fmt.Errorf("manifest %q contains an entry without path or blob", manifest.TreeID)
		}
		if seenPaths[entry.Path] {
			return fmt.Errorf("manifest %q duplicates path %q", manifest.TreeID, entry.Path)
		}
		seenPaths[entry.Path] = true
		if entry.Generation < 1 {
			return fmt.Errorf("manifest %q has invalid generation for %q", manifest.TreeID, entry.Path)
		}
		observations := []struct {
			kind  string
			items []Observation
		}{
			{kind: "symbol", items: entry.Symbols},
			{kind: "claim", items: entry.Claims},
			{kind: "producer", items: entry.Producers},
			{kind: "evaluator", items: entry.Evaluators},
		}
		for _, group := range observations {
			for _, observation := range group.items {
				if strings.TrimSpace(observation.ID) == "" || strings.TrimSpace(observation.Authority) == "" {
					return fmt.Errorf("manifest %q has incomplete %s observation in %q", manifest.TreeID, group.kind, entry.Path)
				}
			}
		}
	}
	return nil
}

func ValidateDenominator(denominator Denominator) error {
	if denominator.Schema != DenominatorSchema {
		return fmt.Errorf("unsupported denominator schema %q", denominator.Schema)
	}
	if denominator.Total != FixedDenominator || len(denominator.Cells) != FixedDenominator {
		return fmt.Errorf("denominator must contain exactly %d cells", FixedDenominator)
	}
	if strings.TrimSpace(denominator.DenominatorID) == "" {
		return errors.New("denominator_id is required")
	}
	proofCounts := make(map[string]int)
	indicatorCounts := make(map[string]int)
	pairs := make(map[string]int)
	activities := make(map[string]bool)
	metrics := make(map[string]bool)
	for _, cell := range denominator.Cells {
		if strings.TrimSpace(cell.Activity) == "" || strings.TrimSpace(cell.MetricID) == "" {
			return errors.New("denominator cells require activity and metric_id")
		}
		if activities[cell.Activity] || metrics[cell.MetricID] {
			return fmt.Errorf("denominator duplicates activity or metric %q", cell.Activity)
		}
		activities[cell.Activity] = true
		metrics[cell.MetricID] = true
		proofCounts[cell.ProofChoice]++
		indicatorCounts[cell.IndicatorClass]++
		pairs[cell.ProofChoice+"\x00"+cell.IndicatorClass]++
	}
	for _, choice := range []string{"FOUNDATION", "COHERENCE", "REGRESSION"} {
		if proofCounts[choice] != 3 {
			return fmt.Errorf("proof choice %q is not balanced at 3", choice)
		}
	}
	for _, class := range []string{"DRIVER", "OUTCOME", "GUARDRAIL"} {
		if indicatorCounts[class] != 3 {
			return fmt.Errorf("indicator class %q is not balanced at 3", class)
		}
	}
	for _, choice := range []string{"FOUNDATION", "COHERENCE", "REGRESSION"} {
		for _, class := range []string{"DRIVER", "OUTCOME", "GUARDRAIL"} {
			if pairs[choice+"\x00"+class] != 1 {
				return fmt.Errorf("denominator pair %s/%s is not exactly one", choice, class)
			}
		}
	}
	return nil
}

func ValidateAuthority(authority AuthorityDeclaration, denominator Denominator, sourcePath string, source []byte) error {
	if authority.Schema != AuthoritySchema {
		return fmt.Errorf("unsupported authority schema %q", authority.Schema)
	}
	if authority.Phase == "FIXED_POINT" {
		return errors.New("authority declaration is at FIXED_POINT")
	}
	if authority.Phase != "OPEN" {
		return fmt.Errorf("authority phase must be OPEN, got %q", authority.Phase)
	}
	if !authority.OntologyReadOnly || !strings.Contains(authority.OntologyReference, "meta-ontology-go") {
		return errors.New("meta-ontology-go must be an external read-only ontology reference")
	}
	if len(authority.Activities) != denominator.Total {
		return fmt.Errorf("authority has %d activities, denominator requires %d", len(authority.Activities), denominator.Total)
	}
	lineByName := sourceActivityLines(source)
	byActivity := make(map[string]DenominatorCell, len(denominator.Cells))
	for _, cell := range denominator.Cells {
		byActivity[cell.Activity] = cell
	}
	seenActivities := make(map[string]bool, len(authority.Activities))
	seenLines := make(map[int]bool, len(authority.Activities))
	seenIR := make(map[string]bool, len(authority.Activities))
	seenArtifacts := make(map[string]bool, len(authority.Activities))
	seenEvaluators := make(map[string]bool, len(authority.Activities))
	for _, activity := range authority.Activities {
		if seenActivities[activity.Name] || seenLines[activity.SourceLine] || seenIR[activity.IRNode] || seenArtifacts[activity.GeneratedArtifact] || seenEvaluators[activity.Evaluator] {
			return fmt.Errorf("authority binding is not one-to-one for activity %q", activity.Name)
		}
		cell, ok := byActivity[activity.Name]
		if !ok {
			return fmt.Errorf("authority activity %q is absent from denominator", activity.Name)
		}
		if activity.SourcePath != sourcePath && !strings.HasSuffix(filepath.ToSlash(sourcePath), "/"+filepath.ToSlash(activity.SourcePath)) {
			return fmt.Errorf("activity %q source path does not bind to %q", activity.Name, sourcePath)
		}
		if activity.SourceLine < 1 || lineByName[activity.Name] != activity.SourceLine {
			return fmt.Errorf("activity %q source line does not bind to Gooo source", activity.Name)
		}
		if activity.ProofChoice != cell.ProofChoice || activity.IndicatorClass != cell.IndicatorClass {
			return fmt.Errorf("activity %q is not bound to its fixed denominator cell", activity.Name)
		}
		for _, value := range []string{activity.IRNode, activity.GeneratedArtifact, activity.Evaluator} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("activity %q has an empty generated binding", activity.Name)
			}
		}
		seenActivities[activity.Name] = true
		seenLines[activity.SourceLine] = true
		seenIR[activity.IRNode] = true
		seenArtifacts[activity.GeneratedArtifact] = true
		seenEvaluators[activity.Evaluator] = true
	}
	for activity := range byActivity {
		if !seenActivities[activity] {
			return fmt.Errorf("denominator activity %q is absent from authority", activity)
		}
	}
	if len(authority.MetaActivities) != 1 {
		return errors.New("authority requires exactly one preferredDurationUnit meta activity")
	}
	meta := authority.MetaActivities[0]
	if meta.Name != "preferredDurationUnit" {
		return fmt.Errorf("unexpected duration meta activity %q", meta.Name)
	}
	if meta.SourcePath != sourcePath && !strings.HasSuffix(filepath.ToSlash(sourcePath), "/"+filepath.ToSlash(meta.SourcePath)) {
		return errors.New("preferredDurationUnit source path does not bind to Gooo source")
	}
	if meta.SourceLine < 1 || lineByName[meta.Name] != meta.SourceLine {
		return errors.New("preferredDurationUnit source line does not bind to Gooo source")
	}
	for _, value := range []string{meta.IRNode, meta.GeneratedArtifact, meta.Evaluator} {
		if strings.TrimSpace(value) == "" {
			return errors.New("preferredDurationUnit has an empty generated binding")
		}
	}
	if seenLines[meta.SourceLine] || seenIR[meta.IRNode] || seenArtifacts[meta.GeneratedArtifact] || seenEvaluators[meta.Evaluator] {
		return errors.New("preferredDurationUnit binding is not one-to-one")
	}
	for phase, duration := range authority.DurationNS {
		if duration < 0 {
			return fmt.Errorf("malformed duration for phase %q: %d ns", phase, duration)
		}
	}
	return nil
}

func Build(input BuildInput) (Plan, CounterexampleReport, error) {
	if err := ValidateManifest(input.Left); err != nil {
		return Plan{}, CounterexampleReport{}, err
	}
	if err := ValidateManifest(input.Right); err != nil {
		return Plan{}, CounterexampleReport{}, err
	}
	if err := ValidateDenominator(input.Denominator); err != nil {
		return Plan{}, CounterexampleReport{}, err
	}
	if err := ValidateAuthority(input.Authority, input.Denominator, input.SourcePath, input.Source); err != nil {
		return Plan{}, CounterexampleReport{}, err
	}

	candidates := append(collectCandidates(input.Left), collectCandidates(input.Right)...)
	groups := make(map[string][]Candidate)
	for _, candidate := range candidates {
		key := candidate.Kind + "\x00" + candidate.Identity
		groups[key] = append(groups[key], candidate)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]PlanItem, 0, len(keys))
	cardinality := make([]CardinalityCheck, 0, len(keys))
	refutations := make([]Counterexample, 0)
	unknowns := make([]UnknownWitness, 0)
	state := StateClosed
	for _, key := range keys {
		group := groups[key]
		sort.Slice(group, func(i, j int) bool {
			if group[i].Kind != group[j].Kind {
				return group[i].Kind < group[j].Kind
			}
			if group[i].Identity != group[j].Identity {
				return group[i].Identity < group[j].Identity
			}
			return group[i].Path < group[j].Path
		})
		reasons := refutationReasons(group)
		unknown := hasUnobservable(group)
		itemState := StateClosed
		action := "retain_single_authority"
		if len(reasons) > 0 {
			itemState = StateRefuted
			action = "reject_refuted_binding"
			if state != StateRefuted {
				state = StateRefuted
			}
		} else if unknown {
			itemState = StateUnknown
			action = "preserve_unknown_binding"
			if state == StateClosed {
				state = StateUnknown
			}
		}
		authorityCount := uniqueAuthorities(group)
		exactlyOne := len(group) == 1 && authorityCount == 1 && itemState == StateClosed
		item := PlanItem{
			Kind:           group[0].Kind,
			Identity:       group[0].Identity,
			State:          itemState,
			Action:         action,
			CandidateCount: len(group),
			AuthorityCount: authorityCount,
			ExactlyOne:     exactlyOne,
			Candidates:     group,
			Reasons:        reasons,
		}
		if unknown && len(reasons) == 0 {
			item.Unknown = unknownRecord(group[0])
			unknowns = append(unknowns, UnknownWitness{Kind: item.Kind, Identity: item.Identity, Unknown: *item.Unknown})
		}
		if len(reasons) > 0 {
			refutations = append(refutations, Counterexample{Kind: item.Kind, Identity: item.Identity, Reasons: reasons, Candidates: group})
		}
		items = append(items, item)
		cardinality = append(cardinality, CardinalityCheck{Kind: item.Kind, Identity: item.Identity, CandidateCount: len(group), AuthorityCount: authorityCount, ExactlyOne: exactlyOne})
	}

	decision := DecisionUnion
	if state == StateUnknown {
		decision = DecisionUnknown
	}
	if state == StateRefuted {
		decision = DecisionRefuted
	}
	bindingChecks := make([]BindingCheck, 0, len(input.Authority.Activities)+len(input.Authority.MetaActivities))
	cellByActivity := make(map[string]DenominatorCell, len(input.Denominator.Cells))
	for _, cell := range input.Denominator.Cells {
		cellByActivity[cell.Activity] = cell
	}
	activities := append([]ActivityBinding(nil), input.Authority.Activities...)
	sort.Slice(activities, func(i, j int) bool { return activities[i].Name < activities[j].Name })
	for _, activity := range activities {
		bindingChecks = append(bindingChecks, BindingCheck{
			Activity:          activity.Name,
			MetricID:          cellByActivity[activity.Name].MetricID,
			SourcePath:        activity.SourcePath,
			SourceLine:        activity.SourceLine,
			IRNode:            activity.IRNode,
			GeneratedArtifact: activity.GeneratedArtifact,
			Evaluator:         activity.Evaluator,
			ExactlyOne:        true,
		})
	}
	meta := input.Authority.MetaActivities[0]
	bindingChecks = append(bindingChecks, BindingCheck{
		Activity:          meta.Name,
		MetricID:          "meta.preferred_duration_unit",
		SourcePath:        meta.SourcePath,
		SourceLine:        meta.SourceLine,
		IRNode:            meta.IRNode,
		GeneratedArtifact: meta.GeneratedArtifact,
		Evaluator:         meta.Evaluator,
		ExactlyOne:        true,
	})
	plan := Plan{
		Schema:                ProposalSchema,
		Decision:              decision,
		State:                 state,
		Improvement:           StateUnknown,
		Precedence:            append([]string(nil), Precedence...),
		LeftTreeID:            input.Left.TreeID,
		RightTreeID:           input.Right.TreeID,
		LeftTreeDigest:        input.LeftDigest,
		RightTreeDigest:       input.RightDigest,
		AuthorityDigest:       input.AuthorityDigest,
		DenominatorDigest:     input.DenominatorDigest,
		SourceDigest:          Digest(input.Source),
		SourceTextMerged:      false,
		AuthorityBindings:     bindingChecks,
		Cardinality:           cardinality,
		Items:                 items,
		InputRepositoryWrites: 0,
	}
	report := CounterexampleReport{
		Schema:                CounterexampleSchema,
		State:                 state,
		Precedence:            append([]string(nil), Precedence...),
		Refutations:           refutations,
		Unknowns:              unknowns,
		InputRepositoryWrites: 0,
	}
	return plan, report, nil
}

func collectCandidates(manifest Manifest) []Candidate {
	var result []Candidate
	groups := []struct {
		kind  string
		items []Observation
	}{
		{kind: "symbol", items: nil},
		{kind: "claim", items: nil},
		{kind: "producer", items: nil},
		{kind: "evaluator", items: nil},
	}
	for _, entry := range manifest.Entries {
		groups[0].items = entry.Symbols
		groups[1].items = entry.Claims
		groups[2].items = entry.Producers
		groups[3].items = entry.Evaluators
		for _, group := range groups {
			for _, observation := range group.items {
				status := observation.Status
				if status == "" {
					status = entry.Status
				}
				stale := observation.StaleAPI || strings.EqualFold(status, "stale") || strings.EqualFold(status, "stale_api") || strings.EqualFold(entry.Status, "stale") || strings.EqualFold(entry.Status, "stale_api")
				result = append(result, Candidate{
					Kind:         group.kind,
					Identity:     observation.ID,
					TreeID:       manifest.TreeID,
					Path:         entry.Path,
					Blob:         entry.Blob,
					Authority:    observation.Authority,
					Version:      observation.Version,
					Status:       status,
					Ownership:    entry.Ownership,
					Generation:   entry.Generation,
					NewFile:      entry.NewFile,
					PreviousFile: entry.PreviousFile,
					Observable:   observation.Observable,
					StaleAPI:     stale,
				})
			}
		}
	}
	return result
}

func uniqueAuthorities(group []Candidate) int {
	seen := make(map[string]bool, len(group))
	for _, candidate := range group {
		seen[candidate.Authority] = true
	}
	return len(seen)
}

func hasUnobservable(group []Candidate) bool {
	for _, candidate := range group {
		if !candidate.Observable {
			return true
		}
	}
	return false
}

func refutationReasons(group []Candidate) []string {
	reasons := make([]string, 0, 4)
	authorityCounts := make(map[string]int, len(group))
	hasNew := false
	hasPrevious := false
	for _, candidate := range group {
		authorityCounts[candidate.Authority]++
		hasNew = hasNew || candidate.NewFile || candidate.Ownership == "new"
		hasPrevious = hasPrevious || candidate.PreviousFile || candidate.Ownership == "previous"
		if candidate.StaleAPI {
			reasons = appendUnique(reasons, "stale_api")
		}
	}
	if len(group) > 1 {
		for _, count := range authorityCounts {
			if count > 1 {
				reasons = appendUnique(reasons, "same_authority_twice")
			}
		}
		if hasNew && hasPrevious {
			reasons = appendUnique(reasons, "new_and_previous_file_double_ownership")
		}
		if uniqueAuthorities(group) > 1 {
			reasons = appendUnique(reasons, "multiple_authorities")
		}
		if len(reasons) == 0 {
			reasons = append(reasons, "duplicate_binding")
		}
	}
	return reasons
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func unknownRecord(candidate Candidate) *Unknown {
	return &Unknown{
		Stage:         "authority-union",
		Step:          "observe-" + candidate.Kind,
		Reason:        "binding is not observable in the immutable manifest",
		UnknownClass:  "UNOBSERVABLE_BINDING",
		NextOperation: "provide an observable source-backed binding",
		BlockedBy:     []string{candidate.Kind + ":" + candidate.Identity},
	}
}

func sourceActivityLines(source []byte) map[string]int {
	result := make(map[string]int)
	for lineNumber, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "activity ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(trimmed, "activity "))
		if open := strings.IndexByte(name, '('); open > 0 {
			result[strings.TrimSpace(name[:open])] = lineNumber + 1
		}
	}
	return result
}
