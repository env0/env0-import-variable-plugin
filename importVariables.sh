#!/bin/bash
########################
# Fetch Variables 
# env0 Plugin to Pull outputs from an existing environment and save them as input variables
#
########################

# Parse all tf vars from env0.env-envs.json
# look for each key, check value in format of ${env0:envid:outputname}
# fetch the output variable using API and match outputname, then write key value with a new file env1.auto.tfvars.json

# https://developer.hashicorp.com/terraform/language/values/variables#variable-definition-precedence
# lexical order, so last file wins!

### Repeat process for Environment Variables

KEYS=($(jq -rc 'keys | .[]' env0.env-vars.json))
VALUES=($(jq -c '.[]' env0.env-vars.json))
LENGTH=$(jq 'length' env0.env-vars.json)

# write to ENV0_ENV
# for each variable in env0.env-vars.json 
for ((i = 0; i < LENGTH; i++)); do
  # check for environment id (UUID) format
  if [[ ${VALUES[i]} =~ ^\"\$\{env0:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}:.*\}\"$ ]]; then
    echo ${KEYS[i]}:${VALUES[i]}
    # split the string across ':'
    SPLIT_VALUES=($(echo ${VALUES[i]} | tr ":" "\n"))
    SOURCE_ENV0_ENVIRONMENT_ID=${SPLIT_VALUES[1]}
    len=$((${#SPLIT_VALUES[2]}-2))
    SOURCE_OUTPUT_NAME=${SPLIT_VALUES[2]:0:$len}
    echo "fetch value for ${KEYS[i]}:$SOURCE_OUTPUT_NAME from ${SOURCE_ENV0_ENVIRONMENT_ID}"

    # fetch logs from environment
    if [[ ! -e $SOURCE_ENV0_ENVIRONMENT_ID.json ]]; then
      curl -s --request GET \
      --url https://api.env0.com/environments/$SOURCE_ENV0_ENVIRONMENT_ID \
      --header 'accept: application/json' \
      -u $ENV0_API_KEY:$ENV0_API_SECRET \
      -o $SOURCE_ENV0_ENVIRONMENT_ID.json
    fi 

    # fetch value from environment 
    SOURCE_OUTPUT_VALUE=$(jq ".latestDeploymentLog.output.$SOURCE_OUTPUT_NAME.value" $SOURCE_ENV0_ENVIRONMENT_ID.json)
    #echo $SOURCE_OUTPUT_VALUE
    echo "${KEYS[i]}=$SOURCE_OUTPUT_VALUE"
    echo "${KEYS[i]}=$SOURCE_OUTPUT_VALUE" >> $ENV0_ENV

  # check for ${env0:environmentname:output}
  elif [[ ${VALUES[i]} =~ ^(\"\$\{env0:.*:.*\}\")$ ]]; then
    echo ${KEYS[i]}:${VALUES[i]}
    SPLIT_VALUES=($(echo ${VALUES[i]} | tr ":" "\n")) 
    SOURCE_ENV0_ENVIRONMENT_NAME=${SPLIT_VALUES[1]}
    len=$((${#SPLIT_VALUES[2]}-2))
    SOURCE_OUTPUT_NAME=${SPLIT_VALUES[2]:0:$len}
    echo "fetch value for ${KEYS[i]}:$SOURCE_OUTPUT_NAME from ${SOURCE_ENV0_ENVIRONMENT_NAME}"

    if [[ ! -e $SOURCE_ENV0_ENVIRONMENT_NAME.json ]]; then
      curl -s --request GET \
      --url "https://api.env0.com/environments?organizationId=$ENV0_ORGANIZATION_ID&name=$SOURCE_ENV0_ENVIRONMENT_NAME" \
      --header 'accept: application/json' \
      -u $ENV0_API_KEY:$ENV0_API_SECRET \
      -o $SOURCE_ENV0_ENVIRONMENT_NAME.json
    fi

    SOURCE_OUTPUT_VALUE=$(jq ".[0].latestDeploymentLog.output.$SOURCE_OUTPUT_NAME.value" $SOURCE_ENV0_ENVIRONMENT_NAME.json)
    #echo $SOURCE_OUTPUT_VALUE
    echo "${KEYS[i]}=$SOURCE_OUTPUT_VALUE"
    echo "${KEYS[i]}=$SOURCE_OUTPUT_VALUE" >> $ENV0_ENV 
  fi
done
