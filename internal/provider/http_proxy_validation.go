package provider

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *HTTPProxyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data HTTPProxyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateHTTPProxyFrontend(data, resp)
	validateHTTPProxyBackend(data, resp)
	validateHTTPProxyCustomPages(data.CustomPages, resp)
	validateHTTPProxyWAF(data.WAF, resp)
	validateHTTPProxyAccessControl(data.IPBasedAccessControl, resp)
	validateHTTPProxyTrafficRules(data.TrafficRules, resp)
}

func validateHTTPProxyFrontend(data HTTPProxyResourceModel, resp *resource.ValidateConfigResponse) {
	if data.Frontend == nil {
		return
	}
	frontend := data.Frontend
	frontendPath := path.Root("frontend")
	if knownEmpty(frontend.IPv4) && knownEmpty(frontend.IPv6) {
		resp.Diagnostics.AddAttributeError(frontendPath.AtName("ipv4"), "Missing Frontend IP", "At least one of frontend.ipv4 or frontend.ipv6 must be configured.")
	}
	if frontend.ConnectionType.IsNull() || frontend.ConnectionType.IsUnknown() {
		return
	}

	secure := frontend.ConnectionType.ValueString() == "SECURE"
	if data.ProtocolSettings != nil && knownString(data.ProtocolSettings.Version) {
		if data.ProtocolSettings.Version.ValueString() == "HTTP2.0" && !secure {
			resp.Diagnostics.AddAttributeError(path.Root("protocol_settings").AtName("version"), "Invalid HTTP/2 Frontend", "HTTP/2 requires frontend.connection_type to be SECURE.")
		}
		if data.ProtocolSettings.Version.ValueString() == "HTTP2.0" && knownBool(data.ProtocolSettings.EnableWebsockets) && data.ProtocolSettings.EnableWebsockets.ValueBool() {
			resp.Diagnostics.AddAttributeError(path.Root("protocol_settings").AtName("enable_websockets"), "Invalid WebSocket Setting", "enable_websockets cannot be true when protocol_settings.version is HTTP2.0.")
		}
	}

	if !secure {
		if !frontend.RedirectHTTP.IsNull() {
			resp.Diagnostics.AddAttributeError(frontendPath.AtName("redirect_http"), "Invalid Plaintext Frontend Setting", "redirect_http is only valid for SECURE frontends.")
		}
		if frontend.HSTS != nil {
			resp.Diagnostics.AddAttributeError(frontendPath.AtName("hsts"), "Invalid Plaintext Frontend Setting", "hsts is only valid for SECURE frontends.")
		}
		if frontend.ClientCertificateVerification != nil {
			resp.Diagnostics.AddAttributeError(frontendPath.AtName("client_certificate_verification"), "Invalid Plaintext Frontend Setting", "client_certificate_verification is only valid for SECURE frontends.")
		}
		for i, host := range frontend.Hosts {
			if !host.CertificateID.IsNull() || !host.TLSConfig.IsNull() {
				resp.Diagnostics.AddAttributeError(frontendPath.AtName("hosts").AtListIndex(i), "Invalid Plaintext Host", "certificate_id and tls_config are only valid for SECURE frontend hosts.")
			}
		}
		return
	}

	for i, host := range frontend.Hosts {
		hostPath := frontendPath.AtName("hosts").AtListIndex(i)
		if knownEmpty(host.CertificateID) {
			resp.Diagnostics.AddAttributeError(hostPath.AtName("certificate_id"), "Missing Frontend Certificate", "SECURE frontend hosts require certificate_id.")
		} else {
			validateUUID(host.CertificateID, hostPath.AtName("certificate_id"), "Frontend Certificate ID", resp)
		}
		if knownEmpty(host.TLSConfig) {
			resp.Diagnostics.AddAttributeError(hostPath.AtName("tls_config"), "Missing TLS Configuration", "SECURE frontend hosts require tls_config.")
		}
	}
	if knownBool(frontend.RedirectHTTP) && frontend.RedirectHTTP.ValueBool() && knownInt64(frontend.Port) && frontend.Port.ValueInt64() == 80 {
		resp.Diagnostics.AddAttributeError(frontendPath.AtName("redirect_http"), "Invalid Secure Redirect", "redirect_http cannot be enabled when the secure frontend port is 80.")
	}
	if frontend.ClientCertificateVerification != nil && knownString(frontend.ClientCertificateVerification.Mode) && frontend.ClientCertificateVerification.Mode.ValueString() == "VERIFY_AND_REJECT" {
		if len(frontend.ClientCertificateVerification.CACertificateIDs) == 0 {
			resp.Diagnostics.AddAttributeError(frontendPath.AtName("client_certificate_verification").AtName("ca_certificate_ids"), "Missing CA Certificates", "VERIFY_AND_REJECT requires at least one ca_certificate_id.")
		}
		for i, id := range frontend.ClientCertificateVerification.CACertificateIDs {
			validateUUID(id, frontendPath.AtName("client_certificate_verification").AtName("ca_certificate_ids").AtListIndex(i), "CA Certificate ID", resp)
		}
	}
}

func validateHTTPProxyBackend(data HTTPProxyResourceModel, resp *resource.ValidateConfigResponse) {
	if data.Backend == nil {
		return
	}
	backendPath := path.Root("backend")
	secure := data.Frontend != nil && knownString(data.Frontend.ConnectionType) && data.Frontend.ConnectionType.ValueString() == "SECURE"
	if !secure && data.Backend.TLSSettings != nil {
		resp.Diagnostics.AddAttributeError(backendPath.AtName("tls_settings"), "Invalid Plaintext Backend TLS", "backend.tls_settings is only used when frontend.connection_type is SECURE.")
		return
	}
	if data.Backend.TLSSettings == nil {
		return
	}
	if !data.Backend.TLSSettings.ClientCertificateID.IsNull() {
		validateUUID(data.Backend.TLSSettings.ClientCertificateID, backendPath.AtName("tls_settings").AtName("client_certificate_id"), "Backend Client Certificate ID", resp)
	}
	verify := data.Backend.TLSSettings.VerifyCertificate
	if verify == nil || !knownString(verify.Mode) || verify.Mode.ValueString() != "CUSTOM_TRUSTSTORE" {
		return
	}
	verifyPath := backendPath.AtName("tls_settings").AtName("verify_certificate").AtName("ca_certificate_ids")
	if len(verify.CACertificateIDs) == 0 {
		resp.Diagnostics.AddAttributeError(verifyPath, "Missing CA Certificates", "CUSTOM_TRUSTSTORE requires at least one ca_certificate_id.")
	}
	for i, id := range verify.CACertificateIDs {
		validateUUID(id, verifyPath.AtListIndex(i), "CA Certificate ID", resp)
	}
}

func validateHTTPProxyCustomPages(pages []HTTPProxyCustomPageModel, resp *resource.ValidateConfigResponse) {
	seenTypes := map[string]struct{}{}
	for i, page := range pages {
		pagePath := path.Root("custom_pages").AtListIndex(i)
		validateUUID(page.ID, pagePath.AtName("id"), "Custom Page ID", resp)
		if knownString(page.Type) {
			value := page.Type.ValueString()
			if _, exists := seenTypes[value]; exists {
				resp.Diagnostics.AddAttributeError(pagePath.AtName("type"), "Duplicate Custom Page Type", fmt.Sprintf("custom_pages type %q can only be configured once.", value))
			}
			seenTypes[value] = struct{}{}
		}
	}
}

func validateHTTPProxyWAF(waf *HTTPProxyWAFModel, resp *resource.ValidateConfigResponse) {
	if waf == nil {
		return
	}
	wafPath := path.Root("waf")
	if waf.SourceExclusions != nil && knownBool(waf.SourceExclusions.Enabled) && waf.SourceExclusions.Enabled.ValueBool() && len(waf.SourceExclusions.Sources) == 0 {
		resp.Diagnostics.AddAttributeError(wafPath.AtName("source_exclusions").AtName("sources"), "Missing WAF Source Exclusions", "At least one source CIDR is required when source exclusions are enabled.")
	}
	if waf.SourceExclusions != nil {
		for i, source := range waf.SourceExclusions.Sources {
			validateCIDR(source, wafPath.AtName("source_exclusions").AtName("sources").AtListIndex(i), "WAF Source Exclusion", resp)
		}
	}
	if waf.HTTPCompliance != nil {
		methods := map[string]bool{}
		for _, method := range waf.HTTPCompliance.AllowedMethods {
			if knownString(method) {
				methods[method.ValueString()] = true
			}
		}
		if len(waf.HTTPCompliance.AllowedMethods) > 0 && (!methods["GET"] || !methods["POST"]) {
			resp.Diagnostics.AddAttributeError(wafPath.AtName("http_compliance").AtName("allowed_methods"), "Invalid HTTP Compliance Methods", "allowed_methods must include both GET and POST.")
		}
	}
	validateHTTPProxyWAFExclusions(waf.PathExclusions, waf.CookieExclusions, wafPath, resp)

	if waf.StagedWAF == nil {
		return
	}
	staged := waf.StagedWAF
	stagedPath := wafPath.AtName("staged_waf")
	if knownString(staged.State) && staged.State.ValueString() == "ENABLED" && knownString(waf.Enforcement) && waf.Enforcement.ValueString() != "BLOCK" {
		resp.Diagnostics.AddAttributeError(wafPath.AtName("enforcement"), "Invalid Staged WAF Configuration", "waf.enforcement must be BLOCK when staged_waf.state is ENABLED.")
	}
	if knownString(staged.Mode) {
		switch staged.Mode.ValueString() {
		case "PARANOIA_LEVEL":
			if staged.ParanoiaLevel.IsNull() {
				resp.Diagnostics.AddAttributeError(stagedPath.AtName("paranoia_level"), "Missing Staged WAF Paranoia Level", "paranoia_level is required when staged_waf.mode is PARANOIA_LEVEL.")
			}
			if !staged.CoreRuleSetVersion.IsNull() {
				resp.Diagnostics.AddAttributeError(stagedPath.AtName("core_rule_set_version"), "Invalid Staged WAF Setting", "core_rule_set_version is only valid when staged_waf.mode is CRS_VERSION.")
			}
		case "CRS_VERSION":
			if knownEmpty(staged.CoreRuleSetVersion) {
				resp.Diagnostics.AddAttributeError(stagedPath.AtName("core_rule_set_version"), "Missing Staged WAF CRS Version", "core_rule_set_version is required when staged_waf.mode is CRS_VERSION.")
			}
			if !staged.ParanoiaLevel.IsNull() {
				resp.Diagnostics.AddAttributeError(stagedPath.AtName("paranoia_level"), "Invalid Staged WAF Setting", "paranoia_level is only valid when staged_waf.mode is PARANOIA_LEVEL.")
			}
		}
	}
	validateHTTPProxyWAFExclusions(staged.PathExclusions, staged.CookieExclusions, stagedPath, resp)
}

func validateHTTPProxyWAFExclusions(paths []HTTPProxyWAFPathExclusionModel, cookies []HTTPProxyWAFCookieExclusionModel, base path.Path, resp *resource.ValidateConfigResponse) {
	for i, exclusion := range paths {
		itemPath := base.AtName("path_exclusions").AtListIndex(i)
		if knownString(exclusion.Match) && !strings.HasPrefix(exclusion.Match.ValueString(), "/") {
			resp.Diagnostics.AddAttributeError(itemPath.AtName("match"), "Invalid WAF Exclusion Path", "WAF path exclusion matches must begin with '/'.")
		}
		for j, id := range exclusion.RuleIDs {
			validateUUID(id, itemPath.AtName("rule_ids").AtListIndex(j), "WAF Rule ID", resp)
		}
	}
	for i, exclusion := range cookies {
		itemPath := base.AtName("cookie_exclusions").AtListIndex(i)
		for j, id := range exclusion.RuleIDs {
			validateUUID(id, itemPath.AtName("rule_ids").AtListIndex(j), "WAF Rule ID", resp)
		}
	}
}

func validateHTTPProxyAccessControl(access *HTTPProxyIPAccessControlModel, resp *resource.ValidateConfigResponse) {
	if access == nil || access.Rules == nil {
		return
	}
	base := path.Root("ip_based_access_control")
	hasAllow := false
	seen := map[string]map[string]struct{}{}
	checkUnique := func(kind, value string, itemPath path.Path) {
		if _, ok := seen[kind]; !ok {
			seen[kind] = map[string]struct{}{}
		}
		if _, ok := seen[kind][value]; ok {
			resp.Diagnostics.AddAttributeError(itemPath, "Duplicate Access Control Rule", fmt.Sprintf("%s value %q can only be configured once.", kind, value))
		}
		seen[kind][value] = struct{}{}
	}
	for i, rule := range access.Rules.IPRanges {
		if rule.Policy.ValueString() == "ALLOW" {
			hasAllow = true
		}
		checkUnique("IP range", rule.Address.ValueString(), base.AtName("rules").AtName("ip_ranges").AtListIndex(i).AtName("address"))
		validateCIDR(rule.Address, base.AtName("rules").AtName("ip_ranges").AtListIndex(i).AtName("address"), "IP Range", resp)
	}
	for i, rule := range access.Rules.IPLists {
		if rule.Policy.ValueString() == "ALLOW" {
			hasAllow = true
		}
		checkUnique("IP list", rule.ID.ValueString(), base.AtName("rules").AtName("ip_lists").AtListIndex(i).AtName("id"))
		validateUUID(rule.ID, base.AtName("rules").AtName("ip_lists").AtListIndex(i).AtName("id"), "IP List ID", resp)
	}
	for i, rule := range access.Rules.KnownServices {
		if rule.Policy.ValueString() == "ALLOW" {
			hasAllow = true
		}
		checkUnique("known service", rule.ID.ValueString(), base.AtName("rules").AtName("known_services").AtListIndex(i).AtName("id"))
	}
	for i, rule := range access.Rules.GeoLocations {
		if rule.Policy.ValueString() == "ALLOW" {
			hasAllow = true
		}
		checkUnique("geo location", rule.Region.ValueString(), base.AtName("rules").AtName("geo_locations").AtListIndex(i).AtName("region"))
	}
	for i, rule := range access.Rules.ASNs {
		if rule.Policy.ValueString() == "ALLOW" {
			hasAllow = true
		}
		checkUnique("ASN", fmt.Sprintf("%d", rule.ASN.ValueInt64()), base.AtName("rules").AtName("asns").AtListIndex(i).AtName("asn"))
	}
	if knownString(access.DefaultPolicy) && access.DefaultPolicy.ValueString() == "BLOCK" && !hasAllow {
		resp.Diagnostics.AddAttributeError(base.AtName("rules"), "Invalid Access Control Default Policy", "At least one ALLOW rule is required when default_policy is BLOCK.")
	}
}

func validateHTTPProxyTrafficRules(rules []HTTPProxyTrafficRuleModel, resp *resource.ValidateConfigResponse) {
	seenNames := map[string]struct{}{}
	seenRuleMatches := map[string]struct{}{}
	for i, rule := range rules {
		rulePath := path.Root("traffic_rules").AtListIndex(i)
		name := rule.Name.ValueString()
		if _, exists := seenNames[name]; exists {
			resp.Diagnostics.AddAttributeError(rulePath.AtName("name"), "Duplicate Traffic Rule Name", fmt.Sprintf("Traffic rule name %q must be unique.", name))
		}
		seenNames[name] = struct{}{}

		conditionKeys := make([]string, 0, len(rule.MatchingConditions))
		seenConditions := map[string]struct{}{}
		for j, condition := range rule.MatchingConditions {
			conditionPath := rulePath.AtName("matching_conditions").AtListIndex(j)
			paths := httpProxyStringValues(condition.Paths)
			for k, matchPath := range paths {
				if !strings.HasPrefix(matchPath, "/") || strings.Contains(matchPath, " ") {
					resp.Diagnostics.AddAttributeError(conditionPath.AtName("paths").AtListIndex(k), "Invalid Traffic Rule Path", "Traffic rule paths must begin with '/' and cannot contain spaces.")
				}
			}
			sort.Strings(paths)
			hostKey := ""
			if condition.Hosts != nil {
				hostKey = condition.Hosts.Type.ValueString()
				values := httpProxyStringValues(condition.Hosts.Values)
				sort.Strings(values)
				if hostKey == "SELECT" && len(values) == 0 {
					resp.Diagnostics.AddAttributeError(conditionPath.AtName("hosts").AtName("values"), "Missing Selected Hosts", "hosts.values must contain at least one hostname when hosts.type is SELECT.")
				}
				if hostKey == "ALL" && len(values) > 0 {
					resp.Diagnostics.AddAttributeError(conditionPath.AtName("hosts").AtName("values"), "Invalid Host Selector", "hosts.values must be omitted when hosts.type is ALL.")
				}
				hostKey += ":" + strings.Join(values, ",")
			}
			key := hostKey + "|" + strings.Join(paths, ",")
			if _, exists := seenConditions[key]; exists {
				resp.Diagnostics.AddAttributeError(conditionPath, "Duplicate Matching Condition", "Matching host and path sets must be unique within a traffic rule.")
			}
			seenConditions[key] = struct{}{}
			conditionKeys = append(conditionKeys, key)
		}
		sort.Strings(conditionKeys)
		ruleMatchKey := strings.Join(conditionKeys, ";")
		if _, exists := seenRuleMatches[ruleMatchKey]; exists {
			resp.Diagnostics.AddAttributeError(rulePath.AtName("matching_conditions"), "Duplicate Traffic Rule Match", "Matching conditions must be unique across traffic rules.")
		}
		seenRuleMatches[ruleMatchKey] = struct{}{}
		validateHTTPProxyTrafficRuleActions(rule.Actions, rulePath.AtName("actions"), resp)
	}
}

func validateHTTPProxyTrafficRuleActions(actions *HTTPProxyTrafficRuleActionsModel, actionPath path.Path, resp *resource.ValidateConfigResponse) {
	if actions == nil {
		return
	}
	actionCount := 0
	if len(actions.Backends) > 0 {
		actionCount++
	}
	if len(actions.Headers) > 0 {
		actionCount++
	}
	if !actions.HostHeader.IsNull() {
		actionCount++
	}
	if actions.Redirect != nil {
		actionCount++
	}
	if actions.RateLimit != nil {
		actionCount++
		if actions.RateLimit.BySourceIP == nil || actions.RateLimit.BySourceIPAndURL == nil {
			resp.Diagnostics.AddAttributeError(actionPath.AtName("rate_limit"), "Incomplete Traffic Rule Rate Limit", "rate_limit requires both by_source_ip and by_source_ip_and_url.")
		}
	}
	if actions.MaxBodySize != nil {
		actionCount++
	}
	if actions.BotProtection != nil {
		actionCount++
	}
	if actionCount == 0 {
		resp.Diagnostics.AddAttributeError(actionPath, "Missing Traffic Rule Action", "At least one traffic rule action must be configured.")
	}
	seenHeaders := map[string]struct{}{}
	for i, header := range actions.Headers {
		key := strings.ToLower(header.Key.ValueString())
		headerPath := actionPath.AtName("headers").AtListIndex(i).AtName("key")
		if key == "host" {
			resp.Diagnostics.AddAttributeError(headerPath, "Invalid Header Action", "Use host_header instead of setting the Host header directly.")
		}
		if _, exists := seenHeaders[key]; exists {
			resp.Diagnostics.AddAttributeError(headerPath, "Duplicate Header Action", "Traffic rule header keys must be unique.")
		}
		seenHeaders[key] = struct{}{}
	}
	if actions.Redirect != nil && knownString(actions.Redirect.URL) {
		parsed, err := url.ParseRequestURI(actions.Redirect.URL.ValueString())
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			resp.Diagnostics.AddAttributeError(actionPath.AtName("redirect").AtName("url"), "Invalid Redirect URL", "Traffic rule redirect.url must be an absolute HTTP or HTTPS URL.")
		}
	}
	if actions.MaxBodySize != nil && knownString(actions.MaxBodySize.Enforcement) {
		if actions.MaxBodySize.Enforcement.ValueString() == "ENABLED" && actions.MaxBodySize.ValueBytes.IsNull() {
			resp.Diagnostics.AddAttributeError(actionPath.AtName("max_body_size").AtName("value_bytes"), "Missing Maximum Body Size", "value_bytes is required when max_body_size.enforcement is ENABLED.")
		}
		if actions.MaxBodySize.Enforcement.ValueString() == "DISABLED" && !actions.MaxBodySize.ValueBytes.IsNull() {
			resp.Diagnostics.AddAttributeError(actionPath.AtName("max_body_size").AtName("value_bytes"), "Invalid Maximum Body Size", "value_bytes must be omitted when max_body_size.enforcement is DISABLED.")
		}
	}
}

func validateUUID(value types.String, attributePath path.Path, label string, resp *resource.ValidateConfigResponse) {
	if !knownString(value) || value.ValueString() == "" {
		return
	}
	if _, err := uuid.Parse(value.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(attributePath, "Invalid "+label, fmt.Sprintf("%s must be a UUID.", label))
	}
}

func validateCIDR(value types.String, attributePath path.Path, label string, resp *resource.ValidateConfigResponse) {
	if !knownString(value) || value.ValueString() == "" {
		return
	}
	if _, _, err := net.ParseCIDR(value.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(attributePath, "Invalid "+label, fmt.Sprintf("%s must be a valid IPv4 or IPv6 CIDR.", label))
	}
}

func knownString(value types.String) bool { return !value.IsNull() && !value.IsUnknown() }
func knownBool(value types.Bool) bool     { return !value.IsNull() && !value.IsUnknown() }
func knownInt64(value types.Int64) bool   { return !value.IsNull() && !value.IsUnknown() }
func knownEmpty(value types.String) bool {
	return value.IsNull() || (knownString(value) && value.ValueString() == "")
}
