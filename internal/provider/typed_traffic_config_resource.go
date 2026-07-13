package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type typedTrafficConfigResourceModel interface {
	trafficConfigID() string
	trafficConfigRequest() client.TrafficConfigRequest
}

func updateTypedTrafficConfigResource[Model typedTrafficConfigResourceModel](
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
	apiClient *client.Client,
	objectName string,
	read func(context.Context, Model) (Model, error),
	addReadDiagnostic func(errorDiagnostics, string, error, string),
) {
	var data Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.trafficConfigID()
	trafficConfig, err := apiClient.UpdateTrafficConfig(ctx, id, data.trafficConfigRequest())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update %s traffic config, got error: %s", objectName, err))
		return
	}
	if err := apiClient.WaitForTrafficConfigChange(ctx, id, trafficConfig.Data.ActiveChangeID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for %s traffic config update, got error: %s", objectName, err))
		return
	}

	data, err = read(ctx, data)
	if err != nil {
		addReadDiagnostic(&resp.Diagnostics, id, err, "after update")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
