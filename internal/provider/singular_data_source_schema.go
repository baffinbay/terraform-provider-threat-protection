package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var singularDataSourceUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func singularDataSourceIDAttribute(description string) datasourceschema.StringAttribute {
	return datasourceschema.StringAttribute{
		Required:            true,
		MarkdownDescription: description,
		Validators: []validator.String{
			stringvalidator.RegexMatches(singularDataSourceUUIDPattern, "must be a valid UUID"),
		},
	}
}

func setComputedDataSourceSchemaFromResource(ctx context.Context, resp *datasource.SchemaResponse, source resource.Resource, description string) {
	resourceResp := &resource.SchemaResponse{}
	source.Schema(ctx, resource.SchemaRequest{}, resourceResp)
	resp.Diagnostics.Append(resourceResp.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	attributes, err := computedDataSourceAttributes(resourceResp.Schema.Attributes)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Build Data Source Schema", err.Error())
		return
	}
	attributes["id"] = singularDataSourceIDAttribute("The UUID of the remote object to retrieve.")
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: description,
		Attributes:          attributes,
	}
}

func computedDataSourceAttributes(attributes map[string]resourceschema.Attribute) (map[string]datasourceschema.Attribute, error) {
	result := make(map[string]datasourceschema.Attribute, len(attributes))
	for name, attribute := range attributes {
		converted, err := computedDataSourceAttribute(attribute)
		if err != nil {
			return nil, fmt.Errorf("convert attribute %q: %w", name, err)
		}
		result[name] = converted
	}
	return result, nil
}

func computedDataSourceAttribute(attribute resourceschema.Attribute) (datasourceschema.Attribute, error) {
	switch value := attribute.(type) {
	case resourceschema.StringAttribute:
		return datasourceschema.StringAttribute{Computed: true, Sensitive: value.Sensitive, Description: value.Description, MarkdownDescription: value.MarkdownDescription, DeprecationMessage: value.DeprecationMessage}, nil
	case resourceschema.BoolAttribute:
		return datasourceschema.BoolAttribute{Computed: true, Sensitive: value.Sensitive, Description: value.Description, MarkdownDescription: value.MarkdownDescription, DeprecationMessage: value.DeprecationMessage}, nil
	case resourceschema.Int64Attribute:
		return datasourceschema.Int64Attribute{Computed: true, Sensitive: value.Sensitive, Description: value.Description, MarkdownDescription: value.MarkdownDescription, DeprecationMessage: value.DeprecationMessage}, nil
	case resourceschema.ListAttribute:
		return datasourceschema.ListAttribute{Computed: true, Sensitive: value.Sensitive, ElementType: value.ElementType, Description: value.Description, MarkdownDescription: value.MarkdownDescription, DeprecationMessage: value.DeprecationMessage}, nil
	case resourceschema.ListNestedAttribute:
		attributes, err := computedDataSourceAttributes(value.NestedObject.Attributes)
		if err != nil {
			return nil, err
		}
		return datasourceschema.ListNestedAttribute{Computed: true, Sensitive: value.Sensitive, NestedObject: datasourceschema.NestedAttributeObject{Attributes: attributes}, Description: value.Description, MarkdownDescription: value.MarkdownDescription, DeprecationMessage: value.DeprecationMessage}, nil
	case resourceschema.SingleNestedAttribute:
		attributes, err := computedDataSourceAttributes(value.Attributes)
		if err != nil {
			return nil, err
		}
		return datasourceschema.SingleNestedAttribute{Computed: true, Sensitive: value.Sensitive, Attributes: attributes, Description: value.Description, MarkdownDescription: value.MarkdownDescription, DeprecationMessage: value.DeprecationMessage}, nil
	default:
		return nil, fmt.Errorf("unsupported resource attribute type %T", attribute)
	}
}

func addSingularDataSourceReadError(diags interface{ AddError(summary, detail string) }, objectName, id string, err error) {
	if client.IsNotFound(err) {
		diags.AddError(objectName+" Not Found", fmt.Sprintf("Unable to find %s with ID %q.", objectName, id))
		return
	}
	diags.AddError("Client Error", fmt.Sprintf("Unable to read %s %q, got error: %s", objectName, id, err))
}

func addTypedTrafficDataSourceReadError(diags interface{ AddError(summary, detail string) }, objectName, id, expectedType string, err error) {
	if mismatch, ok := err.(trafficConfigTypeMismatchError); ok {
		diags.AddError("Traffic Config Type Mismatch", fmt.Sprintf("Unable to read %s %q: %s. Reference a traffic config with API type %q.", objectName, id, mismatch, expectedType))
		return
	}
	addSingularDataSourceReadError(diags, objectName, id, err)
}
