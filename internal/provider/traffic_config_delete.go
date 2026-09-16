package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type trafficConfigUndeployFunc func(context.Context) (activeRolloutID string, err error)

func deleteTrafficConfigResource(
	ctx context.Context,
	resp *resource.DeleteResponse,
	apiClient *client.Client,
	objectName string,
	id string,
	canUndeploy func(*client.TrafficConfigResponse) bool,
	undeploy trafficConfigUndeployFunc,
	addReadDiagnostic func(errorDiagnostics, string, error, string),
) {
	stateWasReconciled, ok := ensureTrafficConfigReadyForDeleteFromRemote(ctx, resp, apiClient, objectName, id, canUndeploy, undeploy, addReadDiagnostic, nil)
	if !ok {
		return
	}

	err := apiClient.DeleteTrafficConfig(ctx, id)
	if err == nil {
		return
	}
	if !client.IsHTTPStatus(err, http.StatusConflict) || stateWasReconciled {
		addTrafficConfigDeleteDiagnostic(resp, objectName, err)
		return
	}

	if _, ok := ensureTrafficConfigReadyForDeleteFromRemote(ctx, resp, apiClient, objectName, id, canUndeploy, undeploy, addReadDiagnostic, err); !ok {
		return
	}
	if err := apiClient.DeleteTrafficConfig(ctx, id); err != nil {
		addTrafficConfigDeleteDiagnostic(resp, objectName, err)
	}
}

func ensureTrafficConfigReadyForDeleteFromRemote(
	ctx context.Context,
	resp *resource.DeleteResponse,
	apiClient *client.Client,
	objectName string,
	id string,
	canUndeploy func(*client.TrafficConfigResponse) bool,
	undeploy trafficConfigUndeployFunc,
	addReadDiagnostic func(errorDiagnostics, string, error, string),
	originalErr error,
) (stateWasReconciled bool, ok bool) {
	currentResponse, readErr := apiClient.GetTrafficConfig(ctx, id)
	if readErr != nil {
		if client.IsHTTPStatus(readErr, http.StatusNotFound) {
			return false, true
		}
		addReadDiagnostic(&resp.Diagnostics, id, readErr, "before delete")
		return false, false
	}
	if currentResponse.Data.ActiveRolloutID() != "" {
		return true, waitForTrafficConfigRolloutBeforeDelete(ctx, resp, apiClient, objectName, id, currentResponse.Data.ActiveRolloutID())
	}
	if trafficConfigResponseDeploymentState(currentResponse) == "DEPLOYED" && canUndeploy(currentResponse) {
		return true, undeployTrafficConfigBeforeDelete(ctx, resp, apiClient, objectName, id, undeploy)
	}
	if originalErr == nil {
		return false, true
	}

	addTrafficConfigDeleteDiagnostic(resp, objectName, originalErr)
	return false, false
}

func trafficConfigResponseDeploymentState(tc *client.TrafficConfigResponse) string {
	if tc.Data.Attributes.Deployment == nil || tc.Data.Attributes.Deployment.State == nil {
		return ""
	}
	return *tc.Data.Attributes.Deployment.State
}

func undeployTrafficConfigBeforeDelete(
	ctx context.Context,
	resp *resource.DeleteResponse,
	apiClient *client.Client,
	objectName string,
	id string,
	undeploy trafficConfigUndeployFunc,
) bool {
	activeRolloutID, err := undeploy(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to undeploy %s traffic config before delete, got error: %s", objectName, err))
		return false
	}
	return waitForTrafficConfigRolloutBeforeDelete(ctx, resp, apiClient, objectName, id, activeRolloutID)
}

func waitForTrafficConfigRolloutBeforeDelete(
	ctx context.Context,
	resp *resource.DeleteResponse,
	apiClient *client.Client,
	objectName string,
	id string,
	rolloutID string,
) bool {
	if err := apiClient.WaitForTrafficConfigRollout(ctx, id, rolloutID); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for %s traffic config to finish undeploying before delete, got error: %s", objectName, err))
		return false
	}
	return true
}

func addTrafficConfigDeleteDiagnostic(resp *resource.DeleteResponse, objectName string, err error) {
	resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete %s traffic config, got error: %s", objectName, err))
}
