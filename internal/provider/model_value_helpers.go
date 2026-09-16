package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

func optionalStringPtrValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

func optionalInt64PtrValue(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

func stringSliceToTypeValues(values []string) []types.String {
	result := make([]types.String, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}

	return result
}
