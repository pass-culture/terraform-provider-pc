package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccFirestoreDocument_basic(t *testing.T) {
	projectID, projectIDSet := os.LookupEnv("TF_TEST_GCP_PROJECT_ID")
	if !projectIDSet {
		t.Fatalf("TF_TEST_GCP_PROJECT_ID not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"passculture": providerserver.NewProtocol6WithError(New()()),
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {VersionConstraint: "6.16.0", Source: "google"},
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "google_firestore_document" "test" {
  project     = %q
  database    = "test-firestore-infra" 
  collection  = "provider"
  document_id = "basicTest"
  fields      = "{\"key1\":{\"stringValue\":\"avalue\"}, \"key2\":{\"integerValue\":\"5\"}, \"key3\":{\"booleanValue\":true}}"
}

data "passculture_firestore_document" "test" {
  project = %q
  database = google_firestore_document.test.database
  collection = google_firestore_document.test.collection
  document_id = google_firestore_document.test.document_id
}`, projectID, projectID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.passculture_firestore_document.test",
						tfjsonpath.New("fields").AtMapKey("key1"),
						knownvalue.StringExact("avalue"),
					),
					statecheck.ExpectKnownValue(
						"data.passculture_firestore_document.test",
						tfjsonpath.New("fields").AtMapKey("key2"),
						knownvalue.Int64Exact(5),
					),
					statecheck.ExpectKnownValue(
						"data.passculture_firestore_document.test",
						tfjsonpath.New("fields").AtMapKey("key3"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}
