package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mabel-dev/terraform-provider-opteryx/internal/client"
)

var _ provider.Provider = &OpteryxProvider{}

// OpteryxProvider talks to policy.opteryx's access-policy CRUD API
// (app/routes/v1/access.py).
type OpteryxProvider struct {
	// version is set to the release tag by goreleaser; "dev" for local builds.
	version string
}

type OpteryxProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *OpteryxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "opteryx"
	resp.Version = p.version
}

func (p *OpteryxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages policy.opteryx access policies.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the policy.opteryx service. Defaults to the OPTERYX_ENDPOINT environment variable, falling back to https://policy.opteryx.app.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer JWT minted by authenticate.opteryx. Defaults to the OPTERYX_TOKEN environment variable. The identity behind this token cannot create, modify, or delete a policy that grants itself access -- policy.opteryx rejects self-grants at every write endpoint -- so this token must belong to an owner/admin identity distinct from any principal it manages.",
			},
		},
	}
}

func (p *OpteryxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data OpteryxProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("OPTERYX_ENDPOINT")
	if !data.Endpoint.IsNull() && data.Endpoint.ValueString() != "" {
		endpoint = data.Endpoint.ValueString()
	}
	if endpoint == "" {
		endpoint = "https://policy.opteryx.app"
	}

	token := os.Getenv("OPTERYX_TOKEN")
	if !data.Token.IsNull() && data.Token.ValueString() != "" {
		token = data.Token.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing API token",
			"A bearer token is required to call policy.opteryx. Set the provider's token attribute or the OPTERYX_TOKEN environment variable.",
		)
		return
	}

	c := client.New(endpoint, token)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OpteryxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAccessPolicyResource,
	}
}

func (p *OpteryxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccessPoliciesDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OpteryxProvider{
			version: version,
		}
	}
}
