package config

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"

	"sigs.k8s.io/yaml"
)

func ParseMatricesConfigFile(configFile string) (*v1.MatricesSpec, error) {
	var err error
	var configYaml []byte
	fmt.Println("Reading from", configFile)
	if configFile == "-" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			configYaml = append(configYaml, scanner.Bytes()...)
			configYaml = append(configYaml, '\n')
		}
	} else {
		configYaml, err = ioutil.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("read error: %v", err)
		}
	}
	var spec v1.MatricesSpec
	err = yaml.Unmarshal(configYaml, &spec)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %v", err)
	}
	fmt.Println("--")
	str, err := yaml.Marshal(spec)
	fmt.Println(string(str))
	fmt.Println("--")
	return &spec, nil
}
