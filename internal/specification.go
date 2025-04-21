package main

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Specification struct {
	LlmUrl    string `split_words:"true"`
	LlmApiKey string `split_words:"true"`
	LlmModel  string `split_words:"true"`

	EmailTo       string `split_words:"true"`
	EmailFrom     string `split_words:"true"`
	EmailHost     string `split_words:"true"`
	EmailPort     string `split_words:"true"`
	EmailUsername string `split_words:"true"`
	EmailPassword string `split_words:"true"`

	DebugMockRss         bool `split_words:"true"`
	DebugMockLLM         bool `split_words:"true"`
	DebugMockSkipEmail   bool `split_words:"true"`
	DebugOutputBenchmark bool `split_words:"true"`
}

func GetConfig() (*Specification, error) {
	var s Specification
	err := envconfig.Process("anp", &s)
	if err != nil {
		return nil, fmt.Errorf("could not load specification: %w", err)
	}

	return &s, nil
}
