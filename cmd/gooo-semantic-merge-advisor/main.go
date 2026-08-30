package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kimjooyoon/gooo-semantic-merge-advisor/internal/advisor"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "plan" {
		fmt.Fprintln(os.Stderr, "usage: gooo-semantic-merge-advisor plan --left FILE --right FILE --authority FILE --source FILE --denominator FILE --output DIR")
		os.Exit(2)
	}
	if err := runPlan(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "semantic merge advisor:", err)
		os.Exit(1)
	}
}

func runPlan(args []string) error {
	flags := flag.NewFlagSet("plan", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	leftPath := flags.String("left", "", "immutable left Git tree manifest")
	rightPath := flags.String("right", "", "immutable right Git tree manifest")
	authorityPath := flags.String("authority", "", "Gooo authority declaration")
	sourcePath := flags.String("source", "", "Gooo source declaration")
	denominatorPath := flags.String("denominator", "", "fixed denominator")
	outputDir := flags.String("output", "", "artifact output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"left": *leftPath, "right": *rightPath, "authority": *authorityPath,
		"source": *sourcePath, "denominator": *denominatorPath, "output": *outputDir,
	} {
		if value == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}

	loadStarted := time.Now()
	left, leftBytes, err := advisor.LoadJSON[advisor.Manifest](*leftPath)
	if err != nil {
		return err
	}
	right, rightBytes, err := advisor.LoadJSON[advisor.Manifest](*rightPath)
	if err != nil {
		return err
	}
	authority, authorityBytes, err := advisor.LoadJSON[advisor.AuthorityDeclaration](*authorityPath)
	if err != nil {
		return err
	}
	denominator, denominatorBytes, err := advisor.LoadJSON[advisor.Denominator](*denominatorPath)
	if err != nil {
		return err
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		return err
	}
	phaseWallMS := map[string]int64{"load_inputs": time.Since(loadStarted).Milliseconds()}

	input := advisor.BuildInput{
		Left:              left,
		Right:             right,
		Authority:         authority,
		Denominator:       denominator,
		SourcePath:        *sourcePath,
		Source:            source,
		LeftDigest:        advisor.CanonicalDigest(leftBytes),
		RightDigest:       advisor.CanonicalDigest(rightBytes),
		AuthorityDigest:   advisor.CanonicalDigest(authorityBytes),
		DenominatorDigest: advisor.CanonicalDigest(denominatorBytes),
	}

	serializeStarted := time.Now()
	generated, files, err := advisor.GenerateFiles(input, phaseWallMS, advisor.PeakRSSBytes())
	if err != nil {
		return err
	}
	phaseWallMS["serialize_artifacts"] = time.Since(serializeStarted).Milliseconds()
	writeMS, err := advisor.WriteFiles(*outputDir, files)
	if err != nil {
		return err
	}
	phaseWallMS["write_artifacts"] = writeMS
	generated, files, err = advisor.GenerateFiles(input, phaseWallMS, advisor.PeakRSSBytes())
	if err != nil {
		return err
	}
	if _, err := advisor.WriteFiles(*outputDir, files); err != nil {
		return err
	}

	summary, err := json.Marshal(map[string]any{
		"decision": generated.Proposal.Decision,
		"state":    generated.Proposal.State,
		"output":   *outputDir,
	})
	if err != nil {
		return err
	}
	fmt.Println(string(summary))
	return nil
}
