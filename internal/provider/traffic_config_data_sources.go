package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &TrafficConfigDataSource{}
var _ datasource.DataSource = &HTTPProxyDataSource{}
var _ datasource.DataSource = &L4ProxyDataSource{}
var _ datasource.DataSource = &RoutedDsrDataSource{}

func NewTrafficConfigDataSource() datasource.DataSource {
	return &TrafficConfigDataSource{}
}

func NewHTTPProxyDataSource() datasource.DataSource {
	return &HTTPProxyDataSource{}
}

func NewL4ProxyDataSource() datasource.DataSource {
	return &L4ProxyDataSource{}
}

func NewRoutedDsrDataSource() datasource.DataSource {
	return &RoutedDsrDataSource{}
}

type TrafficConfigDataSource struct {
	client *client.Client
}

type HTTPProxyDataSource struct {
	client *client.Client
}

type L4ProxyDataSource struct {
	client *client.Client
}

type RoutedDsrDataSource struct {
	client *client.Client
}

func (d *TrafficConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic_config"
}

func (d *TrafficConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	setComputedDataSourceSchemaFromResource(ctx, resp, NewTrafficConfigResource(), "Retrieves a Baffin Bay Traffic Configuration by UUID.")
}

func (d *TrafficConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureSingularDataSourceClient(req, resp)
}

func (d *TrafficConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TrafficConfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetTrafficConfig(ctx, state.ID.ValueString())
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "Traffic Configuration", state.ID.ValueString(), err)
		return
	}
	state = mapTrafficConfigResponseToModel(response, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *HTTPProxyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_proxy"
}

func (d *HTTPProxyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	setComputedDataSourceSchemaFromResource(ctx, resp, NewHTTPProxyResource(), "Retrieves a Baffin Bay HTTP Proxy traffic configuration by UUID.")
}

func (d *HTTPProxyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureSingularDataSourceClient(req, resp)
}

func (d *HTTPProxyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state HTTPProxyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetHTTPProxy(ctx, state.ID.ValueString())
	if err != nil {
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, "HTTP Proxy", state.ID.ValueString(), httpProxyTrafficConfigType, err)
		return
	}
	if response.Data.Type != httpProxyTrafficConfigType {
		err = trafficConfigTypeMismatchError{ID: state.ID.ValueString(), GotType: response.Data.Type, WantType: httpProxyTrafficConfigType}
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, "HTTP Proxy", state.ID.ValueString(), httpProxyTrafficConfigType, err)
		return
	}
	state = mapHTTPProxyResponseToModel(response, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *L4ProxyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_l4_proxy"
}

func (d *L4ProxyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	setComputedDataSourceSchemaFromResource(ctx, resp, NewL4ProxyResource(), "Retrieves a Baffin Bay L4 Proxy traffic configuration by UUID.")
}

func (d *L4ProxyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureSingularDataSourceClient(req, resp)
}

func (d *L4ProxyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state L4ProxyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetTrafficConfig(ctx, state.ID.ValueString())
	if err != nil {
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, "L4 Proxy", state.ID.ValueString(), l4ProxyTrafficConfigType, err)
		return
	}
	if response.Data.Type != l4ProxyTrafficConfigType {
		err = trafficConfigTypeMismatchError{ID: state.ID.ValueString(), GotType: response.Data.Type, WantType: l4ProxyTrafficConfigType}
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, "L4 Proxy", state.ID.ValueString(), l4ProxyTrafficConfigType, err)
		return
	}
	state = mapL4ProxyResponseToModel(response, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *RoutedDsrDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routed_dsr"
}

func (d *RoutedDsrDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	setComputedDataSourceSchemaFromResource(ctx, resp, NewRoutedDsrResource(), "Retrieves a Baffin Bay Routed DSR traffic configuration by UUID.")
}

func (d *RoutedDsrDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureSingularDataSourceClient(req, resp)
}

func (d *RoutedDsrDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state RoutedDsrResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetTrafficConfig(ctx, state.ID.ValueString())
	if err != nil {
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, "Routed DSR", state.ID.ValueString(), routedDsrTrafficConfigType, err)
		return
	}
	if response.Data.Type != routedDsrTrafficConfigType {
		err = trafficConfigTypeMismatchError{ID: state.ID.ValueString(), GotType: response.Data.Type, WantType: routedDsrTrafficConfigType}
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, "Routed DSR", state.ID.ValueString(), routedDsrTrafficConfigType, err)
		return
	}
	state = mapRoutedDsrResponseToModel(response, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func configureSingularDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return nil
	}
	return configuredClient
}
