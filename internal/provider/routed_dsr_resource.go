package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const routedDsrTrafficConfigType = "routedDsr"

var _ resource.Resource = &RoutedDsrResource{}
var _ resource.ResourceWithImportState = &RoutedDsrResource{}

func NewRoutedDsrResource() resource.Resource {
	return &RoutedDsrResource{}
}

type RoutedDsrResource struct {
	configuredResource
}

type RoutedDsrResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Prefix          types.String `tfsdk:"prefix"`
	Announced       types.Bool   `tfsdk:"announced"`
	DeploymentState types.String `tfsdk:"deployment_state"`
}

func (r *RoutedDsrResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routed_dsr"
}

func (r *RoutedDsrResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Baffin Bay Routed DSR traffic configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Routed DSR traffic configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Routed DSR traffic configuration.",
			},
			"prefix": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The CIDR prefix for the Routed DSR traffic configuration.",
			},
			"announced": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Controls if the Routed DSR traffic configuration is announced.",
				Default:             booldefault.StaticBool(false),
			},
			"deployment_state": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The deployment state (DEPLOYED, UNDEPLOYED).",
				Default:             stringdefault.StaticString("UNDEPLOYED"),
				Validators: []validator.String{
					stringvalidator.OneOf("DEPLOYED", "UNDEPLOYED"),
				},
			},
		},
	}
}

func (r *RoutedDsrResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoutedDsrResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tc, err := r.client.CreateRoutedDsr(ctx, mapRoutedDsrModelToRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Routed DSR traffic config, got error: %s", err))
		return
	}

	data.ID = types.StringValue(tc.Data.ID)

	if err := r.client.WaitForTrafficConfigRollout(ctx, tc.Data.ID, tc.Data.ActiveRolloutID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for Routed DSR traffic config create, got error: %s", err))
		return
	}

	data, err = r.readRoutedDsr(ctx, data)
	if err != nil {
		addRoutedDsrReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "after create")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoutedDsrResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoutedDsrResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.readRoutedDsr(ctx, data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		addRoutedDsrReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoutedDsrResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RoutedDsrResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tc, err := r.client.UpdateRoutedDsr(ctx, data.ID.ValueString(), mapRoutedDsrModelToRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Routed DSR traffic config, got error: %s", err))
		return
	}
	if err := r.client.WaitForTrafficConfigRollout(ctx, data.ID.ValueString(), tc.Data.ActiveRolloutID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for Routed DSR traffic config update, got error: %s", err))
		return
	}

	data, err = r.readRoutedDsr(ctx, data)
	if err != nil {
		addRoutedDsrReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "after update")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoutedDsrResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoutedDsrResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	undeploy := func(ctx context.Context) (string, error) {
		current, err := r.readRoutedDsr(ctx, data)
		if err != nil {
			return "", err
		}
		current.DeploymentState = types.StringValue("UNDEPLOYED")
		tc, err := r.client.UpdateRoutedDsr(ctx, current.ID.ValueString(), mapRoutedDsrModelToRequest(current))
		if err != nil {
			return "", err
		}
		return tc.Data.ActiveRolloutID(), nil
	}

	deleteTrafficConfigResource(
		ctx,
		resp,
		r.client,
		"Routed DSR",
		data.ID.ValueString(),
		func(tc *client.TrafficConfigResponse) bool { return tc.Data.Type == routedDsrTrafficConfigType },
		undeploy,
		addRoutedDsrReadDiagnostic,
	)
}

func (r *RoutedDsrResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapRoutedDsrModelToRequest(data RoutedDsrResourceModel) client.RoutedDsrRequest {
	attrs := client.RoutedDsrAttributes{
		Name:    data.Name.ValueString(),
		Version: "0.1.0",
		Prefix:  data.Prefix.ValueString(),
	}

	if !data.DeploymentState.IsNull() {
		attrs.Deployment.State = data.DeploymentState.ValueString()
	} else {
		attrs.Deployment.State = "UNDEPLOYED"
	}

	announced := false
	if !data.Announced.IsNull() {
		announced = data.Announced.ValueBool()
	}
	attrs.Announced = &announced

	return client.NewRoutedDsrRequest(attrs)
}

func (r *RoutedDsrResource) readRoutedDsr(ctx context.Context, prior RoutedDsrResourceModel) (RoutedDsrResourceModel, error) {
	tc, err := r.client.GetRoutedDsr(ctx, prior.ID.ValueString())
	if err != nil {
		return prior, err
	}

	if tc.Data.Type != routedDsrTrafficConfigType {
		return prior, trafficConfigTypeMismatchError{
			ID:       prior.ID.ValueString(),
			GotType:  tc.Data.Type,
			WantType: routedDsrTrafficConfigType,
		}
	}

	return mapRoutedDsrResponseToModel(tc, prior), nil
}

func mapRoutedDsrResponseToModel(tc *client.TrafficConfigResponse, prior RoutedDsrResourceModel) RoutedDsrResourceModel {
	data := prior

	if tc.Data.ID != "" {
		data.ID = types.StringValue(tc.Data.ID)
	}

	attrs := tc.Data.Attributes
	if attrs.Name != nil {
		data.Name = types.StringValue(*attrs.Name)
	}
	if attrs.Prefix != nil {
		data.Prefix = types.StringValue(*attrs.Prefix)
	}
	if attrs.Announced != nil {
		data.Announced = types.BoolValue(*attrs.Announced)
	} else if data.Announced.IsNull() || data.Announced.IsUnknown() {
		data.Announced = types.BoolValue(false)
	}
	if attrs.Deployment != nil && attrs.Deployment.State != nil {
		data.DeploymentState = types.StringValue(*attrs.Deployment.State)
	} else if data.DeploymentState.IsNull() || data.DeploymentState.IsUnknown() {
		data.DeploymentState = types.StringValue("UNDEPLOYED")
	}

	return data
}

type trafficConfigTypeMismatchError struct {
	ID       string
	GotType  string
	WantType string
}

func (e trafficConfigTypeMismatchError) Error() string {
	return fmt.Sprintf("traffic config %q has type %q, expected %q", e.ID, e.GotType, e.WantType)
}

func addRoutedDsrReadDiagnostic(diags errorDiagnostics, id string, err error, context string) {
	if _, ok := err.(trafficConfigTypeMismatchError); ok {
		diags.AddError(
			"Traffic Config Type Mismatch",
			fmt.Sprintf("Unable to read Routed DSR traffic config %q%s: %s. Import or reference a traffic config with API type %q.", id, readContextSuffix(context), err, routedDsrTrafficConfigType),
		)
		return
	}

	diags.AddError(
		"Client Error",
		fmt.Sprintf("Unable to read Routed DSR traffic config %q%s, got error: %s", id, readContextSuffix(context), err),
	)
}

func readContextSuffix(context string) string {
	if context == "" {
		return ""
	}

	return " " + context
}
