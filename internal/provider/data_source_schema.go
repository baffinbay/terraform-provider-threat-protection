package provider

import "github.com/hashicorp/terraform-plugin-framework/datasource/schema"

func collectionDataSourceSchema(description, collectionAttribute, labelAttribute string) schema.Schema {
	itemAttributes := map[string]schema.Attribute{
		"id":   schema.StringAttribute{Computed: true},
		"type": schema.StringAttribute{Computed: true},
		labelAttribute: schema.StringAttribute{
			Computed: true,
		},
	}
	return schema.Schema{
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			collectionAttribute: schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: itemAttributes,
				},
			},
		},
	}
}
