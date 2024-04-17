# Import Variable plugin

This plugin will fetch output values from another environment and insert them as terraform and/or environment variables.

Similar to self hosted agent secrets, use this notation in the value of the terraform input value:

* `${env0:<environment id>:<output name>}`
* `${env0:<environment name>:<output name>}` (see note below about Environment Names restrictions)

For fetching JSON output values - make sure you select JSON type for your input variable, and in the value use the following JSON schema.
* `{"ENV0_ENVIRONMENT_NAME":<environment name>, "output": <output name>}`


Terraform Provider Example:

```
resource "env0_configuration_variable" "ami-id" {
  name        = "ami-id"
  value       = $${env0-workflow:aws-base-ami:ami-id}  \\ use an extra `$` to escape $. see https://developer.hashicorp.com/terraform/language/expressions/strings
}
```

## Workflows

When using the plugin within a workflow you can use the following notation:

string types: `${env0-workflow:<parent name>:<output name>}`
json types: `{"ENV0_WORKFLOW_PARENT":<parent name>, "output": <output name>}`

In this case, the parent name is the yaml parent, not the "environment name." 
For example, given the following `env0.workflow.yaml` the variable structure to fetch the "vpc-id" from the "parent vpc" would be `${env0-workflow:vpc:vpc-id}`
Similarly, to fetch the tags (in json) from the vpc: `{"ENV0_WORKFLOW_PARENT":vpc, "output": tags}`

```
environments:
  vpc: 
    name: 'My VPC'
    templateName: 'vpc-template'
  subnet:
    name: 'My Subnets'
    templateName: 'subnet-template'
    needs:
      - vpc

```

## Requirements

The plugin uses the env0 API to fetch the output values from another environment. Therefore, we need to declare the ENV0_API_KEY and ENV0_API_SECRET environment variables in the environment or project with access to the source environments. You can either use [Organization API Keys](https://docs.env0.com/docs/api-keys) or [Personal API Keys](https://docs.env0.com/reference/authentication#creating-a-personal-api-key)
* `ENV0_API_KEY`
* `ENV0_API_SECRET` 

## Environment Name Restrictions

* Environment Names must be unique, otherwise, the script just uses the "first" matching environment name.
* Environment Names must not include spaces " " or slashes `/`. Ideally, your environment only contains alphanumeric characters and dashes `-`. **

## Inputs

N/A

## Example Usage

In this example we will run fetch the variable from a "Dev VPC" environment.

```yaml
version: 2
deploy:
  steps:
    terraformPlan:
      before:
        - name: Import Variables # The name that will be presented in the UI for this step
          use: https://github.com/env0/env0-import-variable-plugin@0.4.3
          inputs: {}

```
1. Configure the Custom Flow above with a new environment or an existing environment
2. Add a Terraform Variable
3. The `Key` is the name of your Terraform variable
4. The `Value` is a _reference_ to another environment's output variable.  For example, if I needed the VPC ID from my "Dev VPC" Environment:
  * First I need to get the ENV0_ENVIRONMENT_ID from that environment.
     note: the Environment ID can be found in the URL: `https://app.env0.com/p/7320dd7a-4822-426c-84b5-62ddd8be0799/environments/9cec1eb6-c17f-4cca-9cdf-606a23cdf6b5` where `9cec1eb6-c17f-4cca-9cdf-606a23cdf6b5` is the ENV0_ENVIRONMENT_ID.
  * Find the output name in the environment Resources tab.  e.g. `vpc_id`
  * The value you enter would be then: `${env0:9cec1eb6-c17f-4cca-9cdf-606a23cdf6b5:vpc_id}`
5. Run the environment, and env0 will fetch the value

## Further Reading

This plugin takes advantage of [Terraform variable precendence](https://developer.hashicorp.com/terraform/language/values/variables#variable-definition-precedence) and *.auto.tfvars. 


** If you must know why, it's because this script is written in Bash, and I'm taking advantage of Bash arrays which doesn't process spaces well, and also I'm saving the outputs to the filesystem which gets confused with slashes.
