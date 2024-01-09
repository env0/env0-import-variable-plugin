#!/bin/bash
########################
# Fetch Variables 
# env0 Plugin to Pull outputs from an existing environment and save them as input variables
#
########################

# Parse all tf vars from env0.env-envs.json
# look for each key, check value in format of ${env0:envid:outputname} or ${env0-workflow:parentname:outputname}
# fetch the output variable using API and match outputname, then write key value with a new file env1.auto.tfvars.json

# https://developer.hashicorp.com/terraform/language/values/variables#variable-definition-precedence
# lexical order, so last file wins!

### Repeat process for Environment Variables

KEYS=($(jq -rc 'keys_unsorted | .[]' env0.env-vars.json))
VALUES=($(jq -c '.[]' env0.env-vars.json))
LENGTH=$(jq 'length' env0.env-vars.json)

UUID_REGEX='[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'

# write to ENV0_ENV
# for each variable in env0.env-vars.json 
for ((i = 0; i < LENGTH; i++)); do
  # check for environment id (UUID) format
  if [[ ${VALUES[i]} =~ ^\"\$\{env0:$UUID_REGEX:.*\}\"$ ]]; then
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

  # check for ${env0-workflow:parentname:output}
  elif [[ ${VALUES[i]} =~ ^(\"\$\{env0-workflow:.*:.*\}\")$ ]]; then
    echo ${KEYS[i]}:${VALUES[i]}
    SPLIT_VALUES=($(echo ${VALUES[i]} | tr ":" "\n")) 
    SOURCE_PARENT_NAME=${SPLIT_VALUES[1]}
    len=$((${#SPLIT_VALUES[2]}-2))
    SOURCE_OUTPUT_NAME=${SPLIT_VALUES[2]:0:$len}
    echo "fetch value for ${KEYS[i]}:$SOURCE_OUTPUT_NAME from parent ${SOURCE_PARENT_NAME}"

    # check if current environment is part of a workflow

    if [[ ! -e $ENV0_ENVIRONMENT_ID.json ]]; then 
      curl -s --request GET \
        --url "https://api.env0.com/environments/$ENV0_ENVIRONMENT_ID" \
        --header 'accept: application/json' \
        -u $ENV0_API_KEY:$ENV0_API_SECRET \
        -o $ENV0_ENVIRONMENT_ID.json
    fi

    WORKFLOW_ENVIRONMENT_ID="$(jq -r '.workflowEnvironmentId' "$ENV0_ENVIRONMENT_ID.json")"
    if [[ ! $WORKFLOW_ENVIRONMENT_ID =~ $UUID_REGEX ]]; then
      echo "$ENV0_ENVIRONMENT_ID is not part of a workflow. skipping ${KEYS[i]}:${VALUES[i]}"
      continue
    fi

    # get requested source parent environment id from workflow

    if [[ ! -e $WORKFLOW_ENVIRONMENT_ID.json ]]; then 
      curl -s --request GET \
        --url "https://api.env0.com/environments/$WORKFLOW_ENVIRONMENT_ID" \
        --header 'accept: application/json' \
        -u $ENV0_API_KEY:$ENV0_API_SECRET \
        -o $WORKFLOW_ENVIRONMENT_ID.json
    fi 

    SOURCE_PARENT_UUID="$(jq -r ".latestDeploymentLog.workflowFile.environments.$SOURCE_PARENT_NAME.environmentId" "$WORKFLOW_ENVIRONMENT_ID.json")"
    if [[ ! $SOURCE_PARENT_UUID =~ $UUID_REGEX ]]; then
      echo "$SOURCE_PARENT_NAME not found in current workflow. skipping ${KEYS[i]}:${VALUES[i]}"
      continue
    fi
    
    # get requested source output

    if [[ ! -e $SOURCE_PARENT_UUID.json ]]; then 
      curl -s --request GET \
        --url "https://api.env0.com/environments/$SOURCE_PARENT_UUID" \
        --header 'accept: application/json' \
        -u $ENV0_API_KEY:$ENV0_API_SECRET \
        -o $SOURCE_PARENT_UUID.json
    fi 

    SOURCE_OUTPUT_VALUE="$(jq -r ".latestDeploymentLog.output.$SOURCE_OUTPUT_NAME.value" "$SOURCE_PARENT_UUID.json")"
    echo "${KEYS[i]}=$SOURCE_OUTPUT_VALUE"
    echo "${KEYS[i]}=$SOURCE_OUTPUT_VALUE" >> $ENV0_ENV 
  fi
done
