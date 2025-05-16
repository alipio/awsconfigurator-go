package configurator

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

func (c *Config) Validate() error {
	if c.Region == "" {
		return &InvalidConfigError{Message: "region is missing"}
	}

	if c.Prefix == "" {
		return &InvalidConfigError{Message: "prefix is missing"}
	}

	if c.Environment == "" {
		return &InvalidConfigError{Message: "environment is missing"}
	}

	if c.AccountID == "" {
		return &InvalidConfigError{Message: "account ID is missing"}
	}

	if len(c.SNSTopics) > 0 {
		for _, t := range c.SNSTopics {
			if t.Name == "" {
				return &InvalidConfigError{Message: "some topic entries have missing names"}
			}
		}
	}

	if len(c.Queues) > 0 {
		for _, q := range c.Queues {
			if q.Name == "" {
				return &InvalidConfigError{Message: "some queue entries have missing names"}
			}

			if len(q.SNSTopics) > 0 {
				for _, t := range q.SNSTopics {
					if t.Name == "" {
						return &InvalidConfigError{
							Message: fmt.Sprintf("%q has items with missing names: queue=%s", "sns_topics", q.Name),
						}
					}
				}
			}
		}
	}

	return nil
}

func LoadConfig(filename string) (*Config, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	expanded := expandEnvVars(string(contents))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil { //nolint:govet // shadowing is ok here.
		return nil, err
	}

	if err := cfg.Validate(); err != nil { //nolint:govet // shadowing is ok here.
		return nil, err
	}

	return &cfg, nil
}

func expandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		return ""
	})
}
