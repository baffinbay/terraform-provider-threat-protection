package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CertificatesDataSource{}

func NewCertificatesDataSource() datasource.DataSource {
	return &CertificatesDataSource{}
}

type CertificatesDataSource struct {
	configuredDataSource
}

type CertificatesDataSourceModel struct {
	Certificates []CertificateModel `tfsdk:"certificates"`
}

type CertificateModel struct {
	ID         types.String `tfsdk:"id"`
	Type       types.String `tfsdk:"type"`
	CommonName types.String `tfsdk:"common_name"`
}

func (d *CertificatesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificates"
}

func (d *CertificatesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = collectionDataSourceSchema("Retrieves all Baffin Bay Certificates.", "certificates", "common_name")
}

func (d *CertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state CertificatesDataSourceModel

	respCerts, err := d.client.GetCertificates(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read certificates, got error: %s", err))
		return
	}

	for _, cert := range respCerts.Data {
		state.Certificates = append(state.Certificates, CertificateModel{
			ID:         types.StringValue(cert.ID),
			Type:       types.StringValue(cert.Type),
			CommonName: types.StringValue(cert.Attributes.CommonName),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
