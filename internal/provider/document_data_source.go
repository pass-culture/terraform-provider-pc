package provider

import (
	"context"
	"fmt"

	firestore "cloud.google.com/go/firestore"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = (*documentDataSource)(nil)

type documentDataSource struct {
	provider passcultureProvider
}

func NewFirestoreDocumentDataSource() datasource.DataSource {
	return &documentDataSource{}
}

func (e *documentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firestore_document"
}

func (e *documentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				MarkdownDescription: "Project ID",
				Required:            true,
			},
			"database": schema.StringAttribute{
				MarkdownDescription: "Firestore database",
				Required:            true,
			},
			"collection": schema.StringAttribute{
				MarkdownDescription: "Firestore collection",
				Required:            true,
			},
			"document_id": schema.StringAttribute{
				MarkdownDescription: "Document ID",
				Required:            true,
			},
			"fields": schema.DynamicAttribute{
				MarkdownDescription: "Document fields",
				Computed:            true,
			},
		},
	}
}

type documentDataSourceData struct {
	Project    types.String  `tfsdk:"project"`
	Database   types.String  `tfsdk:"database"`
	Collection types.String  `tfsdk:"collection"`
	DocumentID types.String  `tfsdk:"document_id"`
	Fields     types.Dynamic `tfsdk:"fields"`
}

func (e *documentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data documentDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := firestore.NewClientWithDatabase(ctx, data.Project.ValueString(), data.Database.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not init firestore SDK",
			err.Error(),
		)
		return
	}
	defer client.Close()
	documentPath := fmt.Sprintf("%s/%s", data.Collection.ValueString(), data.DocumentID.ValueString())
	tflog.Info(ctx, documentPath)
	firestoreDoc := client.Doc(documentPath)
	if firestoreDoc == nil {
		resp.Diagnostics.AddError(
			"Invalid document path",
			"Path must be a list of string separated by slashes of even length. It must not start with a /",
		)
		return
	}
	doc, err := firestoreDoc.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not find document",
			err.Error(),
		)
		return
	}
	newFields := make(map[string]attr.Value, len(doc.Data()))
	newFieldsTypes := make(map[string]attr.Type, len(doc.Data()))
	for key, untypedValue := range doc.Data() {
		switch value := untypedValue.(type) {
		case bool:
			newFields[key] = types.DynamicValue(types.BoolValue(value))
			newFieldsTypes[key] = newFields[key].Type(ctx)
		case string:
			newFields[key] = types.DynamicValue(types.StringValue(value))
			newFieldsTypes[key] = newFields[key].Type(ctx)
		case int64:
			newFields[key] = types.DynamicValue(types.Int64Value(value))
			newFieldsTypes[key] = newFields[key].Type(ctx)
		case nil:
			newFields[key] = types.StringNull()
			newFieldsTypes[key] = newFields[key].Type(ctx)
		default:
			resp.Diagnostics.AddError(
				"Invalid data type in firestore",
				fmt.Sprintf(`Only Boolean, String, Integer and null values are readable via this provider`),
			)
		}
	}
	mapValue, diags := types.ObjectValue(newFieldsTypes, newFields)
	resp.Diagnostics.Append(diags...)
	data.Fields = types.DynamicValue(mapValue)
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
