package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

/*
env0-import-variable-plugin takes variables configured in env0 UI and finds any
variables matching a certain regex pattern. For those it matches, it tries to
pull the corresponding values using the env0 API keys present in the environ-
ment.
*/
func main() {
	fmt.Println("Hello, Import Variable Plugin!")

	fmt.Println("Open env0.auto.tfvars.json")
	fi, err := os.ReadFile("env0.auto.tfvars.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("cat env0.auto.tfvars.json:\n%s\n", fi)

	fmt.Println("UnMarshall env0.auto.tfvars.json")

	var env0TfVars map[string]json.RawMessage

	err = json.Unmarshal(fi, &env0TfVars)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range env0TfVars {
		var jsonValue TfVarsJSON
		err = json.Unmarshal(v, &jsonValue)
		if err != nil {
			log.Fatal(err)
		}
		// switch jsonValue.Type {

		// }
		fmt.Println(k, v)
	}

	fmt.Println(env0TfVars)

	fmt.Println("parse tfvars for matching regex patterns")

	fmt.Println("call API to fetch environments by ID or by name")

	fmt.Println("parse for outputs and save/Marshall outputs")

	fmt.Println("Done")
}
