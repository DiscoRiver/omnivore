package filters

import "github.com/aws/aws-sdk-go-v2/service/ec2/types"

// GenerateFilterSlice returns a slice of types.Filter for use in ec2.DescribeInstancesInput.Filters
func GenerateFilterSlice(filters map[string][]string) (f []types.Filter) {
	for k, v := range filters {
		f = append(f, types.Filter{
			Name:   &k,
			Values: v,
		})
	}
	return
}
