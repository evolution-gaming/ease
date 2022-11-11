// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Encoding plan configuration related abstractions.
package encoding

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PlanConfigError error type defines PlanConfig validation failures.
type PlanConfigError struct {
	msg     string
	reasons []string
}

func (e *PlanConfigError) Error() string {
	if len(e.reasons) > 0 {
		return fmt.Sprintf("%s with reasons:\n%s", e.msg, strings.Join(e.reasons, "\n"))
	}
	return e.msg
}

func (e *PlanConfigError) Reasons() []string {
	return e.reasons
}

func (e *PlanConfigError) addReason(reason string) {
	e.reasons = append(e.reasons, reason)
}

// PlanConfig holds configuration for new Plan creation.
type PlanConfig struct {
	// List of source (mezzanine) video files.
	Inputs  []string
	Schemes []Scheme
}

// NewPlanConfigFromJSON will unmarshal JSON into PlanConfig instance.
func NewPlanConfigFromJSON(jdoc []byte) (PlanConfig, error) {
	var pc PlanConfig
	err := json.Unmarshal(jdoc, &pc)
	if err != nil {
		return pc, err
	}
	return pc, nil
}

func (p *PlanConfig) IsValid() (bool, error) {
	errPlanConfig := &PlanConfigError{msg: "validation error"}

	if len(p.Inputs) == 0 {
		errPlanConfig.addReason("Inputs missing")
	}
	if hasDuplicates(p.Inputs) {
		errPlanConfig.addReason("Duplicate inputs detected")
	}
	if len(p.Schemes) == 0 {
		errPlanConfig.addReason("Schemes missing")
	}

	for _, i := range p.Inputs {
		if _, err := os.Stat(i); err != nil {
			errPlanConfig.addReason(err.Error())
		}
	}

	// Check if there were any validation errors?
	if len(errPlanConfig.reasons) != 0 {
		return false, errPlanConfig
	}
	return true, nil
}

// hasDuplicates checks if slice has duplicate elements.
func hasDuplicates(items []string) bool {
	// Create a poor man's seen
	seen := make(map[string]struct{}, len(items))
	for _, v := range items {
		if _, ok := seen[v]; ok {
			return true
		} else {
			seen[v] = struct{}{}
		}
	}
	return false
}
