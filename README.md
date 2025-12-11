# pc-terraform-providers

This repo holds our custom Terraform providers (hopefully only one).

## Testing manually

If you want to test manually in a real world terraform example, you can simply tell Terraform to use your local installation of the provider instead of the one downloaded from the registry.

To do so simply run :

```
go install ./... 
```

Of course the GOBIN path should be in your $PATH for this to work. The default location is `/Users/<user>/go/bin` on mac.

Then add this configuration to your `.terraformrc` in your home directory

```
provider_installation {
  dev_overrides {
    "registry.terraform.io/pass-culture/pc" = "/Users/louis/go/bin"
  }
  direct {}
}
```


## Running tests

Since the scope of this repo should stay very limited, tests are not included on CI and must be run manually.

> :warning: **Testing Terraform provider will create real ressources**: Be very mindful of the costs and to not disrupt critical infrastructure!

```bash
TF_TEST_GCP_PROJECT_ID="GCP_PROJECT_TO_RUN_TEST_IN" TF_ACC=1 go test -v ./...
```
