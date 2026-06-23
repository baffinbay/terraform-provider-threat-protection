package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &CaCertificateResource{}
var _ resource.ResourceWithImportState = &CaCertificateResource{}

func NewCaCertificateResource() resource.Resource {
	return &CaCertificateResource{}
}

type CaCertificateResource struct {
	client *client.Client
}

type CaCertificateResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Certificate types.String `tfsdk:"certificate"`
}

func (r *CaCertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ca_certificate"
}

func (r *CaCertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Baffin Bay CA Certificate.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the CA certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The API-derived name of the CA certificate.",
			},
			"certificate": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "The PEM encoded CA certificate(s).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *CaCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *CaCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CaCertificateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	caCertificate, err := r.client.CreateCaCertificate(ctx, "", data.Certificate.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create CA certificate, got error: %s", err))
		return
	}

	data.ID = types.StringValue(caCertificate.Data.ID)

	data, err = r.readCaCertificate(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read CA certificate after create, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CaCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CaCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.readCaCertificate(ctx, data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read CA certificate, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CaCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unsupported CA Certificate Update",
		"The Baffin Bay CA certificate API does not support in-place CA certificate updates. Configurable CA certificate fields are marked RequiresReplace; reaching Update indicates an unexpected planning path.",
	)
}

func (r *CaCertificateResource) readCaCertificate(ctx context.Context, prior CaCertificateResourceModel) (CaCertificateResourceModel, error) {
	caCertificate, err := r.client.GetCaCertificate(ctx, prior.ID.ValueString())
	if err != nil {
		return prior, err
	}

	return mapCaCertificateDataToModel(caCertificate.Data, prior), nil
}

func mapCaCertificateDataToModel(caCertificate client.CaCertificateData, prior CaCertificateResourceModel) CaCertificateResourceModel {
	data := prior
	data.ID = types.StringValue(caCertificate.ID)

	if caCertificate.Attributes.Name != "" {
		data.Name = types.StringValue(caCertificate.Attributes.Name)
	}

	if data.Certificate.IsNull() || data.Certificate.IsUnknown() {
		if certificate := caCertificatePEM(caCertificate.Attributes.Certificates); certificate != "" {
			data.Certificate = types.StringValue(certificate)
		}
	}

	return data
}

func caCertificatePEM(certificates []client.CaCertificateDataCertificate) string {
	parts := make([]string, 0, len(certificates))
	for _, certificate := range certificates {
		if certificate.Certificate != "" {
			parts = append(parts, certificate.Certificate)
		}
	}

	return strings.Join(parts, "\n")
}

func (r *CaCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CaCertificateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCaCertificate(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete CA certificate, got error: %s", err))
		return
	}
}

func (r *CaCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
