package config

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Location string  `yaml:"location"`
	FeedIds  []int64 `yaml:"feedIds"`
	DbFile   string  `yaml:"dbFile"`
	OpenAi   struct {
		Model  string `yaml:"model"`
		ApiKey string `yaml:"api-key"`
	} `yaml:"open-ai"`
	Github struct {
		Repo   string `yaml:"repo"`
		Branch string `yaml:"branch"`
		Path   string `yaml:"path"`
		Pat    string `yaml:"pat"`
	} `yaml:"github"`
}

const yamlConfigurationTemplate = `
# Configuration example

location: #add your location here like Europe/Madrid

feedIds: #add your feed ids here like [3, 4, 6, 11]

dbFile: #add your full freshRss db file here like db.sqlite

open-ai:
  model: #add your openai model here like gpt-4o-mini
  api-key: #add your openai api key here

github:
  repo: #add your github repo here like enolgor/news
  branch: #add your github branch here like main
  path: #add your github path here like reports
  pat: #add your github personal access token here
`

func ParseConfig(path string) (*Configuration, error) {
	config := &Configuration{}
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("could not open config file: %s\n", err.Error())
	}
	defer file.Close()
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(config); err != nil {
		log.Fatalf("could not decode config file: %s\n", err.Error())
	}
	return config, nil
}

func EnsureConfigFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Config file not found. Creating default config at: %s\n", path)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("could not create config directory: %s\n", err.Error())
		}
		if err := os.WriteFile(path, []byte(yamlConfigurationTemplate), 0644); err != nil {
			log.Fatalf("could not create config file: %s\n", err.Error())
		}
		os.Exit(0)
	}
}
