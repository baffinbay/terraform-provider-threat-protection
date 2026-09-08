package provider

import (
	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mapHTTPProxyModelToCreateRequest(data HTTPProxyResourceModel) client.HTTPProxyRequest {
	return newHTTPProxyRequest(mapHTTPProxyManagedAttributes(data))
}

func mapHTTPProxyModelToUpdateRequest(data HTTPProxyResourceModel) client.HTTPProxyRequest {
	return newHTTPProxyRequest(mapHTTPProxyManagedAttributes(data))
}

func newHTTPProxyRequest(attributes client.HTTPProxyAttributes) client.HTTPProxyRequest {
	var req client.HTTPProxyRequest
	req.Data.Type = httpProxyTrafficConfigType
	req.Data.Attributes = attributes
	return req
}

func mapHTTPProxyManagedAttributes(data HTTPProxyResourceModel) client.HTTPProxyAttributes {
	attributes := client.HTTPProxyAttributes{
		Name:                   data.Name.ValueString(),
		Version:                httpProxyTrafficConfigVersion,
		Frontend:               mapHTTPProxyFrontendToAPI(data.Frontend),
		Backend:                mapHTTPProxyBackendToAPI(data.Backend, data.Frontend),
		ProtocolSettings:       mapHTTPProxyProtocolSettingsToAPI(data.ProtocolSettings),
		Deployment:             &client.HTTPProxyDeployment{State: data.DeploymentState.ValueString()},
		WAF:                    mapHTTPProxyWAFToAPI(data.WAF),
		IPBasedAccessControl:   mapHTTPProxyIPAccessControlToAPI(data.IPBasedAccessControl),
		RateLimiting:           mapHTTPProxyRateLimitingToAPI(data.RateLimiting),
		TrafficRules:           mapHTTPProxyTrafficRulesToAPI(data.TrafficRules),
		DataProtection:         mapHTTPProxyDataProtectionToAPI(data.DataProtection),
		CustomPages:            mapHTTPProxyCustomPagesToAPI(data.CustomPages),
		BotProtection:          mapHTTPProxyBotProtectionToAPI(data.BotProtection),
		ConnectionReuseEnabled: boolValuePointer(data.ConnectionReuse),
	}
	return attributes
}

func mapHTTPProxyFrontendToAPI(data *HTTPProxyFrontendModel) *client.HTTPProxyFrontend {
	if data == nil {
		return nil
	}
	frontend := &client.HTTPProxyFrontend{
		ConnectionType: data.ConnectionType.ValueString(),
		IPv4:           data.IPv4.ValueString(),
		IPv6:           data.IPv6.ValueString(),
		Port:           data.Port.ValueInt64(),
		Hosts:          make([]client.HTTPProxyFrontendHost, 0, len(data.Hosts)),
	}
	for _, host := range data.Hosts {
		apiHost := client.HTTPProxyFrontendHost{Host: host.Host.ValueString()}
		if frontend.ConnectionType == "SECURE" {
			apiHost.CertificateID = host.CertificateID.ValueString()
			apiHost.TLSConfig = host.TLSConfig.ValueString()
		}
		frontend.Hosts = append(frontend.Hosts, apiHost)
	}
	if frontend.ConnectionType == "SECURE" {
		frontend.RedirectHTTP = boolValuePointer(data.RedirectHTTP)
		if data.HSTS != nil {
			frontend.HTTPStrictTransportSecurity = &client.HTTPProxyHSTS{
				Enabled:           data.HSTS.Enabled.ValueBool(),
				MaxAge:            data.HSTS.MaxAge.ValueInt64(),
				IncludeSubdomains: data.HSTS.IncludeSubdomains.ValueBool(),
				Preload:           data.HSTS.Preload.ValueBool(),
			}
		}
		if data.ClientCertificateVerification != nil {
			frontend.ClientCertificateVerification = &client.HTTPProxyClientCertificateVerification{
				Mode:             data.ClientCertificateVerification.Mode.ValueString(),
				CACertificateIDs: httpProxyStringValues(data.ClientCertificateVerification.CACertificateIDs),
			}
		}
	}
	return frontend
}

func mapHTTPProxyBackendToAPI(data *HTTPProxyBackendModel, frontend *HTTPProxyFrontendModel) *client.HTTPProxyBackend {
	if data == nil {
		return nil
	}
	backend := &client.HTTPProxyBackend{
		Hosts:          make([]client.HTTPProxyBackendHost, 0, len(data.Hosts)),
		DeliveryMethod: data.DeliveryMethod.ValueString(),
		ServerName:     data.ServerName.ValueString(),
	}
	for _, host := range data.Hosts {
		backend.Hosts = append(backend.Hosts, client.HTTPProxyBackendHost{Address: host.Address.ValueString(), Port: host.Port.ValueInt64()})
	}
	if frontend != nil && frontend.ConnectionType.ValueString() == "SECURE" && data.TLSSettings != nil {
		backend.TLSSettings = &client.HTTPProxyTLSSettings{ClientCertificateID: data.TLSSettings.ClientCertificateID.ValueString()}
		if data.TLSSettings.VerifyCertificate != nil {
			backend.TLSSettings.VerifyCertificate = &client.HTTPProxyBackendVerifyCertificate{
				Mode:             data.TLSSettings.VerifyCertificate.Mode.ValueString(),
				CACertificateIDs: httpProxyStringValues(data.TLSSettings.VerifyCertificate.CACertificateIDs),
			}
		}
	}
	return backend
}

func mapHTTPProxyProtocolSettingsToAPI(data *HTTPProxyProtocolSettingsModel) *client.HTTPProxyProtocolSettings {
	if data == nil {
		return nil
	}
	settings := &client.HTTPProxyProtocolSettings{HTTPVersion: data.Version.ValueString()}
	if settings.HTTPVersion == "HTTP1.1" {
		settings.EnableWebsockets = boolValuePointer(data.EnableWebsockets)
	}
	return settings
}

func mapHTTPProxyBotProtectionToAPI(data *HTTPProxyBotProtectionModel) *client.HTTPProxyBotProtection {
	if data == nil {
		return nil
	}
	return &client.HTTPProxyBotProtection{Strategy: data.Strategy.ValueString(), ChallengeType: data.ChallengeType.ValueString()}
}

func mapHTTPProxyDataProtectionToAPI(data *HTTPProxyDataProtectionModel) *client.HTTPProxyDataProtection {
	if data == nil || data.LogRedaction == nil {
		return nil
	}
	return &client.HTTPProxyDataProtection{LogRedaction: client.HTTPProxyLogRedaction{
		Headers: httpProxyStringValues(data.LogRedaction.Headers),
		Cookies: httpProxyStringValues(data.LogRedaction.Cookies),
	}}
}

func mapHTTPProxyCustomPagesToAPI(data []HTTPProxyCustomPageModel) *[]client.HTTPProxyCustomPage {
	if data == nil {
		return nil
	}
	pages := make([]client.HTTPProxyCustomPage, 0, len(data))
	for _, page := range data {
		pages = append(pages, client.HTTPProxyCustomPage{ID: page.ID.ValueString(), Type: page.Type.ValueString()})
	}
	return &pages
}

func mapHTTPProxyRateLimitingToAPI(data *HTTPProxyRateLimitingModel) *client.HTTPProxyRateLimiting {
	if data == nil || data.BySourceIP == nil || data.BySourceIPAndURL == nil {
		return nil
	}
	return &client.HTTPProxyRateLimiting{
		BySourceIP:       mapHTTPProxyRateLimitRuleToAPI(data.BySourceIP),
		BySourceIPAndURL: mapHTTPProxyRateLimitRuleToAPI(data.BySourceIPAndURL),
		Exclusions:       httpProxyStringValues(data.Exclusions),
	}
}

func mapHTTPProxyRateLimitRuleToAPI(data *HTTPProxyRateLimitRuleModel) client.HTTPProxyRateLimitRule {
	rule := client.HTTPProxyRateLimitRule{Enforcement: data.Enforcement.ValueString(), Burst: data.Burst.ValueInt64()}
	if data.Rate != nil {
		rule.Rate = client.HTTPProxyRateLimitRate{Value: data.Rate.Value.ValueInt64(), Unit: data.Rate.Unit.ValueString()}
	}
	return rule
}

func mapHTTPProxyIPAccessControlToAPI(data *HTTPProxyIPAccessControlModel) *client.HTTPProxyIPBasedAccessControl {
	if data == nil || data.Rules == nil {
		return nil
	}
	result := &client.HTTPProxyIPBasedAccessControl{
		DefaultPolicy: data.DefaultPolicy.ValueString(),
		Rules: client.HTTPProxyIPBasedAccessRuleGroup{
			IPRanges:      make([]client.HTTPProxyIPRangeRule, 0, len(data.Rules.IPRanges)),
			IPLists:       make([]client.HTTPProxyIPListRule, 0, len(data.Rules.IPLists)),
			KnownServices: make([]client.HTTPProxyKnownServiceRule, 0, len(data.Rules.KnownServices)),
			GeoLocations:  make([]client.HTTPProxyGeoLocationRule, 0, len(data.Rules.GeoLocations)),
			ASNs:          make([]client.HTTPProxyASNRule, 0, len(data.Rules.ASNs)),
		},
	}
	for _, rule := range data.Rules.IPRanges {
		result.Rules.IPRanges = append(result.Rules.IPRanges, client.HTTPProxyIPRangeRule{Policy: rule.Policy.ValueString(), Address: rule.Address.ValueString(), Note: rule.Note.ValueString(), BypassProtection: client.HTTPProxyBypassProtection{BotProtection: rule.BypassBotProtection.ValueBool()}})
	}
	for _, rule := range data.Rules.IPLists {
		result.Rules.IPLists = append(result.Rules.IPLists, client.HTTPProxyIPListRule{Policy: rule.Policy.ValueString(), ID: rule.ID.ValueString(), BypassProtection: client.HTTPProxyBypassProtection{BotProtection: rule.BypassBotProtection.ValueBool()}})
	}
	for _, rule := range data.Rules.KnownServices {
		result.Rules.KnownServices = append(result.Rules.KnownServices, client.HTTPProxyKnownServiceRule{Policy: rule.Policy.ValueString(), ID: rule.ID.ValueString(), Note: rule.Note.ValueString(), BypassProtection: client.HTTPProxyBypassProtection{BotProtection: rule.BypassBotProtection.ValueBool()}})
	}
	for _, rule := range data.Rules.GeoLocations {
		result.Rules.GeoLocations = append(result.Rules.GeoLocations, client.HTTPProxyGeoLocationRule{Policy: rule.Policy.ValueString(), Region: rule.Region.ValueString(), Note: rule.Note.ValueString()})
	}
	for _, rule := range data.Rules.ASNs {
		result.Rules.ASNs = append(result.Rules.ASNs, client.HTTPProxyASNRule{Policy: rule.Policy.ValueString(), ASN: rule.ASN.ValueInt64(), Note: rule.Note.ValueString()})
	}
	return result
}

func mapHTTPProxyWAFToAPI(data *HTTPProxyWAFModel) *client.HTTPProxyWAF {
	if data == nil || data.SourceExclusions == nil || data.HTTPCompliance == nil || data.HTTPCompliance.ParameterLimit == nil {
		return nil
	}
	waf := &client.HTTPProxyWAF{
		Enforcement:        data.Enforcement.ValueString(),
		ParanoiaLevel:      data.ParanoiaLevel.ValueInt64(),
		CoreRuleSet:        client.HTTPProxyCoreRuleSet{Version: data.CoreRuleSetVersion.ValueString()},
		MatchedDataEnabled: data.MatchedDataEnabled.ValueBool(),
		SourceExclusions: client.HTTPProxySourceExclusions{
			Enabled: data.SourceExclusions.Enabled.ValueBool(),
			Sources: httpProxyStringValues(data.SourceExclusions.Sources),
		},
		HTTPCompliance: client.HTTPProxyHTTPCompliance{
			GlobalConfig: client.HTTPProxyHTTPComplianceGlobalConfig{
				ParameterLimit:      client.HTTPProxyParameterLimit{Enabled: data.HTTPCompliance.ParameterLimit.Enabled.ValueBool(), Limit: data.HTTPCompliance.ParameterLimit.Limit.ValueInt64()},
				AllowedHTTPMethods:  httpProxyStringValues(data.HTTPCompliance.AllowedMethods),
				AllowedHTTPVersions: httpProxyStringValues(data.HTTPCompliance.AllowedVersions),
			},
			ResourceConfigs: make([]client.HTTPProxyHTTPComplianceResourceConfig, 0, len(data.HTTPCompliance.ResourceConfigs)),
		},
		PathExclusions:   mapHTTPProxyPathExclusionsToAPI(data.PathExclusions),
		CookieExclusions: mapHTTPProxyCookieExclusionsToAPI(data.CookieExclusions),
		StagedWAF:        mapHTTPProxyStagedWAFToAPI(data.StagedWAF),
	}
	for _, config := range data.HTTPCompliance.ResourceConfigs {
		matches := make([]client.HTTPProxyResourceMatch, 0, len(config.Matches))
		for _, match := range config.Matches {
			matches = append(matches, client.HTTPProxyResourceMatch{Path: match.ValueString()})
		}
		waf.HTTPCompliance.ResourceConfigs = append(waf.HTTPCompliance.ResourceConfigs, client.HTTPProxyHTTPComplianceResourceConfig{
			Matches: matches,
			Config:  client.HTTPProxyResourceConfig{ParseJSONEnabled: config.ParseJSONEnabled.ValueBool(), ParseXMLEnabled: config.ParseXMLEnabled.ValueBool(), ParseMultipartRequestEnabled: config.ParseMultipartRequestEnabled.ValueBool()},
		})
	}
	return waf
}

func mapHTTPProxyPathExclusionsToAPI(data []HTTPProxyWAFPathExclusionModel) []client.HTTPProxyPathExclusion {
	result := make([]client.HTTPProxyPathExclusion, 0, len(data))
	for _, exclusion := range data {
		rules := make([]client.HTTPProxyExclusionRule, 0, len(exclusion.RuleIDs))
		for _, id := range exclusion.RuleIDs {
			rules = append(rules, client.HTTPProxyExclusionRule{Type: "rule", ID: id.ValueString()})
		}
		result = append(result, client.HTTPProxyPathExclusion{Match: exclusion.Match.ValueString(), DisableAll: exclusion.DisableAll.ValueBool(), Rules: rules})
	}
	return result
}

func mapHTTPProxyCookieExclusionsToAPI(data []HTTPProxyWAFCookieExclusionModel) []client.HTTPProxyCookieExclusion {
	result := make([]client.HTTPProxyCookieExclusion, 0, len(data))
	for _, exclusion := range data {
		rules := make([]client.HTTPProxyExclusionRule, 0, len(exclusion.RuleIDs))
		for _, id := range exclusion.RuleIDs {
			rules = append(rules, client.HTTPProxyExclusionRule{Type: "rule", ID: id.ValueString()})
		}
		result = append(result, client.HTTPProxyCookieExclusion{CookieName: exclusion.CookieName.ValueString(), ExcludeAllRules: exclusion.ExcludeAllRules.ValueBool(), ExcludedRules: rules})
	}
	return result
}

func mapHTTPProxyStagedWAFToAPI(data *HTTPProxyStagedWAFModel) *client.HTTPProxyStagedWAF {
	if data == nil {
		return nil
	}
	result := &client.HTTPProxyStagedWAF{
		State:            data.State.ValueString(),
		Mode:             data.Mode.ValueString(),
		PathExclusions:   mapHTTPProxyPathExclusionsToAPI(data.PathExclusions),
		CookieExclusions: mapHTTPProxyCookieExclusionsToAPI(data.CookieExclusions),
	}
	if result.Mode == "PARANOIA_LEVEL" {
		value := data.ParanoiaLevel.ValueInt64()
		result.ParanoiaLevel = &value
	}
	if result.Mode == "CRS_VERSION" {
		result.CoreRuleSet = &client.HTTPProxyCoreRuleSet{Version: data.CoreRuleSetVersion.ValueString()}
	}
	return result
}

func mapHTTPProxyTrafficRulesToAPI(data []HTTPProxyTrafficRuleModel) *[]client.HTTPProxyTrafficRule {
	if data == nil {
		return nil
	}
	result := make([]client.HTTPProxyTrafficRule, 0, len(data))
	for _, rule := range data {
		apiRule := client.HTTPProxyTrafficRule{Name: rule.Name.ValueString(), MatchingConditions: make([]client.HTTPProxyTrafficMatchingCondition, 0, len(rule.MatchingConditions))}
		for _, condition := range rule.MatchingConditions {
			apiCondition := client.HTTPProxyTrafficMatchingCondition{Paths: httpProxyStringValues(condition.Paths)}
			if condition.Hosts != nil {
				apiCondition.Hosts = client.HTTPProxyTrafficRuleHostMatch{Type: condition.Hosts.Type.ValueString(), Values: httpProxyStringValues(condition.Hosts.Values)}
			}
			apiRule.MatchingConditions = append(apiRule.MatchingConditions, apiCondition)
		}
		apiRule.Actions = mapHTTPProxyTrafficRuleActionsToAPI(rule.Actions)
		result = append(result, apiRule)
	}
	return &result
}

func mapHTTPProxyTrafficRuleActionsToAPI(data *HTTPProxyTrafficRuleActionsModel) client.HTTPProxyTrafficRuleActions {
	actions := client.HTTPProxyTrafficRuleActions{SetBackends: []client.HTTPProxyBackendHost{}, SetHeaders: []client.HTTPProxyHeader{}}
	if data == nil {
		return actions
	}
	for _, backend := range data.Backends {
		actions.SetBackends = append(actions.SetBackends, client.HTTPProxyBackendHost{Address: backend.Address.ValueString(), Port: backend.Port.ValueInt64()})
	}
	for _, header := range data.Headers {
		actions.SetHeaders = append(actions.SetHeaders, client.HTTPProxyHeader{Key: header.Key.ValueString(), Value: header.Value.ValueString()})
	}
	if !data.HostHeader.IsNull() && !data.HostHeader.IsUnknown() {
		value := data.HostHeader.ValueString()
		actions.SetHostHeader = &value
	}
	if data.Redirect != nil {
		actions.SetRedirect = &client.HTTPProxyRedirect{URL: data.Redirect.URL.ValueString(), StatusCode: data.Redirect.StatusCode.ValueInt64(), AppendOriginalPath: data.Redirect.AppendOriginalPath.ValueBool()}
	}
	if data.RateLimit != nil && data.RateLimit.BySourceIP != nil && data.RateLimit.BySourceIPAndURL != nil {
		actions.SetRateLimit = &client.HTTPProxyTrafficRateLimit{BySourceIP: mapHTTPProxyRateLimitRuleToAPI(data.RateLimit.BySourceIP), BySourceIPAndURL: mapHTTPProxyRateLimitRuleToAPI(data.RateLimit.BySourceIPAndURL)}
	}
	if data.MaxBodySize != nil {
		actions.SetMaxBodySize = &client.HTTPProxySetMaximumBodySize{Enforcement: data.MaxBodySize.Enforcement.ValueString()}
		if actions.SetMaxBodySize.Enforcement == "ENABLED" {
			value := data.MaxBodySize.ValueBytes.ValueInt64()
			actions.SetMaxBodySize.Value = &value
		}
	}
	if data.BotProtection != nil {
		actions.SetBotProtection = &client.HTTPProxyTrafficRuleBotProtection{Strategy: data.BotProtection.Strategy.ValueString()}
	}
	return actions
}

func mapHTTPProxyResponseToModel(tc *client.HTTPProxyResponse, prior HTTPProxyResourceModel) HTTPProxyResourceModel {
	attributes := tc.Data.Attributes
	prior.ID = types.StringValue(tc.Data.ID)
	prior.Name = types.StringValue(attributes.Name)
	if attributes.Deployment != nil {
		prior.DeploymentState = types.StringValue(attributes.Deployment.State)
	}
	if attributes.ConnectionReuseEnabled != nil {
		prior.ConnectionReuse = types.BoolValue(*attributes.ConnectionReuseEnabled)
	}
	if attributes.Frontend != nil {
		prior.Frontend = mapHTTPProxyFrontendFromAPI(attributes.Frontend)
	}
	if attributes.Backend != nil {
		prior.Backend = mapHTTPProxyBackendFromAPI(attributes.Backend)
	}
	if attributes.ProtocolSettings != nil {
		prior.ProtocolSettings = mapHTTPProxyProtocolSettingsFromAPI(attributes.ProtocolSettings)
	}
	if attributes.BotProtection != nil {
		prior.BotProtection = &HTTPProxyBotProtectionModel{Strategy: types.StringValue(attributes.BotProtection.Strategy), ChallengeType: types.StringValue(attributes.BotProtection.ChallengeType)}
	}
	if attributes.DataProtection != nil {
		prior.DataProtection = &HTTPProxyDataProtectionModel{LogRedaction: &HTTPProxyLogRedactionModel{Headers: stringSliceToTypeValues(attributes.DataProtection.LogRedaction.Headers), Cookies: stringSliceToTypeValues(attributes.DataProtection.LogRedaction.Cookies)}}
	}
	if attributes.CustomPages != nil {
		prior.CustomPages = mapHTTPProxyCustomPagesFromAPI(*attributes.CustomPages)
	}
	if attributes.WAF != nil {
		prior.WAF = mapHTTPProxyWAFFromAPI(attributes.WAF)
	}
	if attributes.RateLimiting != nil {
		prior.RateLimiting = mapHTTPProxyRateLimitingFromAPI(attributes.RateLimiting)
	}
	if attributes.IPBasedAccessControl != nil {
		prior.IPBasedAccessControl = mapHTTPProxyIPAccessControlFromAPI(attributes.IPBasedAccessControl)
	}
	if attributes.TrafficRules != nil {
		prior.TrafficRules = mapHTTPProxyTrafficRulesFromAPI(*attributes.TrafficRules, prior.TrafficRules)
	}
	return prior
}

func mapHTTPProxyFrontendFromAPI(data *client.HTTPProxyFrontend) *HTTPProxyFrontendModel {
	result := &HTTPProxyFrontendModel{ConnectionType: types.StringValue(data.ConnectionType), Port: types.Int64Value(data.Port), Hosts: make([]HTTPProxyFrontendHostModel, 0, len(data.Hosts))}
	result.IPv4 = nullableStringValue(data.IPv4)
	result.IPv6 = nullableStringValue(data.IPv6)
	for _, host := range data.Hosts {
		result.Hosts = append(result.Hosts, HTTPProxyFrontendHostModel{Host: types.StringValue(host.Host), CertificateID: nullableStringValue(host.CertificateID), TLSConfig: nullableStringValue(host.TLSConfig)})
	}
	if data.ConnectionType == "SECURE" {
		if data.RedirectHTTP != nil {
			result.RedirectHTTP = types.BoolValue(*data.RedirectHTTP)
		}
		if data.HTTPStrictTransportSecurity != nil {
			result.HSTS = &HTTPProxyHSTSModel{Enabled: types.BoolValue(data.HTTPStrictTransportSecurity.Enabled), MaxAge: types.Int64Value(data.HTTPStrictTransportSecurity.MaxAge), IncludeSubdomains: types.BoolValue(data.HTTPStrictTransportSecurity.IncludeSubdomains), Preload: types.BoolValue(data.HTTPStrictTransportSecurity.Preload)}
		}
		if data.ClientCertificateVerification != nil {
			result.ClientCertificateVerification = &HTTPProxyClientCertificateVerificationModel{Mode: types.StringValue(data.ClientCertificateVerification.Mode), CACertificateIDs: stringSliceToTypeValues(data.ClientCertificateVerification.CACertificateIDs)}
		}
	}
	return result
}

func mapHTTPProxyBackendFromAPI(data *client.HTTPProxyBackend) *HTTPProxyBackendModel {
	result := &HTTPProxyBackendModel{Hosts: make([]HTTPProxyBackendHostModel, 0, len(data.Hosts)), DeliveryMethod: types.StringValue(data.DeliveryMethod), ServerName: nullableStringValue(data.ServerName)}
	for _, host := range data.Hosts {
		result.Hosts = append(result.Hosts, HTTPProxyBackendHostModel{Address: types.StringValue(host.Address), Port: types.Int64Value(host.Port)})
	}
	if data.TLSSettings != nil {
		result.TLSSettings = &HTTPProxyTLSSettingsModel{ClientCertificateID: nullableStringValue(data.TLSSettings.ClientCertificateID)}
		if data.TLSSettings.VerifyCertificate != nil {
			result.TLSSettings.VerifyCertificate = &HTTPProxyVerifyCertificateModel{Mode: types.StringValue(data.TLSSettings.VerifyCertificate.Mode), CACertificateIDs: stringSliceToTypeValues(data.TLSSettings.VerifyCertificate.CACertificateIDs)}
		}
	}
	return result
}

func mapHTTPProxyProtocolSettingsFromAPI(data *client.HTTPProxyProtocolSettings) *HTTPProxyProtocolSettingsModel {
	result := &HTTPProxyProtocolSettingsModel{Version: types.StringValue(data.HTTPVersion), EnableWebsockets: types.BoolValue(false)}
	if data.EnableWebsockets != nil {
		result.EnableWebsockets = types.BoolValue(*data.EnableWebsockets)
	}
	return result
}

func mapHTTPProxyCustomPagesFromAPI(data []client.HTTPProxyCustomPage) []HTTPProxyCustomPageModel {
	result := make([]HTTPProxyCustomPageModel, 0, len(data))
	for _, page := range data {
		result = append(result, HTTPProxyCustomPageModel{ID: types.StringValue(page.ID), Type: types.StringValue(page.Type)})
	}
	return result
}

func mapHTTPProxyRateLimitingFromAPI(data *client.HTTPProxyRateLimiting) *HTTPProxyRateLimitingModel {
	return &HTTPProxyRateLimitingModel{BySourceIP: mapHTTPProxyRateLimitRuleFromAPI(data.BySourceIP), BySourceIPAndURL: mapHTTPProxyRateLimitRuleFromAPI(data.BySourceIPAndURL), Exclusions: stringSliceToTypeValues(data.Exclusions)}
}

func mapHTTPProxyRateLimitRuleFromAPI(data client.HTTPProxyRateLimitRule) *HTTPProxyRateLimitRuleModel {
	return &HTTPProxyRateLimitRuleModel{Enforcement: types.StringValue(data.Enforcement), Rate: &HTTPProxyRateLimitRateModel{Value: types.Int64Value(data.Rate.Value), Unit: types.StringValue(data.Rate.Unit)}, Burst: types.Int64Value(data.Burst)}
}

func mapHTTPProxyIPAccessControlFromAPI(data *client.HTTPProxyIPBasedAccessControl) *HTTPProxyIPAccessControlModel {
	result := defaultHTTPProxyIPAccessControl()
	result.DefaultPolicy = types.StringValue(data.DefaultPolicy)
	for _, rule := range data.Rules.IPRanges {
		result.Rules.IPRanges = append(result.Rules.IPRanges, HTTPProxyIPRangeRuleModel{Policy: types.StringValue(rule.Policy), Address: types.StringValue(rule.Address), Note: nullableStringValue(rule.Note), BypassBotProtection: types.BoolValue(rule.BypassProtection.BotProtection)})
	}
	for _, rule := range data.Rules.IPLists {
		result.Rules.IPLists = append(result.Rules.IPLists, HTTPProxyIPListRuleModel{Policy: types.StringValue(rule.Policy), ID: types.StringValue(rule.ID), BypassBotProtection: types.BoolValue(rule.BypassProtection.BotProtection)})
	}
	for _, rule := range data.Rules.KnownServices {
		result.Rules.KnownServices = append(result.Rules.KnownServices, HTTPProxyKnownServiceRuleModel{Policy: types.StringValue(rule.Policy), ID: types.StringValue(rule.ID), Note: nullableStringValue(rule.Note), BypassBotProtection: types.BoolValue(rule.BypassProtection.BotProtection)})
	}
	for _, rule := range data.Rules.GeoLocations {
		result.Rules.GeoLocations = append(result.Rules.GeoLocations, HTTPProxyGeoLocationRuleModel{Policy: types.StringValue(rule.Policy), Region: types.StringValue(rule.Region), Note: nullableStringValue(rule.Note)})
	}
	for _, rule := range data.Rules.ASNs {
		result.Rules.ASNs = append(result.Rules.ASNs, HTTPProxyASNRuleModel{Policy: types.StringValue(rule.Policy), ASN: types.Int64Value(rule.ASN), Note: nullableStringValue(rule.Note)})
	}
	return result
}

func mapHTTPProxyWAFFromAPI(data *client.HTTPProxyWAF) *HTTPProxyWAFModel {
	coreVersion := data.CoreRuleSet.Version
	if coreVersion == "" {
		coreVersion = data.CoreRuleSetID
	}
	result := &HTTPProxyWAFModel{
		Enforcement: types.StringValue(data.Enforcement), ParanoiaLevel: types.Int64Value(data.ParanoiaLevel), CoreRuleSetVersion: types.StringValue(coreVersion), MatchedDataEnabled: types.BoolValue(data.MatchedDataEnabled),
		SourceExclusions: &HTTPProxySourceExclusionsModel{Enabled: types.BoolValue(data.SourceExclusions.Enabled), Sources: stringSliceToTypeValues(data.SourceExclusions.Sources)},
		HTTPCompliance:   &HTTPProxyHTTPComplianceModel{ParameterLimit: &HTTPProxyParameterLimitModel{Enabled: types.BoolValue(data.HTTPCompliance.GlobalConfig.ParameterLimit.Enabled), Limit: types.Int64Value(data.HTTPCompliance.GlobalConfig.ParameterLimit.Limit)}, AllowedMethods: stringSliceToTypeValues(data.HTTPCompliance.GlobalConfig.AllowedHTTPMethods), AllowedVersions: stringSliceToTypeValues(data.HTTPCompliance.GlobalConfig.AllowedHTTPVersions), ResourceConfigs: []HTTPProxyHTTPComplianceResourceConfigModel{}},
		PathExclusions:   mapHTTPProxyPathExclusionsFromAPI(data.PathExclusions), CookieExclusions: mapHTTPProxyCookieExclusionsFromAPI(data.CookieExclusions), StagedWAF: mapHTTPProxyStagedWAFFromAPI(data.StagedWAF),
	}
	for _, config := range data.HTTPCompliance.ResourceConfigs {
		matches := make([]types.String, 0, len(config.Matches))
		for _, match := range config.Matches {
			matches = append(matches, types.StringValue(match.Path))
		}
		result.HTTPCompliance.ResourceConfigs = append(result.HTTPCompliance.ResourceConfigs, HTTPProxyHTTPComplianceResourceConfigModel{Matches: matches, ParseJSONEnabled: types.BoolValue(config.Config.ParseJSONEnabled), ParseXMLEnabled: types.BoolValue(config.Config.ParseXMLEnabled), ParseMultipartRequestEnabled: types.BoolValue(config.Config.ParseMultipartRequestEnabled)})
	}
	return result
}

func mapHTTPProxyPathExclusionsFromAPI(data []client.HTTPProxyPathExclusion) []HTTPProxyWAFPathExclusionModel {
	result := make([]HTTPProxyWAFPathExclusionModel, 0, len(data))
	for _, exclusion := range data {
		ruleIDs := make([]types.String, 0, len(exclusion.Rules))
		for _, rule := range exclusion.Rules {
			if rule.Type == "rule" {
				ruleIDs = append(ruleIDs, types.StringValue(rule.ID))
			}
		}
		result = append(result, HTTPProxyWAFPathExclusionModel{Match: types.StringValue(exclusion.Match), DisableAll: types.BoolValue(exclusion.DisableAll), RuleIDs: ruleIDs})
	}
	return result
}

func mapHTTPProxyCookieExclusionsFromAPI(data []client.HTTPProxyCookieExclusion) []HTTPProxyWAFCookieExclusionModel {
	result := make([]HTTPProxyWAFCookieExclusionModel, 0, len(data))
	for _, exclusion := range data {
		ruleIDs := make([]types.String, 0, len(exclusion.ExcludedRules))
		for _, rule := range exclusion.ExcludedRules {
			if rule.Type == "rule" {
				ruleIDs = append(ruleIDs, types.StringValue(rule.ID))
			}
		}
		result = append(result, HTTPProxyWAFCookieExclusionModel{CookieName: types.StringValue(exclusion.CookieName), ExcludeAllRules: types.BoolValue(exclusion.ExcludeAllRules), RuleIDs: ruleIDs})
	}
	return result
}

func mapHTTPProxyStagedWAFFromAPI(data *client.HTTPProxyStagedWAF) *HTTPProxyStagedWAFModel {
	if data == nil {
		return nil
	}
	result := &HTTPProxyStagedWAFModel{State: types.StringValue(data.State), Mode: types.StringValue(data.Mode), PathExclusions: mapHTTPProxyPathExclusionsFromAPI(data.PathExclusions), CookieExclusions: mapHTTPProxyCookieExclusionsFromAPI(data.CookieExclusions)}
	if data.ParanoiaLevel != nil {
		result.ParanoiaLevel = types.Int64Value(*data.ParanoiaLevel)
	}
	if data.CoreRuleSet != nil {
		result.CoreRuleSetVersion = types.StringValue(data.CoreRuleSet.Version)
	}
	return result
}

func mapHTTPProxyTrafficRulesFromAPI(data []client.HTTPProxyTrafficRule, prior []HTTPProxyTrafficRuleModel) []HTTPProxyTrafficRuleModel {
	result := make([]HTTPProxyTrafficRuleModel, 0, len(data))
	for i, rule := range data {
		var priorRule *HTTPProxyTrafficRuleModel
		if i < len(prior) {
			priorRule = &prior[i]
		}
		model := HTTPProxyTrafficRuleModel{Name: types.StringValue(rule.Name), MatchingConditions: make([]HTTPProxyTrafficMatchingConditionModel, 0, len(rule.MatchingConditions)), Actions: mapHTTPProxyTrafficRuleActionsFromAPI(rule.Actions, priorRuleActions(priorRule))}
		for j, condition := range rule.MatchingConditions {
			var priorValues []types.String
			if priorRule != nil && j < len(priorRule.MatchingConditions) && priorRule.MatchingConditions[j].Hosts != nil {
				priorValues = priorRule.MatchingConditions[j].Hosts.Values
			}
			model.MatchingConditions = append(model.MatchingConditions, HTTPProxyTrafficMatchingConditionModel{Paths: stringSliceToTypeValues(condition.Paths), Hosts: &HTTPProxyTrafficRuleHostMatchModel{Type: types.StringValue(condition.Hosts.Type), Values: mapHTTPProxyOptionalStringListFromAPI(condition.Hosts.Values, priorValues)}})
		}
		result = append(result, model)
	}
	return result
}

func mapHTTPProxyTrafficRuleActionsFromAPI(data client.HTTPProxyTrafficRuleActions, prior *HTTPProxyTrafficRuleActionsModel) *HTTPProxyTrafficRuleActionsModel {
	result := &HTTPProxyTrafficRuleActionsModel{}
	if prior != nil && prior.Backends != nil {
		result.Backends = []HTTPProxyBackendHostModel{}
	}
	if prior != nil && prior.Headers != nil {
		result.Headers = []HTTPProxyHeaderModel{}
	}
	for _, backend := range data.SetBackends {
		result.Backends = append(result.Backends, HTTPProxyBackendHostModel{Address: types.StringValue(backend.Address), Port: types.Int64Value(backend.Port)})
	}
	for _, header := range data.SetHeaders {
		result.Headers = append(result.Headers, HTTPProxyHeaderModel{Key: types.StringValue(header.Key), Value: types.StringValue(header.Value)})
	}
	if data.SetHostHeader != nil {
		result.HostHeader = types.StringValue(*data.SetHostHeader)
	}
	if data.SetRedirect != nil {
		result.Redirect = &HTTPProxyRedirectModel{URL: types.StringValue(data.SetRedirect.URL), StatusCode: types.Int64Value(data.SetRedirect.StatusCode), AppendOriginalPath: types.BoolValue(data.SetRedirect.AppendOriginalPath)}
	}
	if data.SetRateLimit != nil {
		result.RateLimit = &HTTPProxyTrafficRateLimitModel{BySourceIP: mapHTTPProxyRateLimitRuleFromAPI(data.SetRateLimit.BySourceIP), BySourceIPAndURL: mapHTTPProxyRateLimitRuleFromAPI(data.SetRateLimit.BySourceIPAndURL)}
	}
	if data.SetMaxBodySize != nil {
		result.MaxBodySize = &HTTPProxyMaximumBodySizeModel{Enforcement: types.StringValue(data.SetMaxBodySize.Enforcement)}
		if data.SetMaxBodySize.Value != nil {
			result.MaxBodySize.ValueBytes = types.Int64Value(*data.SetMaxBodySize.Value)
		}
	}
	if data.SetBotProtection != nil {
		result.BotProtection = &HTTPProxyTrafficRuleBotProtectionModel{Strategy: types.StringValue(data.SetBotProtection.Strategy)}
	}
	return result
}

func priorRuleActions(prior *HTTPProxyTrafficRuleModel) *HTTPProxyTrafficRuleActionsModel {
	if prior == nil {
		return nil
	}
	return prior.Actions
}

func mapHTTPProxyOptionalStringListFromAPI(data []string, prior []types.String) []types.String {
	if len(data) > 0 {
		return stringSliceToTypeValues(data)
	}
	if prior != nil {
		return []types.String{}
	}
	return nil
}

func boolValuePointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

func httpProxyStringValues(values []types.String) []string {
	if values == nil {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ValueString())
	}
	return result
}

func nullableStringValue(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
