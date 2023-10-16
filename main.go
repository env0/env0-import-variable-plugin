package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

type var_type int

const (
	tfvar var_type = iota
	envvar
)

// env0JSONVarByName is the data structure in JSON format to pull JSON output.
// UI restricts JSON input so the input would look something like: `{"ENV0_ENVIRONMENT_NAME":"my-vpc-environment", "Output":"my-vpc-id"}`
// OR for Workflows it would look like this: `{"ENV0_WORKFLOW_PARENT": "vpc", "Output": "my-vpc-id"}`
type env0JSONVarByName struct {
	ENV0_ENVIRONMENT_NAME string
	ENV0_WORKFLOW_PARENT  string
	Output                string
}

// env0Settings are used to pull and store the environment variables defined in the runner
type env0Settings struct {
	ENV0_ORGANIZATION_ID string
	ENV0_API_KEY         string
	ENV0_API_SECRET      string
	ENV0_ENVIRONMENT_ID  string
	APIKEYSECRET_ENCODED string // from TF_TOKEN_backend_api_env0_com
	TF_LOG               string // reuse TF_LOG level
}

// env0VariableToImport is a data structure to save what variable(s) needs to be fetched / imported.
type env0VariableToImport struct {
	InputKey              string
	InputType             interface{}
	ENV0_ENVIRONMENT_ID   string
	ENV0_ENVIRONMENT_NAME string
	OutputKey             string
	OutputType            string
	VariableType          var_type //tfvar or envvar
	GenericOutputValue    interface{}
}

// --- Data Structure for Logs
// tfVars - data structure for a Terraform Variable in the DeploymentLog
type tfVars struct {
	Sensitive bool        `json:"sensitive"`
	Type      interface{} `json:"type"`
	Value     interface{} `json:"value"`
}

// workflowLog - data structure for a workflow in the DeploymentLog
type workflowLog struct {
	Id                    string        `json:"id"`
	Name                  string        `json:"name"`
	WorkflowEnvironmentId string        `json:"workflowEnvironmentId"`
	LatestDeploymentLog   deploymentLog `json:"latestDeploymentLog"`
}

// deploymentLog - data structure for a deployment inside the EnvironmentLog
type deploymentLog struct {
	Output       map[string]tfVars `json:"output"`
	WorkflowFile workflowFile      `json:"workflowFile"`
}

// environmentLog - data structure for environmentLog
type environmentLog struct {
	Id                    string        `json:"id"`
	Name                  string        `json:"name"`
	WorkflowEnvironmentId string        `json:"workflowEnvironmentId"`
	LatestDeploymentLog   deploymentLog `json:"latestDeploymentLog"`
}

// --- Data Structure for Workflows ---
// workflowEnvironment - data structure for subEnvironments in a Workflow
type workflowEnvironment struct {
	Name          string `json:"name"`
	TemplateType  string `json:"templateType"`
	EnvironmentId string `json:"environmentId"`
}

// workflowFile - data structure for a complete workflow
type workflowFile struct {
	Environments map[string]workflowEnvironment `json:"environments"`
}

// setup environment variables
func (env *env0Settings) loadEnvs() {
	env.ENV0_API_KEY = os.Getenv("ENV0_API_KEY")
	env.ENV0_API_SECRET = os.Getenv("ENV0_API_SECRET")
	env.ENV0_ORGANIZATION_ID = os.Getenv("ENV0_ORGANIZATION_ID")
	env.ENV0_ENVIRONMENT_ID = os.Getenv("ENV0_ENVIRONMENT_ID")
	env.APIKEYSECRET_ENCODED = os.Getenv("TF_TOKEN_backend_api_env0_com")
	env.TF_LOG = os.Getenv("TF_LOG")
	if env.APIKEYSECRET_ENCODED == "" && (env.ENV0_API_SECRET == "" || env.ENV0_API_KEY == "") {
		sugar.Debugln("Error: ENV0_API_KEY, ENV0_API_SECRET or TF_TOKEN_backend_api_env0_com not found; please remember to set either the key and secret or the token.")
	} else if env.APIKEYSECRET_ENCODED == "" {
		env.APIKEYSECRET_ENCODED = base64.StdEncoding.EncodeToString([]byte(ENV0_SETTINGS.ENV0_API_KEY + ":" + ENV0_SETTINGS.ENV0_API_SECRET))
	}
}

// getEnvironmentIdByName
// updates importVars with the first ENV0_ENVIRONMENT_ID
func getEnvironmentIdByName(index int, importVars []env0VariableToImport) {
	sugar.Debugln("getEnvironmentIdByName: " + importVars[index].ENV0_ENVIRONMENT_NAME + " outputkey: " + importVars[index].OutputKey) // importVars[index].ENV0_ENVIRONMENT_NAME
	// sugar.Debugln("https://api.env0.com/environments?organizationId=" + ENV0_SETTINGS.ENV0_ORGANIZATION_ID + "&name=" + importVars[index].ENV0_ENVIRONMENT_NAME)
	req, _ := http.NewRequest("GET", "https://api.env0.com/environments?organizationId="+ENV0_SETTINGS.ENV0_ORGANIZATION_ID+"&name="+importVars[index].ENV0_ENVIRONMENT_NAME, nil)
	req.Header.Set("Authorization", "Basic "+ENV0_SETTINGS.APIKEYSECRET_ENCODED)
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		sugar.Fatalln("env0 API call error: ", resp.Status)
	}

	// TODO: Make environmentLogs a map, and check for existing logs.
	var environmentLog []environmentLog
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&environmentLog)
	// err = json.Unmarshal(resp.Body, &v)
	if err != nil {
		sugar.Fatalln("Issue unmarshalling environment log: ", err)
	} else {
		importVars[index].InputType = environmentLog[0].LatestDeploymentLog.Output[importVars[index].OutputKey].Type
		importVars[index].GenericOutputValue = environmentLog[0].LatestDeploymentLog.Output[importVars[index].OutputKey].Value
		importVars[index].ENV0_ENVIRONMENT_ID = environmentLog[0].Id
	}
}

// updateByName
// gets environment details by Name, Note: Environment Names aren't necessarily unique
// this "returns" first environment in matching Environemnt Names
func updateByName(index int, importVars []env0VariableToImport) {
	sugar.Debugln("updateByName: " + importVars[index].ENV0_ENVIRONMENT_NAME + " outputkey: " + importVars[index].OutputKey) // importVars[index].ENV0_ENVIRONMENT_NAME
	// sugar.Debugln("https://api.env0.com/environments?organizationId=" + ENV0_SETTINGS.ENV0_ORGANIZATION_ID + "&name=" + importVars[index].ENV0_ENVIRONMENT_NAME)
	req, _ := http.NewRequest("GET", "https://api.env0.com/environments?organizationId="+ENV0_SETTINGS.ENV0_ORGANIZATION_ID+"&name="+importVars[index].ENV0_ENVIRONMENT_NAME, nil)
	req.Header.Set("Authorization", "Basic "+ENV0_SETTINGS.APIKEYSECRET_ENCODED)
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		sugar.Fatalln("env0 API call error: ", resp.Status)
	}

	// TODO: Make environmentLogs a map, and check for existing logs.
	var environmentLog []environmentLog
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&environmentLog)
	// err = json.Unmarshal(resp.Body, &v)
	if err != nil {
		sugar.Fatalln("Issue unmarshalling environment log: ", err)
	} else {
		importVars[index].InputType = environmentLog[0].LatestDeploymentLog.Output[importVars[index].OutputKey].Type
		importVars[index].GenericOutputValue = environmentLog[0].LatestDeploymentLog.Output[importVars[index].OutputKey].Value
		importVars[index].ENV0_ENVIRONMENT_ID = environmentLog[0].Id
	}
}

// updateById
// gets environment details by envid
func updateById(index int, importVars []env0VariableToImport) {
	sugar.Debugln("updateById: " + importVars[index].ENV0_ENVIRONMENT_ID + " outputkey: " + importVars[index].OutputKey)
	req, _ := http.NewRequest("GET", "https://api.env0.com/environments/"+importVars[index].ENV0_ENVIRONMENT_ID, nil)
	req.Header.Set("Authorization", "Basic "+ENV0_SETTINGS.APIKEYSECRET_ENCODED)
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		sugar.Fatalln("env0 API call error: ", resp.Status)
	}

	// TODO: Make environmentLogs a map, and check for existing logs.
	var environmentLog environmentLog
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&environmentLog)
	// err = json.Unmarshal(resp.Body, &v)
	if err != nil {
		sugar.Fatalln("Issue unmarshalling environment log: ", err)
	} else {
		importVars[index].InputType = environmentLog.LatestDeploymentLog.Output[importVars[index].OutputKey].Type
		importVars[index].GenericOutputValue = environmentLog.LatestDeploymentLog.Output[importVars[index].OutputKey].Value
		importVars[index].ENV0_ENVIRONMENT_NAME = environmentLog.Name
	}
}

// getEnvironmentIdOfParent
// returns the envID of a parent in a workflow
func getEnvironmentIdOfParent(parentName string) string {
	sugar.Debugf("getEnvironmentIdOfParent: %s\n", parentName)

	req, _ := http.NewRequest("GET", "https://api.env0.com/environments/"+ENV0_SETTINGS.ENV0_ENVIRONMENT_ID, nil)
	req.Header.Set("Authorization", "Basic "+ENV0_SETTINGS.APIKEYSECRET_ENCODED)
	resp, _ := client.Do(req)
	//sugar.Debugln("\t", resp, err)
	//sugar.Debugln("\t", resp.Body)

	// TODO: Make environmentLogs a map, and check for existing logs.
	var environmentLog, workflowLog environmentLog
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&environmentLog)

	if err != nil {
		sugar.Warnln("Issue unmarshalling environment log: ", err)
	}

	sugar.Debugln("\t", environmentLog)
	sugar.Debugln("\t Workflow Id:", environmentLog.WorkflowEnvironmentId)

	req, _ = http.NewRequest("GET", "https://api.env0.com/environments/"+environmentLog.WorkflowEnvironmentId, nil)
	req.Header.Set("Authorization", "Basic "+ENV0_SETTINGS.APIKEYSECRET_ENCODED)
	resp, _ = client.Do(req)
	//sugar.Debugln("\t", resp, err)
	//sugar.Debugln("\t", resp.Body)

	// TODO: Make environmentLogs a map, and check for existing logs.

	decoder = json.NewDecoder(resp.Body)
	err = decoder.Decode(&workflowLog)

	if err != nil {
		sugar.Warnln("Issue unmarshalling workflow log: ", err)
	}

	sugar.Debugln("\t", workflowLog)
	sugar.Debugln("\t "+parentName+" \t Environment Id:", workflowLog.LatestDeploymentLog.WorkflowFile.Environments[parentName].EnvironmentId)

	return workflowLog.LatestDeploymentLog.WorkflowFile.Environments[parentName].EnvironmentId
}

func newHttpClient() *http.Client {
	return &http.Client{}
}

func loadVarsFile() (map[string]json.RawMessage, map[string]json.RawMessage) {

	var env0TfVars map[string]json.RawMessage
	// Load TF VARS FILE
	sugar.Debugln("Reading env0.auto.tfvars.json:")
	fi, err := os.ReadFile("env0.auto.tfvars.json")
	if err != nil {
		sugar.Fatal("Issue reading tfvars.json: ", err)
	}
	sugar.Debugf("%s\n", fi)
	sugar.Debugln("Loading env0.auto.tfvars.json")

	// UNMARSHALL to JSON
	err = json.Unmarshal(fi, &env0TfVars)
	if err != nil {
		sugar.Fatal("Issue unmarshalling tfvars.json: ", err)
	}

	var env0EnvVars map[string]json.RawMessage
	// Load ENV VARS FILE
	sugar.Debugln("Reading env0.env-vars.json:")
	fi, err = os.ReadFile("env0.env-vars.json")
	if err != nil {
		sugar.Fatal("Issue unmarshalling env-vars.json: ", err)
	}
	sugar.Debugf("%s\n", fi)
	sugar.Debugln("Loading env0.env-vars.json")

	// UNMARSHALL to JSON
	err = json.Unmarshal(fi, &env0EnvVars)
	if err != nil {
		sugar.Warnln(err)
	}

	return env0TfVars, env0EnvVars
}

// parseVars
// inputs: none
// outputs: []env0VariableToImport
// attempts to find all input variables matching accepted patterns
// ${...}, and json format {}
func parseVars(env0TfVars map[string]json.RawMessage, env0EnvVars map[string]json.RawMessage) []env0VariableToImport {

	var importVars []env0VariableToImport

	for k, v := range env0TfVars {
		sugar.Debugf("%s, %s", k, v)
	}

	for k, v := range env0EnvVars {
		sugar.Debugf("%s, %s", k, v)
	}

	sugar.Debugln("parse tfvars for matching regex patterns")
	for k, v := range env0TfVars {
		switch string(v[0:2]) {
		case "{\"":
			sugar.Debugf("found matching json: %s, need to parse: %s", k, v)
			var jsonRef env0JSONVarByName
			err := json.Unmarshal(v, &jsonRef)
			if err != nil {
				sugar.Warnf("error reading json: skipping %v: %v", k, err)
				continue
			}
			sugar.Debugf("parsed value: name: %s, parent: %s, output: %s\n", jsonRef.ENV0_ENVIRONMENT_NAME, jsonRef.ENV0_WORKFLOW_PARENT, jsonRef.Output)
			if jsonRef.ENV0_WORKFLOW_PARENT != "" {
				parentEnvId := getEnvironmentIdOfParent(jsonRef.ENV0_WORKFLOW_PARENT)
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_ID: parentEnvId, OutputKey: jsonRef.Output, OutputType: "json"})
			} else {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_NAME: jsonRef.ENV0_ENVIRONMENT_NAME, OutputKey: jsonRef.Output, OutputType: "json"})
			}
		case "\"$":
			sugar.Debugf("found match key: %s value: %s\n", k, v)
			s := strings.Split(string(v), ":")
			sugar.Debugf("tparsed value: %s, %s\n", s[1], s[2][:(len(s[2])-2)])
			matchWorkflow, err := regexp.MatchString(`\"\${env0-workflow`, s[0])
			if err != nil {
				sugar.Warnf("error reading workflow: skipping %v: %v", k, err)
				continue
			}
			sugar.Debugln("\t", s[0])
			if matchWorkflow {
				sugar.Debugln("Fetch Worfklow variable from: " + s[1] + " output: " + s[2][:len(s[2])-2])
				parentEnvId := getEnvironmentIdOfParent(s[1])
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_ID: parentEnvId, OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
				continue
			}
			matchedbyid, err := regexp.MatchString(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, s[1])
			if err != nil {
				sugar.Warnf("non matching regex: skipping %v: %v", k, err)
			}
			if matchedbyid {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_ID: s[1], OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
			} else {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_NAME: s[1], OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
			}
		default:
			sugar.Debugf("skipping key: %s", k)
		}
	}

	return importVars
}

// GLOBAL VARIABLES

var ENV0_SETTINGS env0Settings
var client *http.Client
var sugar *zap.SugaredLogger

/*
env0-import-variable-plugin takes variables configured in env0 UI and finds any
variables matching a certain regex pattern. For those it matches, it tries to
pull the corresponding values using the env0 API keys present in the environ-
ment.
*/
func main() {

	//logger, _ := zap.NewDevelopment()

	cfg := zap.NewDevelopmentConfig()
	logger, _ := cfg.Build()

	defer logger.Sync()

	sugar = logger.Sugar()
	sugar.Infoln("Hello, Import Variable Plugin!")

	// Read/Load local EnvVars for settings and credentials
	ENV0_SETTINGS.loadEnvs()

	// Using TF_LOG as the log level for import variable plugin
	if ENV0_SETTINGS.TF_LOG == "" || ENV0_SETTINGS.TF_LOG == "info" {
		cfg.Level.SetLevel(zap.InfoLevel)
	} else {
		cfg.Level.SetLevel(zap.DebugLevel)
	}

	// Configure global HTTP client
	client = newHttpClient()

	// Read input vars & parse the variables
	// first unmarshall json inputs `env0.auto.tfvars.json` & `env0.env-vars.json`
	var env0TfVars, env0EnvVars map[string]json.RawMessage
	env0TfVars, env0EnvVars = loadVarsFile()
	var importVars []env0VariableToImport
	importVars = parseVars(env0TfVars, env0EnvVars)

	OutputTFVarsJson := make(map[string]interface{})

	sugar.Debugln("call env0 API to fetch environments by ID or by name")
	for k, v := range importVars {
		if v.ENV0_ENVIRONMENT_ID == "" {
			updateByName(k, importVars)
		} else {
			updateById(k, importVars)
		}

		sugar.Debugf("InputType: %v\t", importVars[k].InputType)
		sugar.Debugf("OutputType: %v\n", importVars[k].OutputType)
		switch importVars[k].InputType.(type) {
		case string:
			if importVars[k].OutputType == "json" {
				OutputTFVarsJson[importVars[k].InputKey] = json.RawMessage((fmt.Sprint(importVars[k].GenericOutputValue)))
			} else {
				OutputTFVarsJson[importVars[k].InputKey] = importVars[k].GenericOutputValue
			}
		default:
			OutputTFVarsJson[importVars[k].InputKey] = importVars[k].GenericOutputValue
		}

		//OutputTFVarsJson[importVars[k].InputKey] = importVars[k].GenericOutputValue
	}

	sugar.Infoln("ImportVars: ", importVars)

	sugar.Infoln("OutputVars: ", OutputTFVarsJson)

	sugar.Debugln("parse for outputs and save/Marshall outputs")

	fo, err := json.Marshal(&OutputTFVarsJson)
	if err != nil {
		sugar.Warn(err)
	}

	err = os.WriteFile("env1.auto.tfvars.json", fo, 0666)
	if err != nil {
		sugar.Warn(err)
	}

	sugar.Infoln("Import Variable Plugin is Done!")
}
