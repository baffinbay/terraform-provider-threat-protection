package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CaCertificatesDataSource{}

func NewCaCertificatesDataSource() datasource.DataSource {
	return &CaCertificatesDataSource{}
}

type CaCertificatesDataSource struct {
	configuredDataSource
}

type CaCertificatesDataSourceModel struct {
	CaCertificates []CaCertificateModel `tfsdk:"ca_certificates"`
}

type CaCertificateModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

func (d *CaCertificatesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ca_certificates"
}

func (d *CaCertificatesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = collectionDataSourceSchema("Retrieves all Baffin Bay CA Certificates.", "ca_certificates", "name")
}

func (d *CaCertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state CaCertificatesDataSourceModel

	respCertificates, err := d.client.GetCaCertificates(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read CA certificates, got error: %s", err))
		return
	}

	for _, certificate := range respCertificates.Data {
		state.CaCertificates = append(state.CaCertificates, CaCertificateModel{
			ID:   types.StringValue(certificate.ID),
			Type: types.StringValue(certificate.Type),
			Name: types.StringValue(certificate.Attributes.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
