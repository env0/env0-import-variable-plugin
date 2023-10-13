package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
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
}

// env0VariableToImport is a data structure to save what variable(s) needs to be fetched / imported.
type env0VariableToImport struct {
	InputKey              string
	ENV0_ENVIRONMENT_ID   string
	ENV0_ENVIRONMENT_NAME string
	OutputKey             string
	OutputType            string
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
	if env.APIKEYSECRET_ENCODED == "" && (env.ENV0_API_SECRET == "" || env.ENV0_API_KEY == "") {
		log.Println("Error: ENV0_API_KEY, ENV0_API_SECRET or TF_TOKEN_backend_api_env0_com not found; please remember to set either the key and secret or the token.")
	} else if env.APIKEYSECRET_ENCODED == "" {
		env.APIKEYSECRET_ENCODED = base64.StdEncoding.EncodeToString([]byte(env0EnvVars.ENV0_API_KEY + ":" + env0EnvVars.ENV0_API_SECRET))
	}
}

// updateByName
// gets environment details by Name, Note: Environment Names aren't necessarily unique
// this "returns" first environment in matching Environemnt Names
func updateByName(index int, importVars []env0VariableToImport) {
	log.Println("updateByName: " + importVars[index].ENV0_ENVIRONMENT_NAME + " outputkey: " + importVars[index].OutputKey) // importVars[index].ENV0_ENVIRONMENT_NAME
	// log.Println("https://api.env0.com/environments?organizationId=" + env0EnvVars.ENV0_ORGANIZATION_ID + "&name=" + importVars[index].ENV0_ENVIRONMENT_NAME)
	req, _ := http.NewRequest("GET", "https://api.env0.com/environments?organizationId="+env0EnvVars.ENV0_ORGANIZATION_ID+"&name="+importVars[index].ENV0_ENVIRONMENT_NAME, nil)
	req.Header.Set("Authorization", "Basic "+env0EnvVars.APIKEYSECRET_ENCODED)
	resp, err := client.Do(req)
	// log.Println(resp, err)
	// log.Println(resp.Body)

	// TODO: Make environmentLogs a map, and check for existing logs.
	var environmentLog []environmentLog
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&environmentLog)
	// err = json.Unmarshal(resp.Body, &v)
	if err != nil {
		log.Fatalln(err)
	} else {
		importVars[index].GenericOutputValue = environmentLog[0].LatestDeploymentLog.Output[importVars[index].OutputKey].Value
		importVars[index].ENV0_ENVIRONMENT_ID = environmentLog[0].Id
	}
}

// updateById
// gets environment details by envid
func updateById(index int, importVars []env0VariableToImport) {
	log.Println("updateById: " + importVars[index].ENV0_ENVIRONMENT_ID + " outputkey: " + importVars[index].OutputKey)
	req, _ := http.NewRequest("GET", "https://api.env0.com/environments/"+importVars[index].ENV0_ENVIRONMENT_ID, nil)
	req.Header.Set("Authorization", "Basic "+env0EnvVars.APIKEYSECRET_ENCODED)
	resp, err := client.Do(req)
	// log.Println(resp, err)
	// log.Println(resp.Body)

	// TODO: Make environmentLogs a map, and check for existing logs.
	var environmentLog environmentLog
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&environmentLog)
	// err = json.Unmarshal(resp.Body, &v)
	if err != nil {
		log.Fatalln(err)
	} else {
		importVars[index].GenericOutputValue = environmentLog.LatestDeploymentLog.Output[importVars[index].OutputKey].Value
		importVars[index].ENV0_ENVIRONMENT_NAME = environmentLog.Name
	}
}

// getEnvironmentIdOfParent
// returns the envID of a parent in a workflow
func getEnvironmentIdOfParent(parentName string) string {
	log.Printf("getEnvironmentIdOfParent: %s\n", parentName)

	req, _ := http.NewRequest("GET", "https://api.env0.com/environments/"+env0EnvVars.ENV0_ENVIRONMENT_ID, nil)
	req.Header.Set("Authorization", "Basic "+env0EnvVars.APIKEYSECRET_ENCODED)
	resp, _ := client.Do(req)
	//log.Println("\t", resp, err)
	//log.Println("\t", resp.Body)

	// TODO: Make environmentLogs a map, and check for existing logs.
	var environmentLog, workflowLog environmentLog
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&environmentLog)

	if err != nil {
		log.Fatalln(err)
	}

	log.Println("\t", environmentLog)
	log.Println("\t Workflow Id:", environmentLog.WorkflowEnvironmentId)

	req, _ = http.NewRequest("GET", "https://api.env0.com/environments/"+environmentLog.WorkflowEnvironmentId, nil)
	req.Header.Set("Authorization", "Basic "+env0EnvVars.APIKEYSECRET_ENCODED)
	resp, _ = client.Do(req)
	//log.Println("\t", resp, err)
	//log.Println("\t", resp.Body)

	// TODO: Make environmentLogs a map, and check for existing logs.

	decoder = json.NewDecoder(resp.Body)
	err = decoder.Decode(&workflowLog)

	if err != nil {
		log.Fatalln(err)
	}

	log.Println("\t", workflowLog)
	log.Println("\t "+parentName+" \t Environment Id:", workflowLog.LatestDeploymentLog.WorkflowFile.Environments[parentName].EnvironmentId)

	return workflowLog.LatestDeploymentLog.WorkflowFile.Environments[parentName].EnvironmentId
}

func newHttpClient() *http.Client {
	return &http.Client{}
}

// GLOBAL VARIABLES

var env0EnvVars env0Settings
var importVars []env0VariableToImport
var client *http.Client

/*
env0-import-variable-plugin takes variables configured in env0 UI and finds any
variables matching a certain regex pattern. For those it matches, it tries to
pull the corresponding values using the env0 API keys present in the environ-
ment.
*/
func main() {

	// Set Log Format
	log.SetFlags(log.Lshortfile)
	log.Println("Hello, Import Variable Plugin!")

	env0EnvVars.loadEnvs()

	client = newHttpClient()

	var env0TfVars map[string]json.RawMessage

	// Load TF VARS FILE
	log.Println("Reading env0.auto.tfvars.json:")
	fi, err := os.ReadFile("env0.auto.tfvars.json")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s\n", fi)

	log.Println("Loading env0.auto.tfvars.json")

	// UNMARSHALL to JSON
	err = json.Unmarshal(fi, &env0TfVars)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("parse tfvars for matching regex patterns")
	for k, v := range env0TfVars {
		switch string(v[0:2]) {
		case "{\"":
			log.Printf("\tfound matching json: %s, need to parse: %s\n", k, v)
			var jsonRef env0JSONVarByName
			err = json.Unmarshal(v, &jsonRef)
			if err != nil {
				log.Printf("\t\terror reading json: skipping %v: %v", k, err)
				continue
			}
			log.Printf("\tparsed value: name: %s, parent: %s, output: %s\n", jsonRef.ENV0_ENVIRONMENT_NAME, jsonRef.ENV0_WORKFLOW_PARENT, jsonRef.Output)
			if jsonRef.ENV0_WORKFLOW_PARENT != "" {
				parentEnvId := getEnvironmentIdOfParent(jsonRef.ENV0_WORKFLOW_PARENT)
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_ID: parentEnvId, OutputKey: jsonRef.Output, OutputType: "json"})
			} else {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_NAME: jsonRef.ENV0_ENVIRONMENT_NAME, OutputKey: jsonRef.Output, OutputType: "json"})
			}
		case "\"$":
			log.Printf("\tfound match key: %s value: %s\n", k, v)
			s := strings.Split(string(v), ":")
			log.Printf("\tparsed value: %s, %s\n", s[1], s[2][:(len(s[2])-2)])
			matchWorkflow, err := regexp.MatchString(`\"\${env0-workflow`, s[0])
			if err != nil {
				log.Printf("\t\terror reading workflow: skipping %v: %v", k, err)
				continue
			}
			log.Println("\t", s[0])
			if matchWorkflow {
				log.Println("\t\tFetch Worfklow variable from: " + s[1] + " output: " + s[2][:len(s[2])-2])
				parentEnvId := getEnvironmentIdOfParent(s[1])
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_ID: parentEnvId, OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
				continue
			}
			matchedbyid, err := regexp.MatchString(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, s[1])
			if err != nil {
				log.Fatalln("non matching regex: ", err)
			}
			if matchedbyid {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_ID: s[1], OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
			} else {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_NAME: s[1], OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
			}
		default:
			log.Printf("\tskipping key: %s", k)
		}
	}

	log.Println("call API to fetch environments by ID or by name")

	OutputTFVarsJson := make(map[string]interface{})

	for k, v := range importVars {
		if v.ENV0_ENVIRONMENT_ID == "" {
			updateByName(k, importVars)
		} else {
			updateById(k, importVars)
		}
		OutputTFVarsJson[importVars[k].InputKey] = importVars[k].GenericOutputValue
	}

	log.Println("ImportVars: ", importVars)

	log.Println("OutputVars: ", OutputTFVarsJson)

	log.Println("parse for outputs and save/Marshall outputs")

	fo, err := json.Marshal(&OutputTFVarsJson)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("env1.auto.tfvars.json", fo, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
}
