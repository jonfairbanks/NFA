# Running Morpheus in Azure


### Setup

To start, get the subscription ID for your Azure account:
```
az account show --query id --output tsv
```

Then allow authentication with Azure, we can create a service principal:
```
az ad sp create-for-rbac --name "terraform-sp" --role="Contributor" --scopes="/subscriptions/<AZURE_SUBSCRIPTION_ID>"
```

Finally you can enable Terraform to authenticate with Azure. The following environment variables should be set locally or within a GitHub Action:
```
export ARM_CLIENT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export ARM_CLIENT_SECRET="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export ARM_SUBSCRIPTION_ID="<SUBSCRIPTION_ID>"
export ARM_TENANT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

Before running the Terraform deployment, you may also want to verify the environment variables defined in `variables.tf`. To override any default values, simply add them to the associated `.tfvars` file for the environment in question.

### Deployment

With the environment variables set, you can proceed with deploying the Morpheus stack into Azure using Terraform. 

You can either initiate Terraform locally or within a GitHub workflow. Be sure to pass the appropriate `.tfvars` file based on the environment you are deploying. 

Local:
```
terraform init
terraform plan -var-file=development.tfvars
terraform apply -var-file=development.tfvars
```

A [sample GitHub workflow](morpheus-azure-deployment.yaml) to deploy this Terraform stack can be found in the repo. 