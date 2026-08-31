package advisor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"time"
)

func LoadJSON[T any](path string) (T, []byte, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, nil, err
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, data, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, data, nil
}

func MarshalArtifact(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// GenerateFiles builds all three artifacts without writing to any input path.
// artifact_bytes is solved as a small fixed point because it is itself part of
// each artifact's JSON metrics object.
func GenerateFiles(input BuildInput, phaseWallMS map[string]int64, peakRSSBytes int64) (Generated, map[string][]byte, error) {
	planStarted := time.Now()
	plan, report, err := Build(input)
	if err != nil {
		return Generated{}, nil, err
	}
	phaseWallMS = copyPhaseWallMS(phaseWallMS)
	phaseWallMS["validate_and_plan"] = time.Since(planStarted).Milliseconds()
	metrics := Metrics{
		PhaseWallMS:           phaseWallMS,
		PeakRSSBytes:          peakRSSBytes,
		ArtifactFiles:         3,
		ArtifactBytes:         0,
		RepositoryWrites:      0,
		InputRepositoryWrites: 0,
		LocalTestsRun:         0,
	}
	var final Generated
	var files map[string][]byte
	for attempt := 0; attempt < 32; attempt++ {
		plan.Metrics = metrics
		report.Metrics = metrics
		proposalBytes, err := MarshalArtifact(plan)
		if err != nil {
			return Generated{}, nil, err
		}
		counterexampleBytes, err := MarshalArtifact(report)
		if err != nil {
			return Generated{}, nil, err
		}
		receipt := AuthorityReceipt{
			Schema:            ReceiptSchema,
			State:             plan.State,
			Decision:          plan.Decision,
			Improvement:       StateUnknown,
			LeftTreeDigest:    input.LeftDigest,
			RightTreeDigest:   input.RightDigest,
			AuthorityDigest:   input.AuthorityDigest,
			DenominatorDigest: input.DenominatorDigest,
			SourceDigest:      Digest(input.Source),
			OntologyReference: input.Authority.OntologyReference,
			OntologyReadOnly:  input.Authority.OntologyReadOnly,
			SourceTextMerged:  false,
			ArtifactDigests: map[string]string{
				"merge-proposal.json":        Digest(proposalBytes),
				"counterexample-report.json": Digest(counterexampleBytes),
			},
			InputRepositoryWrites: 0,
			RepositoryWrites:      0,
			Metrics:               metrics,
		}
		receiptBytes, err := MarshalArtifact(receipt)
		if err != nil {
			return Generated{}, nil, err
		}
		totalBytes := int64(len(proposalBytes) + len(receiptBytes) + len(counterexampleBytes))
		final = Generated{Proposal: plan, Receipt: receipt, Counterexample: report}
		files = map[string][]byte{
			"merge-proposal.json":        proposalBytes,
			"authority-receipt.json":     receiptBytes,
			"counterexample-report.json": counterexampleBytes,
		}
		if totalBytes == metrics.ArtifactBytes {
			return final, files, nil
		}
		metrics.ArtifactBytes = totalBytes
	}
	return Generated{}, nil, errors.New("artifact byte metric did not reach a fixed point")
}

func WriteFiles(outputDir string, files map[string][]byte) (int64, error) {
	if outputDir == "" {
		return 0, errors.New("output directory is required")
	}
	started := time.Now()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return 0, err
	}
	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		if filepath.Base(name) != name {
			return 0, fmt.Errorf("artifact name %q escapes output directory", name)
		}
		if err := os.WriteFile(filepath.Join(outputDir, name), files[name], 0o644); err != nil {
			return 0, err
		}
	}
	return time.Since(started).Milliseconds(), nil
}

func PeakRSSBytes() int64 {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0
	}
	rss := int64(usage.Maxrss)
	if runtime.GOOS == "linux" {
		rss *= 1024
	}
	return rss
}

func copyPhaseWallMS(values map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(values)+1)
	for key, value := range values {
		result[key] = value
	}
	return result
}
