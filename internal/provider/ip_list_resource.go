package provider

import (
	"context"
	"fmt"
	"net/netip"
	"regexp"
	"strings"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &IPListResource{}
var _ resource.ResourceWithImportState = &IPListResource{}
var _ resource.ResourceWithValidateConfig = &IPListResource{}

var ipListNotePattern = regexp.MustCompile("^[!#$%&'*+.^_` |~0-9a-zA-Z-]+$")

func NewIPListResource() resource.Resource {
	return &IPListResource{}
}

type IPListResource struct {
	configuredResource
}

type IPListResourceModel struct {
	ID      types.String       `tfsdk:"id"`
	Name    types.String       `tfsdk:"name"`
	Entries []IPListEntryModel `tfsdk:"entries"`
}

type IPListEntryModel struct {
	Value types.String `tfsdk:"value"`
	Note  types.String `tfsdk:"note"`
}

func (r *IPListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_list"
}

func (r *IPListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Baffin Bay IP List.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the IP list.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the IP list.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"entries": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "The IP or network prefixes in the list.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(500),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "A canonical IPv4 or IPv6 prefix in CIDR notation.",
							Validators: []validator.String{
								ipListCIDRValidator{},
							},
						},
						"note": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "An optional note describing the entry.",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 500),
								stringvalidator.RegexMatches(ipListNotePattern, "must contain only characters supported by the IP list API"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *IPListResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data IPListResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() && strings.TrimSpace(data.Name.ValueString()) != data.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid IP List Name", "IP list name must not contain leading or trailing whitespace.")
	}

	seen := make(map[string]int, len(data.Entries))
	for i, entry := range data.Entries {
		if entry.Value.IsNull() || entry.Value.IsUnknown() {
			continue
		}

		value := entry.Value.ValueString()
		if firstIndex, exists := seen[value]; exists {
			resp.Diagnostics.AddAttributeError(
				path.Root("entries").AtListIndex(i).AtName("value"),
				"Duplicate IP List Entry",
				fmt.Sprintf("IP list entry %q duplicates entries[%d].value; each prefix must be unique.", value, firstIndex),
			)
			continue
		}
		seen[value] = i
	}
}

func (r *IPListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IPListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipList, err := r.client.CreateIPList(ctx, data.Name.ValueString(), ipListEntriesFromModel(data.Entries))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create IP list, got error: %s", err))
		return
	}

	data.ID = types.StringValue(ipList.Data.ID)
	data, err = r.readIPList(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IP list after create, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IPListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.readIPList(ctx, data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IP list, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IPListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateIPList(ctx, data.ID.ValueString(), data.Name.ValueString(), ipListEntriesFromModel(data.Entries))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update IP list, got error: %s", err))
		return
	}

	data, err = r.readIPList(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IP list after update, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IPListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IPListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteIPList(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete IP list, got error: %s", err))
	}
}

func (r *IPListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *IPListResource) readIPList(ctx context.Context, prior IPListResourceModel) (IPListResourceModel, error) {
	ipList, err := r.client.GetIPList(ctx, prior.ID.ValueString())
	if err != nil {
		return prior, err
	}
	if ipList.Data.Type != client.IPListType {
		return prior, fmt.Errorf("IP list %q has type %q, expected %q", ipList.Data.ID, ipList.Data.Type, client.IPListType)
	}

	data := prior
	data.ID = types.StringValue(ipList.Data.ID)
	data.Name = types.StringValue(ipList.Data.Attributes.Name)
	data.Entries = ipListEntriesToModel(ipList.Data.Attributes.Entries)
	return data, nil
}

func ipListEntriesFromModel(entries []IPListEntryModel) []client.IPListEntry {
	result := make([]client.IPListEntry, 0, len(entries))
	for _, entry := range entries {
		var note *string
		if !entry.Note.IsNull() && !entry.Note.IsUnknown() {
			value := entry.Note.ValueString()
			note = &value
		}
		result = append(result, client.IPListEntry{Value: entry.Value.ValueString(), Note: note})
	}
	return result
}

func ipListEntriesToModel(entries []client.IPListEntry) []IPListEntryModel {
	result := make([]IPListEntryModel, 0, len(entries))
	for _, entry := range entries {
		note := types.StringNull()
		if entry.Note != nil && *entry.Note != "" {
			note = types.StringValue(*entry.Note)
		}
		result = append(result, IPListEntryModel{Value: types.StringValue(entry.Value), Note: note})
	}
	return result
}

type ipListCIDRValidator struct{}

func (ipListCIDRValidator) Description(context.Context) string {
	return "value must be a canonical IPv4 or IPv6 prefix in CIDR notation"
}

func (ipListCIDRValidator) MarkdownDescription(ctx context.Context) string {
	return ipListCIDRValidator{}.Description(ctx)
}

func (ipListCIDRValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid IP List Entry", fmt.Sprintf("%q must be a valid IPv4 or IPv6 prefix in CIDR notation.", value))
		return
	}

	canonical := prefix.Masked().String()
	if value != canonical {
		resp.Diagnostics.AddAttributeError(req.Path, "Non-canonical IP List Entry", fmt.Sprintf("%q is not canonical; use %q instead.", value, canonical))
	}
}
