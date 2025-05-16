package configurator

import (
	"context"
	"fmt"
	"strings"
)

type Configurator struct {
	provider AwsProvider
	cfg      *Config
}

func New(provider AwsProvider, cfg *Config) *Configurator {
	return &Configurator{
		provider: provider,
		cfg:      cfg,
	}
}

func (c *Configurator) Run(ctx context.Context) error {
	// Invalid config, give up early.
	if err := c.cfg.Validate(); err != nil {
		return err
	}
	// Nothing to do.
	if len(c.cfg.Queues) == 0 && len(c.cfg.SNSTopics) == 0 {
		return nil
	}

	for _, topic := range c.cfg.SNSTopics {
		topic.Prefix = c.cfg.Prefix
		topic.FullName = fmt.Sprintf("%s_%s_%s", topic.Prefix, c.cfg.Environment, topic.Name)

		if topic.IsFIFO && !strings.HasSuffix(topic.FullName, fifoSuffix) {
			topic.FullName += fifoSuffix
		}

		if _, err := c.provider.CreateTopic(ctx, topic); err != nil {
			return fmt.Errorf("failed to create topic %q: %w", topic.FullName, err)
		}
	}

	for _, queue := range c.cfg.Queues {
		queue.FullName = fmt.Sprintf("%s_%s_%s", c.cfg.Prefix, c.cfg.Environment, queue.Name)

		if queue.IsFIFO && !strings.HasSuffix(queue.FullName, fifoSuffix) {
			queue.FullName += fifoSuffix
		}
		setQueueDefaults(&queue)

		for i := range queue.SNSTopics {
			t := &queue.SNSTopics[i]
			if t.Prefix == "" {
				t.Prefix = c.cfg.Prefix
			}
			t.FullName = fmt.Sprintf("%s_%s_%s", t.Prefix, c.cfg.Environment, t.Name)
		}

		if _, err := c.provider.CreateQueue(ctx, queue); err != nil {
			return fmt.Errorf("failed to create queue %q: %w", queue.FullName, err)
		}
	}

	return nil
}

func setQueueDefaults(q *Queue) {
	mr := q.MessageRetention
	switch {
	case mr <= 0:
		q.MessageRetention = messageRetentionDefault
	case mr < 60:
		q.MessageRetention = messageRetentionMin
	default:
		q.MessageRetention = min(mr, messageRetentionMax)
	}

	if q.VisibilityTimeout <= 0 {
		q.VisibilityTimeout = visibilityTimeoutDefault
	} else {
		q.VisibilityTimeout = min(q.VisibilityTimeout, visibilityTimeoutMax)
	}

	dlq := &q.DeadLetterQueue

	if dlq.Enabled {
		if dlq.MaxReceiveCount <= 0 {
			dlq.MaxReceiveCount = dlqMaxReceiveCountDefault
		}
		if dlq.Suffix == "" {
			dlq.Suffix = dlqSuffixDefault
		}
	}
}
