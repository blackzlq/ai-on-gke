package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

type ShouldReportViolationFunc func(v *v1.Violation) bool
type PurifyViolationFunc func(v *v1.Violation) *v1.Violation

type DefaultResultHandler struct {
	ShouldReportViolation ShouldReportViolationFunc
	PurifyViolation       PurifyViolationFunc
}

type ReportAndRecordRequest struct {
	ResourceName string
	Violations   []*v1.Violation
}

type ReportAndRecordResult struct {
	ResourceName     string
	RecordViolations []*v1.Violation
	ReportViolations []*v1.Violation
}

func (d *DefaultResultHandler) HandleResult(ctx context.Context, objects []ResultObject) error {
	var results []ReportAndRecordResult
	for _, object := range objects {
		// Step 1: Filter the violation (stub for demonstration)
		filteredViolations, err := d.filterViolations(d.deduplicate(object.Violations))
		if err != nil {
			return fmt.Errorf("failed to filter violations: %v", err)
		}

		// Step 2: Report and record (stub for demonstration)
		result, err := d.reportAndRecord(ReportAndRecordRequest{
			ResourceName: object.ResourceName,
			Violations:   filteredViolations,
		})
		if err != nil {
			return fmt.Errorf("failed to report and record: %v", err)
		}

		results = append(results, result)

	}

	return d.examResult(results)
}

func (d *DefaultResultHandler) deduplicate(violations []*v1.Violation) []*v1.Violation {
	violationMap := map[string]map[string]*v1.Violation{}
	var violationArray []*v1.Violation
	for _, violation := range violations {
		v := d.PurifyViolation(violation)
		resourceKey := FetchResourceKey(v)
		if len(resourceKey) == 0 {
			log.Printf("failed to marshal resource key, key: %v", v.ResourceKey)
			violationArray = append(violationArray, v)
		} else {
			if violationMap[resourceKey] == nil {
				violationMap[resourceKey] = map[string]*v1.Violation{}
			}
			violationMap[resourceKey][v.Message] = v
		}
	}
	for _, violationMessageMap := range violationMap {
		for _, violation := range violationMessageMap {
			violationArray = append(violationArray, violation)
		}
	}
	return violationArray
}

func (d *DefaultResultHandler) filterViolations(violations []*v1.Violation) ([]*v1.Violation, error) {
	// Placeholder for actual filtering logic
	log.Println("Filtering violations...")
	return violations, nil
}

func (d *DefaultResultHandler) reportAndRecord(request ReportAndRecordRequest) (ReportAndRecordResult, error) {
	var reportViolations []*v1.Violation
	// Placeholder for actual reporting and recording logic
	log.Println("Generating report and record violations...")
	for _, violation := range request.Violations {
		if d.ShouldReportViolation(violation) {
			reportViolations = append(reportViolations, violation)
		}
	}
	return ReportAndRecordResult{
		ResourceName:     request.ResourceName,
		RecordViolations: request.Violations,
		ReportViolations: reportViolations,
	}, nil
}

func (d *DefaultResultHandler) examResult(results []ReportAndRecordResult) error {
	reportViolations := []*v1.Violation{}
	for _, result := range results {
		violationsByPolicy := map[string][]*v1.Violation{}
		for _, violation := range result.ReportViolations {
			if violationsByPolicy[violation.PolicyName] == nil {
				violationsByPolicy[violation.PolicyName] = []*v1.Violation{}
			}
			violationsByPolicy[violation.PolicyName] = append(violationsByPolicy[violation.PolicyName], violation)
			reportViolations = append(reportViolations, violation)
		}

		for policy, violationList := range violationsByPolicy {
			fileName := fmt.Sprintf("%s-%s.json", result.ResourceName, policy)

			jsonViolations, err := json.MarshalIndent(violationList, "", strings.Repeat(" ", 4))

			if err != nil {
				log.Printf("Failed to create json\n")
			} else {
				log.Printf("%s violation: %s \n \n==================\n", fileName, jsonViolations)

			}
		}
	}

	if len(reportViolations) != 0 {
		return fmt.Errorf("please check the output log see what is new violations")
	}

	return nil
}
