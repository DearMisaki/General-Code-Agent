package memory

import (
	"math"
	"strings"
)

type PolicyOptions struct {
	MinConfidence float64
}

type FilterReason string

const (
	FilterReasonAccepted      FilterReason = "accepted"
	FilterReasonInvalid       FilterReason = "invalid"
	FilterReasonLowConfidence FilterReason = "low_confidence"
	FilterReasonSensitive     FilterReason = "sensitive"
)

type FilterDecision struct {
	Accepted  bool
	Reason    FilterReason
	Candidate MemoryCandidate
}

type MemoryPolicyFilter struct {
	opts PolicyOptions
}

func NewMemoryPolicyFilter(opts PolicyOptions) *MemoryPolicyFilter {
	if opts.MinConfidence == 0 {
		opts.MinConfidence = 0.75
	}
	return &MemoryPolicyFilter{opts: opts}
}

func (f *MemoryPolicyFilter) Filter(c MemoryCandidate) FilterDecision {
	if err := c.Validate(); err != nil {
		return FilterDecision{Reason: FilterReasonInvalid, Candidate: c}
	}
	if containsCredential(c.Memory) || c.Sensitivity == SensitivityCredential {
		return FilterDecision{Reason: FilterReasonSensitive, Candidate: c}
	}
	min := f.opts.MinConfidence
	if c.Kind == KindConstraint || c.Kind == KindArchitectureDecision {
		min = math.Max(min, 0.85)
	}
	if c.Confidence < min {
		return FilterDecision{Reason: FilterReasonLowConfidence, Candidate: c}
	}
	if c.Sensitivity == "" {
		c.Sensitivity = SensitivityInternal
	}
	return FilterDecision{Accepted: true, Reason: FilterReasonAccepted, Candidate: c}
}

func containsCredential(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{"api key", "token", "password", "secret", "cookie", "sk-"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
