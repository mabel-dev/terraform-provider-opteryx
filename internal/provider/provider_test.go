package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories instantiates the provider under test for
// each Terraform CLI command an acceptance test runs.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"opteryx": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates the environment required to run acceptance tests.
// These hit a real policy.opteryx instance and create/destroy real policies,
// so they're gated behind TF_ACC and require credentials for an identity that
// is NOT itself a principal in any policy the tests create -- see the
// self-grant rule documented in the README.
func testAccPreCheck(t *testing.T) {
	if os.Getenv("OPTERYX_TOKEN") == "" {
		t.Fatal("OPTERYX_TOKEN must be set to run acceptance tests, with a token for an owner/admin identity distinct from any test principal")
	}
	if os.Getenv("OPTERYX_TEST_WORKSPACE") == "" {
		t.Fatal("OPTERYX_TEST_WORKSPACE must be set to a workspace the test identity has owner/admin authority over")
	}
	if os.Getenv("OPTERYX_TEST_PRINCIPAL") == "" {
		t.Fatal("OPTERYX_TEST_PRINCIPAL must be set to an identity distinct from OPTERYX_TOKEN's own -- policy.opteryx rejects granting a policy to yourself")
	}
}
