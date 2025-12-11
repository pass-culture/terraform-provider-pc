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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			"optional": schema.BoolAttribute{
				MarkdownDescription: `If set to true and the corresponding document on firestore does not exist.
The data source will not error and return an empty value instead`,
				Optional: true,
			},
			"fields": schema.DynamicAttribute{
				MarkdownDescription: "Document fields (supports nested maps and lists)",
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
	Optional   types.Bool    `tfsdk:"optional"`
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
		resp.Diagnostics.AddError("Could not init Firestore SDK", err.Error())
		return
	}

	defer func() {
		if err := client.Close(); err != nil {
			tflog.Warn(ctx, "could not firestore close client")
		}
	}()

	documentPath := fmt.Sprintf("%s/%s", data.Collection.ValueString(), data.DocumentID.ValueString())
	tflog.Info(ctx, "Reading Firestore document", map[string]any{"path": documentPath})

	firestoreDoc := client.Doc(documentPath)
	if firestoreDoc == nil {
		resp.Diagnostics.AddError("Invalid document path", "Path must be a valid Firestore document reference")
		return
	}

	doc, err := firestoreDoc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			if !data.Optional.ValueBool() {
				resp.Diagnostics.AddError("Could not find document", err.Error())
			} else {
				data.Fields = types.DynamicNull()
				diags = resp.State.Set(ctx, &data)
				resp.Diagnostics.Append(diags...)
				return
			}
		}
		resp.Diagnostics.AddError("Unexpected error", err.Error())
		return
	}

	val, err := convertToDynamic(ctx, doc.Data())
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert Firestore data", err.Error())
		return
	}

	data.Fields = types.DynamicValue(val)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// convertToDynamic recursively converts Firestore structures into Terraform attr.Values
func convertToDynamic(ctx context.Context, value any) (attr.Value, error) {
	switch v := value.(type) {
	case bool:
		return types.BoolValue(v), nil
	case string:
		return types.StringValue(v), nil
	case int64:
		return types.Int64Value(v), nil
	case float64:
		return types.Float64Value(v), nil
	case nil:
		return types.StringNull(), nil
	case map[string]any:
		fields := make(map[string]attr.Value)
		fieldTypes := make(map[string]attr.Type)
		for k, nested := range v {
			nestedVal, err := convertToDynamic(ctx, nested)
			if err != nil {
				return nil, err
			}
			fields[k] = nestedVal
			fieldTypes[k] = nestedVal.Type(ctx)
		}
		return types.ObjectValueMust(fieldTypes, fields), nil
	case []any:
		if len(v) == 0 {
			return types.ListNull(types.StringType), nil
		}
		elements := make([]attr.Value, len(v))
		var elemType attr.Type = types.StringType
		for i, elem := range v {
			converted, err := convertToDynamic(ctx, elem)
			if err != nil {
				return nil, err
			}
			elements[i] = converted
			elemType = converted.Type(ctx)
		}
		return types.ListValueMust(elemType, elements), nil
	default:
		return nil, fmt.Errorf("unsupported Firestore type: %T", v)
	}
}
