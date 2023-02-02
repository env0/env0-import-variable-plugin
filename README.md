# Variable Helper plugin

This plugin will fetch output values from another environment and insert them as terraform variables.


Similar to self hosted agent secrets, use this notation in the value of the terraform input value:

`${env0:<environment id>:<output name>}`

## Requirements

The ENV0 API KEY and SECRET with access to the target environments.
* `ENV0_API_KEY`
* `ENV0_API_SECRET` 

The plugin uses the env0 API to fetch the output values from another environment.

## Inputs

N/A

## Example Usage

In this example we will run fetch the variable from a "Dev VPC" environment.

```yaml
version: 2
deploy:
  steps:
    setupVariables:
      after:
        - name: Fetch Variables # The name that will be presented in the UI for this step
          use: https://github.com/env0/env0-import-variable-plugin

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

This plugin takes advantage of [Terraform variable precendence](https://developer.hashicorp.com/terraform/language/values/variables#variable-definition-precedence) and auto.tfvars behavior. 
