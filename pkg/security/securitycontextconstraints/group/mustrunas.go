package group

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	api "k8s.io/kubernetes/pkg/apis/core"

	securityapi "github.com/openshift/origin/pkg/security/apis/security"
)

// mustRunAs implements the GroupSecurityContextConstraintsStrategy interface
type mustRunAs struct {
	ranges []securityapi.IDRange
	field  string
}

var _ GroupSecurityContextConstraintsStrategy = &mustRunAs{}

// NewMustRunAs provides a new MustRunAs strategy based on ranges.
func NewMustRunAs(ranges []securityapi.IDRange, field string) (GroupSecurityContextConstraintsStrategy, error) {
	if len(ranges) == 0 {
		return nil, fmt.Errorf("ranges must be supplied for MustRunAs")
	}
	return &mustRunAs{
		ranges: ranges,
		field:  field,
	}, nil
}

// Generate creates the group based on policy rules.  By default this returns the first group of the
// first range (min val).
func (s *mustRunAs) Generate(pod *api.Pod) ([]int64, error) {
	return []int64{s.ranges[0].Min}, nil
}

// Generate a single value to be applied.  This is used for FSGroup.  This strategy will return
// the first group of the first range (min val).
func (s *mustRunAs) GenerateSingle(pod *api.Pod) (*int64, error) {
	single := new(int64)
	*single = s.ranges[0].Min
	return single, nil
}

// Validate ensures that the specified values fall within the range of the strategy.
// Groups are passed in here to allow this strategy to support multiple group fields (fsgroup and
// supplemental groups).
func (s *mustRunAs) Validate(pod *api.Pod, groups []int64) field.ErrorList {
	allErrs := field.ErrorList{}

	if pod.Spec.SecurityContext == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("securityContext"), pod.Spec.SecurityContext, "unable to validate nil security context"))
		return allErrs
	}

	if len(groups) == 0 && len(s.ranges) > 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath(s.field), groups, "unable to validate empty groups against required ranges"))
	}

	for _, group := range groups {
		if !s.isGroupValid(group) {
			detail := fmt.Sprintf("%d is not an allowed group", group)
			allErrs = append(allErrs, field.Invalid(field.NewPath(s.field), groups, detail))
		}
	}

	return allErrs
}

func (s *mustRunAs) isGroupValid(group int64) bool {
	for _, rng := range s.ranges {
		if fallsInRange(group, rng) {
			return true
		}
	}
	return false
}

func fallsInRange(group int64, rng securityapi.IDRange) bool {
	return group >= rng.Min && group <= rng.Max
}
