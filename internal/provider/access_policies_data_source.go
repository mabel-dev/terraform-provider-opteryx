package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mabel-dev/terraform-provider-opteryx/internal/client"
)

var _ datasource.DataSource = &AccessPoliciesDataSource{}

func NewAccessPoliciesDataSource() datasource.DataSource {
	return &AccessPoliciesDataSource{}
}

type AccessPoliciesDataSource struct {
	client *client.Client
}

type AccessPoliciesDataSourceModel struct {
	Workspace types.String                `tfsdk:"workspace"`
	Policies  []AccessPolicyListItemModel `tfsdk:"policies"`
}

type AccessPolicyListItemModel struct {
	ID        types.String `tfsdk:"id"`
	Principal types.String `tfsdk:"principal"`
	Role      types.String `tfsdk:"role"`
	Pattern   types.String `tfsdk:"pattern"`
}

func (d *AccessPoliciesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_policies"
}

func (d *AccessPoliciesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the access policies currently granted on a policy.opteryx workspace. Use this to audit existing grants or to look up a policy's id for `terraform import`.",
		Attributes: map[string]schema.Attribute{
			"workspace": schema.StringAttribute{
				Required:    true,
				Description: "Workspace to list policies for.",
			},
			"policies": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Every policy document under the workspace's $policies/access collection.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Policy document ID -- pair with `workspace` as the `terraform import` ID.",
						},
						"principal": schema.StringAttribute{
							Computed:    true,
							Description: "Identity the policy grants access to, or \"*\" for any authenticated user.",
						},
						"role": schema.StringAttribute{
							Computed:    true,
							Description: "Role granted.",
						},
						"pattern": schema.StringAttribute{
							Computed:    true,
							Description: "Resource pattern the role applies to.",
						},
					},
				},
			},
		},
	}
}

func (d *AccessPoliciesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider maintainers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *AccessPoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccessPoliciesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, err := d.client.ListPolicies(ctx, data.Workspace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing access policies", err.Error())
		return
	}

	data.Policies = make([]AccessPolicyListItemModel, 0, len(policies))
	for _, p := range policies {
		data.Policies = append(data.Policies, AccessPolicyListItemModel{
			ID:        types.StringValue(p.Policy),
			Principal: types.StringValue(p.Identity),
			Role:      types.StringValue(p.Role),
			Pattern:   types.StringValue(p.Pattern),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
