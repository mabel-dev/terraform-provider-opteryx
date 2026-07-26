package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mabel-dev/terraform-provider-opteryx/internal/client"
)

var _ resource.Resource = &AccessPolicyResource{}
var _ resource.ResourceWithImportState = &AccessPolicyResource{}

func NewAccessPolicyResource() resource.Resource {
	return &AccessPolicyResource{}
}

type AccessPolicyResource struct {
	client *client.Client
}

// AccessPolicyResourceModel maps the opteryx_access_policy schema to Go.
type AccessPolicyResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Workspace types.String `tfsdk:"workspace"`
	Principal types.String `tfsdk:"principal"`
	Role      types.String `tfsdk:"role"`
	Pattern   types.String `tfsdk:"pattern"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	UpdatedBy types.String `tfsdk:"updated_by"`
}

func (r *AccessPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_policy"
}

func (r *AccessPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A single access-policy grant on a policy.opteryx workspace: one principal, one role, on resources matching one pattern.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The policy document ID assigned by policy.opteryx.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace": schema.StringAttribute{
				Required:    true,
				Description: "Workspace the policy applies to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal": schema.StringAttribute{
				Required:    true,
				Description: "Identity this policy grants access to (user, email, subject, or \"*\" for any authenticated user). policy.opteryx has no endpoint to reassign a policy's principal, so changing this replaces the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required:    true,
				Description: "Role granted: one of owner, admin, writer, reader.",
				Validators: []validator.String{
					stringvalidator.OneOf("owner", "admin", "writer", "reader"),
				},
			},
			"pattern": schema.StringAttribute{
				Required:    true,
				Description: "Resource pattern the role applies to, e.g. \"analytics.sales.*\". A wildcard principal (\"*\") must pair with an exact, non-glob pattern -- policy.opteryx rejects a policy wildcarded on both sides.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the policy was created (ISO 8601, UTC).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the policy was last updated (ISO 8601, UTC).",
			},
			"updated_by": schema.StringAttribute{
				Computed:    true,
				Description: "Identity that last updated the policy.",
			},
		},
	}
}

func (r *AccessPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider maintainers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *AccessPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID, err := r.client.CreatePolicy(ctx, plan.Workspace.ValueString(), plan.Principal.ValueString(), plan.Role.ValueString(), plan.Pattern.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating access policy", err.Error())
		return
	}
	plan.ID = types.StringValue(policyID)

	if found := r.readInto(ctx, &resp.Diagnostics, plan.Workspace.ValueString(), policyID, &plan); !found && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Error reading access policy after create", "policy.opteryx returned success but the policy could not be found immediately afterward")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found := r.readInto(ctx, &resp.Diagnostics, state.Workspace.ValueString(), state.ID.ValueString(), &state)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccessPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// workspace and principal are RequiresReplace, so only role and pattern can
	// reach here -- which matches UpdatePolicyRequest, the only two fields the
	// API itself accepts on PUT.
	if err := r.client.UpdatePolicy(ctx, plan.Workspace.ValueString(), plan.ID.ValueString(), plan.Role.ValueString(), plan.Pattern.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating access policy", err.Error())
		return
	}

	if found := r.readInto(ctx, &resp.Diagnostics, plan.Workspace.ValueString(), plan.ID.ValueString(), &plan); !found && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Error reading access policy after update", "policy.opteryx returned success but the policy could not be found immediately afterward")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePolicy(ctx, state.Workspace.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting access policy", err.Error())
		return
	}
}

// ImportState accepts "workspace/policy_id", since a policy ID alone doesn't
// say which workspace's Firestore subcollection to look in.
func (r *AccessPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier of the form \"workspace/policy_id\", got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// readInto fetches the current policy and fills model's principal/role/pattern
// and computed fields, returning false if the policy no longer exists (so Read
// can drop it from state; Create/Update instead treat a missing policy right
// after a successful write as an error via diags).
func (r *AccessPolicyResource) readInto(ctx context.Context, diags *diag.Diagnostics, workspace, policyID string, model *AccessPolicyResourceModel) bool {
	detail, err := r.client.GetPolicy(ctx, workspace, policyID)
	if err != nil {
		if client.IsNotFound(err) {
			return false
		}
		diags.AddError("Error reading access policy", err.Error())
		return false
	}

	model.Principal = types.StringValue(detail.Principal.Identity)
	model.Role = types.StringValue(detail.Role)
	model.Pattern = types.StringValue(detail.Pattern)
	model.CreatedAt = stringOrEmpty(detail.CreatedAt)
	model.UpdatedAt = stringOrEmpty(detail.UpdatedAt)
	if detail.UpdatedBy != nil {
		model.UpdatedBy = types.StringValue(detail.UpdatedBy.Identity)
	} else {
		model.UpdatedBy = types.StringValue("")
	}
	return true
}

func stringOrEmpty(s *string) types.String {
	if s == nil {
		return types.StringValue("")
	}
	return types.StringValue(*s)
}
