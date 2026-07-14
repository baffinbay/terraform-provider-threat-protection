package provider

import (
	"context"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
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
	configuredDataSource
}

type HTTPProxyDataSource struct {
	configuredDataSource
}

type L4ProxyDataSource struct {
	configuredDataSource
}

type RoutedDsrDataSource struct {
	configuredDataSource
}

func (d *TrafficConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic_config"
}

func (d *TrafficConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	setComputedDataSourceSchemaFromResource(ctx, resp, NewTrafficConfigResource(), "Retrieves a Baffin Bay Traffic Configuration by UUID.")
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

func (d *HTTPProxyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	readTypedTrafficDataSource(
		ctx, req, resp, "HTTP Proxy", httpProxyTrafficConfigType,
		func(state HTTPProxyResourceModel) string { return state.ID.ValueString() },
		d.client.GetHTTPProxy,
		func(response *client.HTTPProxyResponse) string { return response.Data.Type },
		mapHTTPProxyResponseToModel,
	)
}

func (d *L4ProxyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_l4_proxy"
}

func (d *L4ProxyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	setComputedDataSourceSchemaFromResource(ctx, resp, NewL4ProxyResource(), "Retrieves a Baffin Bay L4 Proxy traffic configuration by UUID.")
}

func (d *L4ProxyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	readTypedTrafficDataSource(
		ctx, req, resp, "L4 Proxy", l4ProxyTrafficConfigType,
		func(state L4ProxyResourceModel) string { return state.ID.ValueString() },
		d.client.GetTrafficConfig,
		func(response *client.TrafficConfigResponse) string { return response.Data.Type },
		mapL4ProxyResponseToModel,
	)
}

func (d *RoutedDsrDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routed_dsr"
}

func (d *RoutedDsrDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	setComputedDataSourceSchemaFromResource(ctx, resp, NewRoutedDsrResource(), "Retrieves a Baffin Bay Routed DSR traffic configuration by UUID.")
}

func (d *RoutedDsrDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	readTypedTrafficDataSource(
		ctx, req, resp, "Routed DSR", routedDsrTrafficConfigType,
		func(state RoutedDsrResourceModel) string { return state.ID.ValueString() },
		d.client.GetTrafficConfig,
		func(response *client.TrafficConfigResponse) string { return response.Data.Type },
		mapRoutedDsrResponseToModel,
	)
}

func readTypedTrafficDataSource[State, Response any](
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
	objectName, expectedType string,
	stateID func(State) string,
	read func(context.Context, string) (Response, error),
	responseType func(Response) string,
	mapResponse func(Response, State) State,
) {
	var state State
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := stateID(state)
	response, err := read(ctx, id)
	if err != nil {
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, objectName, id, expectedType, err)
		return
	}
	if actualType := responseType(response); actualType != expectedType {
		err = trafficConfigTypeMismatchError{ID: id, GotType: actualType, WantType: expectedType}
		addTypedTrafficDataSourceReadError(&resp.Diagnostics, objectName, id, expectedType, err)
		return
	}

	state = mapResponse(response, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
