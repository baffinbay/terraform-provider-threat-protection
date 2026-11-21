package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.ActionWithConfigure = &HTTPProxyContentCachePurgeAction{}

// HTTPProxyContentCachePurgeAction purges the content cache of a deployed HTTP Proxy.
// It's an unlinked action: it changes nothing Terraform tracks, so it never causes drift.
type HTTPProxyContentCachePurgeAction struct {
	configuredAction
}

type HTTPProxyContentCachePurgeActionModel struct {
	TrafficConfigID types.String `tfsdk:"traffic_config_id"`
}

func NewHTTPProxyContentCachePurgeAction() action.Action {
	return &HTTPProxyContentCachePurgeAction{}
}

func (a *HTTPProxyContentCachePurgeAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_proxy_content_cache_purge"
}

func (a *HTTPProxyContentCachePurgeAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Purges the content cache of a deployed HTTP Proxy and waits for the purge rollout to complete. " +
			"Run it on demand with `terraform apply -invoke` (Terraform 1.14 or later), or from a resource's `lifecycle.action_trigger`. " +
			"When the proxy itself triggers the action, set `traffic_config_id = caller.id` (Terraform 1.16 or later): " +
			"referencing the triggering resource directly is an action reference cycle.",
		Attributes: map[string]schema.Attribute{
			"traffic_config_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the HTTP Proxy whose content cache is purged: `baffinbay_http_proxy.example.id` for on-demand runs, or `caller.id` when the proxy triggers the action. The proxy must be deployed.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(singularDataSourceUUIDPattern, "must be a valid UUID"),
				},
			},
		},
	}
}

func (a *HTTPProxyContentCachePurgeAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data HTTPProxyContentCachePurgeActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if a.client == nil {
		resp.Diagnostics.AddError("Unconfigured Provider", "The provider was not configured before the action was invoked. Please report this issue to the provider developers.")
		return
	}
	id := data.TrafficConfigID.ValueString()

	tc, err := a.client.PurgeHTTPProxyContentCache(ctx, id)
	if err != nil {
		addContentCachePurgeDiagnostic(&resp.Diagnostics, id, err)
		return
	}

	rolloutID := tc.Data.ActiveRolloutID()
	if rolloutID == "" {
		resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Content cache purge requested for HTTP Proxy %s.", id)})
		return
	}
	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Content cache purge requested for HTTP Proxy %s; waiting for rollout %s.", id, rolloutID)})
	if err := a.client.WaitForTrafficConfigRollout(ctx, id, rolloutID); err != nil {
		resp.Diagnostics.AddError("Content Cache Purge Failed", fmt.Sprintf("The content cache purge of HTTP Proxy %q did not complete: %s", id, err))
		return
	}
	resp.SendProgress(action.InvokeProgressEvent{Message: fmt.Sprintf("Content cache purge rollout %s completed.", rolloutID)})
}

func addContentCachePurgeDiagnostic(diags errorDiagnostics, id string, err error) {
	switch {
	case client.IsHTTPStatus(err, http.StatusBadRequest):
		diags.AddError("Content Cache Purge Rejected", fmt.Sprintf(
			"The content cache can only be purged for a deployed HTTP proxy that is ready for updates. Check that %q is an HTTP Proxy, is deployed, and has no rollout in progress.\n\nAPI response: %s", id, err))
	case client.IsNotFound(err):
		diags.AddError("HTTP Proxy Not Found", fmt.Sprintf("HTTP Proxy %q was not found.", id))
	default:
		diags.AddError("Client Error", fmt.Sprintf("Unable to purge the content cache of HTTP Proxy %q: %s", id, err))
	}
}
