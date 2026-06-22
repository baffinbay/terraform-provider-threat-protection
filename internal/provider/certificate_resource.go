package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &CertificateResource{}
var _ resource.ResourceWithImportState = &CertificateResource{}

func NewCertificateResource() resource.Resource {
	return &CertificateResource{}
}

type CertificateResource struct {
	client *client.Client
}

type CertificateResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Type         types.String `tfsdk:"type"`
	Certificate  types.String `tfsdk:"certificate"`
	Intermediate types.String `tfsdk:"intermediate"`
	Key          types.String `tfsdk:"key"`
	FQDN         types.String `tfsdk:"fqdn"`
}

func (r *CertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate"
}

func (r *CertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Baffin Bay Certificate.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of certificate (pem, lets_encrypt).",
				Validators: []validator.String{
					stringvalidator.OneOf("pem", "lets_encrypt"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The PEM encoded certificate. Required for type 'pem'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"intermediate": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The PEM encoded intermediate certificate. Optional for type 'pem'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The PEM encoded private key. Required for type 'pem'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fqdn": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The FQDN for the certificate. Required for type 'lets_encrypt'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *CertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CertificateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cert, err := r.client.CreateCertificate(
		ctx,
		data.Type.ValueString(),
		data.Certificate.ValueString(),
		data.Intermediate.ValueString(),
		data.Key.ValueString(),
		data.FQDN.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create certificate, got error: %s", err))
		return
	}

	data.ID = types.StringValue(cert.Data.ID)

	data, err = r.readCertificate(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read certificate after create, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.readCertificate(ctx, data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read certificate, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unsupported Certificate Update",
		"The Baffin Bay certificate API does not support in-place certificate updates. Configurable certificate fields are marked RequiresReplace; reaching Update indicates an unexpected planning path.",
	)
}

func (r *CertificateResource) readCertificate(ctx context.Context, prior CertificateResourceModel) (CertificateResourceModel, error) {
	cert, err := r.client.FindCertificateByID(ctx, prior.ID.ValueString())
	if err != nil {
		return prior, err
	}

	return mapCertificateDataToModel(cert, prior), nil
}

func mapCertificateDataToModel(cert *client.CertificateData, prior CertificateResourceModel) CertificateResourceModel {
	data := prior
	data.ID = types.StringValue(cert.ID)
	data.Type = terraformCertificateType(cert.Type, prior.Type)

	if data.Type.ValueString() == "lets_encrypt" {
		if cert.Attributes.FQDN != "" {
			data.FQDN = types.StringValue(cert.Attributes.FQDN)
		} else if cert.Attributes.CommonName != "" && data.FQDN.IsNull() {
			data.FQDN = types.StringValue(cert.Attributes.CommonName)
		}
	}

	return data
}

func terraformCertificateType(apiType string, prior types.String) types.String {
	switch apiType {
	case "importedCertificate", "importPemCertificate", "pem":
		return types.StringValue("pem")
	case "letsEncrypt", "lets_encrypt":
		return types.StringValue("lets_encrypt")
	case "":
		return prior
	default:
		if !prior.IsNull() {
			return prior
		}
		return types.StringValue(apiType)
	}
}

func (r *CertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CertificateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCertificate(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete certificate, got error: %s", err))
		return
	}
}

func (r *CertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
