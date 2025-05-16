package configurator

import "context"

type AwsProvider interface {
	CreateTopic(ctx context.Context, topic Topic) (string, error)
	CreateQueue(ctx context.Context, queue Queue) (string, error)
	Subscribe(ctx context.Context, queue Queue, topics []Topic) error
}

type Config struct {
	Region      string  `yaml:"region"`
	AccountID   string  `yaml:"account_id"`
	Prefix      string  `yaml:"prefix"`
	Environment string  `yaml:"environment"`
	Queues      []Queue `yaml:"queues"`
	SNSTopics   []Topic `yaml:"sns_topics"`
}

type Topic struct {
	Name        string `yaml:"name"`
	Prefix      string `yaml:"prefix"`
	DisplayName string `yaml:"display_name"`
	IsFIFO      bool   `yaml:"fifo"`
	FullName    string
}

type Queue struct {
	Name              string    `yaml:"name"`
	SNSTopics         []Topic   `yaml:"sns_topics"`
	DeadLetterQueue   DLQConfig `yaml:"dead_letter_queue"`
	MessageRetention  int       `yaml:"message_retention_seconds"`
	VisibilityTimeout int       `yaml:"visibility_timeout_seconds"`
	IsFIFO            bool      `yaml:"fifo"`
	FullName          string
	URL               string
}

type DLQConfig struct {
	Enabled         bool   `yaml:"enabled"`
	MaxReceiveCount int    `yaml:"max_receive_count"`
	Suffix          string `yaml:"suffix"`
}
