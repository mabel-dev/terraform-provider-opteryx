package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccAccessPolicyResource exercises create, import, and in-place update
// (role change) against a real policy.opteryx instance. Skipped unless TF_ACC
// is set (resource.Test enforces this itself). Requires OPTERYX_TOKEN,
// OPTERYX_TEST_WORKSPACE, and OPTERYX_TEST_PRINCIPAL -- see testAccPreCheck.
// OPTERYX_TEST_PRINCIPAL must be a real, distinct identity: the token's own
// identity can't be granted a policy (policy.opteryx rejects self-grants).
func TestAccAccessPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccessPolicyResourceConfig("reader", "test.acceptance.*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("opteryx_access_policy.test", "role", "reader"),
					resource.TestCheckResourceAttr("opteryx_access_policy.test", "pattern", "test.acceptance.*"),
					resource.TestCheckResourceAttrSet("opteryx_access_policy.test", "id"),
					resource.TestCheckResourceAttrSet("opteryx_access_policy.test", "created_at"),
				),
			},
			{
				ResourceName:      "opteryx_access_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccAccessPolicyImportStateIDFunc("opteryx_access_policy.test"),
			},
			{
				// role is updatable in place -- workspace/principal are unchanged,
				// so this must not trigger a replace.
				Config: testAccAccessPolicyResourceConfig("writer", "test.acceptance.*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("opteryx_access_policy.test", "role", "writer"),
				),
			},
		},
	})
}

func testAccAccessPolicyResourceConfig(role, pattern string) string {
	return fmt.Sprintf(`
resource "opteryx_access_policy" "test" {
  workspace = %[1]q
  principal = %[2]q
  role      = %[3]q
  pattern   = %[4]q
}
`, os.Getenv("OPTERYX_TEST_WORKSPACE"), os.Getenv("OPTERYX_TEST_PRINCIPAL"), role, pattern)
}

// testAccAccessPolicyImportStateIDFunc builds the "workspace/policy_id" import
// ID from the resource's actual state, since the ID Terraform generated is
// only the policy_id half.
func testAccAccessPolicyImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["workspace"], rs.Primary.ID), nil
	}
}
