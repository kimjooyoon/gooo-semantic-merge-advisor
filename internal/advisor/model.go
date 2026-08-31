package advisor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	ManifestSchema       = "gooo/semantic-merge-advisor/manifest/v1"
	AuthoritySchema      = "gooo/semantic-merge-advisor/authority/v1"
	DenominatorSchema    = "gooo/semantic-merge-advisor/denominator/v1"
	ProposalSchema       = "gooo/semantic-merge-advisor/merge-proposal/v1"
	ReceiptSchema        = "gooo/semantic-merge-advisor/authority-receipt/v1"
	CounterexampleSchema = "gooo/semantic-merge-advisor/counterexample-report/v1"

	StateClosed  = "CLOSED"
	StateUnknown = "UNKNOWN"
	StateRefuted = "REFUTED"

	DecisionUnion    = "UNION"
	DecisionUnknown  = "PRESERVE_UNKNOWN"
	DecisionRefuted  = "REJECT_REFUTED"
	FixedDenominator = 9
)

var Precedence = []string{StateRefuted, StateUnknown, StateClosed}

// Manifest is a semantic, immutable view of a Git tree. It intentionally has
// no source-text field: the advisor reasons over declared observations only.
type Manifest struct {
	Schema           string          `json:"schema"`
	TreeID           string          `json:"tree_id"`
	TreeOID          string          `json:"tree_oid"`
	Immutable        bool            `json:"immutable"`
	RepositoryWrites int             `json:"repository_writes"`
	Entries          []ManifestEntry `json:"entries"`
}

type ManifestEntry struct {
	Path         string        `json:"path"`
	Blob         string        `json:"blob"`
	Status       string        `json:"status"`
	Ownership    string        `json:"ownership"`
	Generation   int           `json:"generation"`
	NewFile      bool          `json:"new_file"`
	PreviousFile bool          `json:"previous_file"`
	Symbols      []Observation `json:"symbols"`
	Claims       []Observation `json:"claims"`
	Producers    []Observation `json:"producers"`
	Evaluators   []Observation `json:"evaluators"`
}

type Observation struct {
	ID         string `json:"id"`
	Authority  string `json:"authority"`
	Version    string `json:"version"`
	Status     string `json:"status"`
	Observable bool   `json:"observable"`
	StaleAPI   bool   `json:"stale_api"`
}

type AuthorityDeclaration struct {
	Schema            string            `json:"schema"`
	Phase             string            `json:"phase"`
	OntologyReference string            `json:"ontology_reference"`
	OntologyReadOnly  bool              `json:"ontology_read_only"`
	Activities        []ActivityBinding `json:"activities"`
}

type ActivityBinding struct {
	Name              string `json:"name"`
	SourcePath        string `json:"source_path"`
	SourceLine        int    `json:"source_line"`
	IRNode            string `json:"ir_node"`
	GeneratedArtifact string `json:"generated_artifact"`
	Evaluator         string `json:"evaluator"`
	ProofChoice       string `json:"proof_choice"`
	IndicatorClass    string `json:"indicator_class"`
}

type Denominator struct {
	Schema        string            `json:"schema"`
	DenominatorID string            `json:"denominator_id"`
	Total         int               `json:"total"`
	Cells         []DenominatorCell `json:"cells"`
}

type DenominatorCell struct {
	Ordinal        int    `json:"ordinal"`
	Activity       string `json:"activity"`
	MetricID       string `json:"metric_id"`
	ProofChoice    string `json:"proof_choice"`
	IndicatorClass string `json:"indicator_class"`
}

type Candidate struct {
	Kind         string `json:"kind"`
	Identity     string `json:"identity"`
	TreeID       string `json:"tree_id"`
	Path         string `json:"path"`
	Blob         string `json:"blob"`
	Authority    string `json:"authority"`
	Version      string `json:"version"`
	Status       string `json:"status"`
	Ownership    string `json:"ownership"`
	Generation   int    `json:"generation"`
	NewFile      bool   `json:"new_file"`
	PreviousFile bool   `json:"previous_file"`
	Observable   bool   `json:"observable"`
	StaleAPI     bool   `json:"stale_api"`
}

type PlanItem struct {
	Kind           string      `json:"kind"`
	Identity       string      `json:"identity"`
	State          string      `json:"state"`
	Action         string      `json:"action"`
	CandidateCount int         `json:"candidate_count"`
	AuthorityCount int         `json:"authority_count"`
	ExactlyOne     bool        `json:"exactly_one"`
	Candidates     []Candidate `json:"candidates"`
	Reasons        []string    `json:"reasons,omitempty"`
	Unknown        *Unknown    `json:"unknown,omitempty"`
}

type BindingCheck struct {
	Activity          string `json:"activity"`
	MetricID          string `json:"metric_id"`
	SourcePath        string `json:"source_path"`
	SourceLine        int    `json:"source_line"`
	IRNode            string `json:"ir_node"`
	GeneratedArtifact string `json:"generated_artifact"`
	Evaluator         string `json:"evaluator"`
	ExactlyOne        bool   `json:"exactly_one"`
}

type CardinalityCheck struct {
	Kind           string `json:"kind"`
	Identity       string `json:"identity"`
	CandidateCount int    `json:"candidate_count"`
	AuthorityCount int    `json:"authority_count"`
	ExactlyOne     bool   `json:"exactly_one"`
}

// Unknown has exactly six semantic coordinates. Keep this type closed so an
// unobservable binding cannot be silently collapsed into a generic error.
type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Metrics struct {
	PhaseWallMS           map[string]int64 `json:"phase_wall_ms"`
	PeakRSSBytes          int64            `json:"peak_rss_bytes"`
	ArtifactFiles         int              `json:"artifact_files"`
	ArtifactBytes         int64            `json:"artifact_bytes"`
	RepositoryWrites      int              `json:"repository_writes"`
	InputRepositoryWrites int              `json:"input_repository_writes"`
	LocalTestsRun         int              `json:"local_tests_run"`
}

type Plan struct {
	Schema                string             `json:"schema"`
	Decision              string             `json:"decision"`
	State                 string             `json:"state"`
	Improvement           string             `json:"improvement"`
	Precedence            []string           `json:"precedence"`
	LeftTreeID            string             `json:"left_tree_id"`
	RightTreeID           string             `json:"right_tree_id"`
	LeftTreeDigest        string             `json:"left_tree_digest"`
	RightTreeDigest       string             `json:"right_tree_digest"`
	AuthorityDigest       string             `json:"authority_digest"`
	DenominatorDigest     string             `json:"denominator_digest"`
	SourceDigest          string             `json:"source_digest"`
	SourceTextMerged      bool               `json:"source_text_merged"`
	AuthorityBindings     []BindingCheck     `json:"authority_bindings"`
	Cardinality           []CardinalityCheck `json:"cardinality"`
	Items                 []PlanItem         `json:"items"`
	InputRepositoryWrites int                `json:"input_repository_writes"`
	Metrics               Metrics            `json:"metrics"`
}

type Counterexample struct {
	Kind       string      `json:"kind"`
	Identity   string      `json:"identity"`
	Reasons    []string    `json:"reasons"`
	Candidates []Candidate `json:"candidates"`
}

type UnknownWitness struct {
	Kind     string  `json:"kind"`
	Identity string  `json:"identity"`
	Unknown  Unknown `json:"unknown"`
}

type CounterexampleReport struct {
	Schema                string           `json:"schema"`
	State                 string           `json:"state"`
	Precedence            []string         `json:"precedence"`
	Refutations           []Counterexample `json:"refutations"`
	Unknowns              []UnknownWitness `json:"unknowns"`
	InputRepositoryWrites int              `json:"input_repository_writes"`
	Metrics               Metrics          `json:"metrics"`
}

type AuthorityReceipt struct {
	Schema                string            `json:"schema"`
	State                 string            `json:"state"`
	Decision              string            `json:"decision"`
	Improvement           string            `json:"improvement"`
	LeftTreeDigest        string            `json:"left_tree_digest"`
	RightTreeDigest       string            `json:"right_tree_digest"`
	AuthorityDigest       string            `json:"authority_digest"`
	DenominatorDigest     string            `json:"denominator_digest"`
	SourceDigest          string            `json:"source_digest"`
	OntologyReference     string            `json:"ontology_reference"`
	OntologyReadOnly      bool              `json:"ontology_read_only"`
	SourceTextMerged      bool              `json:"source_text_merged"`
	ArtifactDigests       map[string]string `json:"artifact_digests"`
	InputRepositoryWrites int               `json:"input_repository_writes"`
	RepositoryWrites      int               `json:"repository_writes"`
	Metrics               Metrics           `json:"metrics"`
}

type Generated struct {
	Proposal       Plan
	Receipt        AuthorityReceipt
	Counterexample CounterexampleReport
}

func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func CanonicalDigest(data []byte) string {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return Digest(data)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return Digest(data)
	}
	return Digest(canonical)
}
