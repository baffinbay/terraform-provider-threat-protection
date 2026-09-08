package provider

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func addFrontendBindingConflictDiagnostic(diags errorDiagnostics, err error, operation, objectName, terraformResourceType, resourceName, binding string) bool {
	if !client.IsHTTPStatus(err, http.StatusConflict) || binding == "" {
		return false
	}

	object := objectName
	if resourceName != "" {
		object = fmt.Sprintf("%s %q", objectName, resourceName)
	}

	diags.AddError(
		"Frontend Binding Conflict",
		fmt.Sprintf(
			"Unable to %s %s because the frontend binding %s is already used by another traffic config.\n\nUse a different frontend IP, port, or protocol; delete or undeploy the conflicting traffic config; or import the existing traffic config if Terraform should manage it:\n\nterraform import %s.<resource_name> <traffic-config-id>\n\nOriginal error: %s",
			operation,
			object,
			binding,
			terraformResourceType,
			err,
		),
	)
	return true
}

func httpProxyFrontendBindingDescription(data HTTPProxyResourceModel) string {
	if data.Frontend == nil {
		return ""
	}

	details := []string{}
	if connectionType := stringValue(data.Frontend.ConnectionType); connectionType != "" {
		details = append(details, connectionType)
	}
	if data.ProtocolSettings != nil {
		if version := stringValue(data.ProtocolSettings.Version); version != "" {
			details = append(details, version)
		}
	}

	return frontendBindingDescription(
		[]string{stringValue(data.Frontend.IPv4), stringValue(data.Frontend.IPv6)},
		data.Frontend.Port,
		details,
	)
}

func l4ProxyFrontendBindingDescription(data L4ProxyResourceModel) string {
	if data.Frontend == nil {
		return ""
	}

	protocols := stringValues(data.Protocols)
	if len(protocols) == 0 {
		protocols = []string{"TCP"}
	}

	return frontendBindingDescription(
		[]string{stringValue(data.Frontend.IPv4), stringValue(data.Frontend.IPv6)},
		data.Frontend.Port,
		[]string{strings.Join(protocols, "/")},
	)
}

func frontendBindingDescription(addresses []string, port types.Int64, details []string) string {
	endpoints := []string{}
	for _, address := range addresses {
		if address == "" {
			continue
		}
		endpoints = append(endpoints, formatEndpoint(address, port))
	}
	if len(endpoints) == 0 {
		return ""
	}

	description := strings.Join(endpoints, " and ")
	if len(details) > 0 {
		description += " (" + strings.Join(details, ", ") + ")"
	}
	return description
}

func formatEndpoint(address string, port types.Int64) string {
	if port.IsNull() || port.IsUnknown() {
		return address
	}
	return net.JoinHostPort(address, strconv.FormatInt(port.ValueInt64(), 10))
}

func stringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func stringValues(values []types.String) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if v := stringValue(value); v != "" {
			result = append(result, v)
		}
	}
	return result
}
