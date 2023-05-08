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

type env0JSONVarByName struct {
	ENV0_ENVIRONMENT_NAME string
	Output                string
}

type env0Settings struct {
	ENV0_ORGANIZATION_ID string
	ENV0_API_KEY         string
	ENV0_API_SECRET      string
}

var env0EnvVars env0Settings

type env0VariableToImport struct {
	InputKey              string
	ENV0_ENVIRONMENT_ID   string
	ENV0_ENVIRONMENT_NAME string
	OutputKey             string
	OutputValue           string
	OutputType            string
}

type environmentLog struct {
	Id                  string        `json:"id"`
	Name                string        `json:"name"`
	LatestDeploymentLog deploymentLog `json:"latestDeploymentLog"`
}

type deploymentLog struct {
	Output       map[string]tfVars `json:"output"`
	WorkflowFile workflowFile      `json:"workflowFile"`
}

type tfVars struct {
	Sensitive bool   `json:"sensitive"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type workflowFile interface{}

var client *http.Client

func newHttpClient() *http.Client {
	return &http.Client{}
}

var APIKEYSECRET_ENCODED string

func getEnvs(env *env0Settings) {
	env.ENV0_API_KEY = os.Getenv("ENV0_API_KEY")
	env.ENV0_API_SECRET = os.Getenv("ENV0_API_SECRET")
	env.ENV0_ORGANIZATION_ID = os.Getenv(("ENV0_ORGANIZATION_ID"))
	APIKEYSECRET_ENCODED = base64.StdEncoding.EncodeToString([]byte(env0EnvVars.ENV0_API_KEY + ":" + env0EnvVars.ENV0_API_SECRET))
}

func updateEnvironmentIdFromName(index int, importVars []env0VariableToImport) {
	// importVars[index].ENV0_ENVIRONMENT_NAME
	// log.Println("https://api.env0.com/environments?organizationId=" + env0EnvVars.ENV0_ORGANIZATION_ID + "&name=" + importVars[index].ENV0_ENVIRONMENT_NAME)
	req, _ := http.NewRequest("GET", "https://api.env0.com/environments?organizationId="+env0EnvVars.ENV0_ORGANIZATION_ID+"&name="+importVars[index].ENV0_ENVIRONMENT_NAME, nil)
	req.Header.Set("Authorization", "Basic "+APIKEYSECRET_ENCODED)
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
		log.Println("\tOutput Value: " + environmentLog[0].LatestDeploymentLog.Output[importVars[index].OutputKey].Value)
		importVars[index].OutputValue = environmentLog[0].LatestDeploymentLog.Output[importVars[index].OutputKey].Value
		importVars[index].ENV0_ENVIRONMENT_ID = environmentLog[0].Id
	}
}

/*
env0-import-variable-plugin takes variables configured in env0 UI and finds any
variables matching a certain regex pattern. For those it matches, it tries to
pull the corresponding values using the env0 API keys present in the environ-
ment.
*/
func main() {
	log.SetFlags(log.Lshortfile)

	log.Println("Hello, Import Variable Plugin!")

	client = newHttpClient()

	getEnvs(&env0EnvVars)

	var importVars []env0VariableToImport

	log.Println("Open env0.auto.tfvars.json")
	fi, err := os.ReadFile("env0.auto.tfvars.json")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("cat env0.auto.tfvars.json:\n%s\n", fi)

	log.Println("UnMarshall env0.auto.tfvars.json")

	var env0TfVars map[string]json.RawMessage

	err = json.Unmarshal(fi, &env0TfVars)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("parse tfvars for matching regex patterns")

	for k, v := range env0TfVars {
		switch string(v[0:2]) {
		case "{\"":
			log.Printf("key: %s, need to parse json: %s\n", k, v)
			var jsonRef env0JSONVarByName
			err = json.Unmarshal(v, &jsonRef)
			log.Printf(" parsed value: %s, %s\n", jsonRef.ENV0_ENVIRONMENT_NAME, jsonRef.Output)
			importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_NAME: jsonRef.ENV0_ENVIRONMENT_NAME, OutputKey: jsonRef.Output, OutputType: "json"})
		case "\"$":
			log.Printf("found match: key: %s value: %s\n", k, v)
			s := strings.Split(string(v), ":")
			log.Printf(" parsed value: %s, %s\n", s[1], s[2][:(len(s[2])-2)])
			matched, err := regexp.MatchString(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, s[1])
			if err != nil {
				log.Fatalln("non matching regex: ", err)
			}
			if matched {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_ID: s[1], OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
			} else {
				importVars = append(importVars, env0VariableToImport{InputKey: k, ENV0_ENVIRONMENT_NAME: s[1], OutputKey: s[2][:len(s[2])-2], OutputType: "string"})
			}
		default:
			log.Printf("ignoring key: %s, value: %s\n", k, v[0:2])
		}
	}

	log.Println("call API to fetch environments by ID or by name")

	OutputTFVarsJson := make(map[string]interface{})

	for k, v := range importVars {
		if v.ENV0_ENVIRONMENT_ID == "" {
			updateEnvironmentIdFromName(k, importVars)
		}
		switch importVars[k].OutputType {
		case "json":
			OutputTFVarsJson[importVars[k].OutputKey] = json.RawMessage(importVars[k].OutputValue)
		default:
			OutputTFVarsJson[importVars[k].OutputKey] = importVars[k].OutputValue
		}
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
