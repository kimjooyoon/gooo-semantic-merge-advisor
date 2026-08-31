package advisor

import (
	"errors"
	"fmt"
	"sort"
)

func phaseMetrics(rawDurations map[string]int64, authority AuthorityDeclaration) (Metrics, []UnknownWitness, error) {
	raw := copyDurations(rawDurations)
	if authority.DurationObservable != nil && !*authority.DurationObservable {
		for phase := range raw {
			raw[phase] = 0
		}
	}
	for phase, duration := range authority.DurationNS {
		if duration < 0 {
			return Metrics{}, nil, fmt.Errorf("malformed duration for phase %q: %d ns", phase, duration)
		}
		if authority.DurationObservable == nil || *authority.DurationObservable {
			raw[phase] = duration
		}
	}
	telemetry := make(map[string]PhaseTelemetry, len(raw))
	wallMS := make(map[string]int64, len(raw))
	unknowns := make([]UnknownWitness, 0)
	phases := make([]string, 0, len(raw))
	for phase := range raw {
		phases = append(phases, phase)
	}
	sort.Strings(phases)
	for _, phase := range phases {
		observation, err := choosePreferredDuration(phase, raw[phase])
		if err != nil {
			return Metrics{}, nil, err
		}
		telemetry[phase] = observation
		wallMS[phase] = observation.DurationMS
		if observation.Unknown != nil {
			unknowns = append(unknowns, UnknownWitness{Kind: "duration", Identity: phase, Unknown: *observation.Unknown})
		}
	}
	return Metrics{
		PhaseWallMS:              wallMS,
		PhaseTelemetry:           telemetry,
		PeakRSSBytes:             0,
		ArtifactFiles:            3,
		ArtifactBytes:            0,
		RepositoryWrites:         0,
		InputRepositoryWrites:    0,
		LocalTestsRun:            0,
	}, unknowns, nil
}

func choosePreferredDuration(phase string, durationNS int64) (PhaseTelemetry, error) {
	if durationNS < 0 {
		return PhaseTelemetry{}, errors.New("duration cannot be negative")
	}
	durationUS := durationNS / 1_000
	durationMS := durationNS / 1_000_000
	telemetry := PhaseTelemetry{
		DurationNS: durationNS,
		DurationUS: durationUS,
		DurationMS: durationMS,
	}
	switch {
	case durationMS >= 1:
		telemetry.PreferredDurationUnit = "ms"
		telemetry.PreferredDurationValue = durationMS
	case durationUS >= 1:
		telemetry.PreferredDurationUnit = "us"
		telemetry.PreferredDurationValue = durationUS
	case durationNS > 0:
		telemetry.PreferredDurationUnit = "ns"
		telemetry.PreferredDurationValue = durationNS
	default:
		telemetry.PreferredDurationUnit = StateUnknown
		telemetry.PreferredDurationValue = 0
		telemetry.Unknown = &Unknown{
			Stage:         "preferred-duration-unit",
			Step:          "select-" + phase,
			Reason:        "duration is zero or unobservable",
			UnknownClass:  "UNOBSERVABLE_DURATION",
			NextOperation: "provide an observable monotonic duration",
			BlockedBy:     []string{"duration:" + phase},
		}
	}
	return telemetry, nil
}

func copyDurations(values map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(values)+2)
	for key, value := range values {
		result[key] = value
	}
	return result
}
