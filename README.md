# pc-terraform-providers

This repo holds our custom Terraform providers (hopefully only one).

## Running tests

Since the scope of this repo should stay very limited, tests are not included on CI and must be run manually.

> :warning: **Testing Terraform provider will create real ressources**: Be very mindful of the costs and to not disrupt critical infrastructure!

```bash
TF_TEST_GCP_PROJECT_ID="GCP_PROJECT_TO_RUN_TEST_IN" TF_ACC=1 go test -v ./...
```
