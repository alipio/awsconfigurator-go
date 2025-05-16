package configurator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

var _ AwsProvider = (*ProviderImpl)(nil)

// NOTE: ProviderImpl is not thread safe and should not be used concurrently.
type ProviderImpl struct {
	sqs *sqs.Client
	sns *sns.Client

	accountID string
	region    string
	// Simple cache to avoid hitting AWS repeatedly.
	topicCache map[string]string
}

type PolicyDocument struct {
	Version   string
	Statement []PolicyStatement
}

type PolicyStatement struct {
	Effect    string
	Action    string
	Principal map[string]string `json:",omitempty"`
	Resource  *string           `json:",omitempty"`
	Condition PolicyCondition   `json:",omitempty"`
}

type PolicyCondition map[string]map[string][]string

func NewAwsProvider(cfg *Config) (AwsProvider, error) {
	if cfg.Region == "" {
		return nil, &InvalidConfigError{Message: "region is missing"}
	}

	if cfg.AccountID == "" {
		return nil, &InvalidConfigError{Message: "account ID is missing"}
	}

	awsCfg, err := awscfg.LoadDefaultConfig(context.Background(), awscfg.WithRegion(cfg.Region))

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	return &ProviderImpl{
		sqs:        sqs.NewFromConfig(awsCfg),
		sns:        sns.NewFromConfig(awsCfg),
		accountID:  cfg.AccountID,
		region:     cfg.Region,
		topicCache: make(map[string]string),
	}, nil
}

func (p *ProviderImpl) CreateTopic(ctx context.Context, topic Topic) (string, error) {
	if arn, ok := p.topicCache[topic.FullName]; ok {
		return arn, nil
	}

	input := &sns.CreateTopicInput{Name: aws.String(topic.FullName)}

	if topic.IsFIFO {
		input.Attributes = map[string]string{
			"FifoTopic":                 "true",
			"ContentBasedDeduplication": "true",
		}
	}

	output, err := p.sns.CreateTopic(ctx, input)

	if err != nil {
		return "", err
	}

	p.topicCache[topic.FullName] = *output.TopicArn

	return *output.TopicArn, nil
}

func (p *ProviderImpl) CreateQueue(ctx context.Context, queue Queue) (string, error) {
	input := &sqs.CreateQueueInput{QueueName: aws.String(queue.FullName)}

	attrs, err := p.buildQueueAttributes(queue)
	if err != nil {
		return "", err
	}

	dlqConf := queue.DeadLetterQueue

	if dlqConf.Enabled {
		e := p.createAndConfigureDLQ(ctx, queue, attrs, dlqConf)
		if e != nil {
			return "", fmt.Errorf("failed to create DLQ for queue %q: %w", queue.FullName, e)
		}
	}

	input.Attributes = attrs

	output, err := p.sqs.CreateQueue(ctx, input)
	if err != nil {
		return "", err
	}
	queue.URL = *output.QueueUrl

	topics := queue.SNSTopics
	if len(topics) > 0 {
		if e := p.Subscribe(ctx, queue, topics); e != nil {
			return "", fmt.Errorf("failed to process subscriptions for queue %q: %w", queue.FullName, e)
		}
	}

	return queue.URL, nil
}

func (p *ProviderImpl) buildQueueAttributes(queue Queue) (map[string]string, error) {
	attrs := map[string]string{
		"MessageRetentionPeriod":        strconv.Itoa(queue.MessageRetention),
		"VisibilityTimeout":             strconv.Itoa(queue.VisibilityTimeout),
		"ReceiveMessageWaitTimeSeconds": "3", // waits up to 3 seconds for a message before returning empty.
	}

	if queue.IsFIFO {
		attrs["FifoQueue"] = "true"
		attrs["ContentBasedDeduplication"] = "true"
	}

	return attrs, nil
}

func (p *ProviderImpl) createAndConfigureDLQ(ctx context.Context, queue Queue, queueAttrs map[string]string, dlqConf DLQConfig) error {
	nameNoSuffix := strings.TrimSuffix(queue.FullName, fifoSuffix)
	fullName := fmt.Sprintf("%s%s", nameNoSuffix, dlqConf.Suffix)

	m := map[string]string{
		"MessageRetentionPeriod": strconv.Itoa(messageRetentionMax),
		"VisibilityTimeout":      strconv.Itoa(visibilityTimeoutDefault),
	}
	// DLQ itself must also be a FIFO queue to maintain message order.
	if queue.IsFIFO {
		fullName += fifoSuffix

		m["FifoQueue"] = "true"
		m["ContentBasedDeduplication"] = "true"
	}

	input := &sqs.CreateQueueInput{QueueName: aws.String(fullName), Attributes: m}
	_, err := p.sqs.CreateQueue(ctx, input)
	if err != nil {
		return err
	}

	redrivePolicy := map[string]string{
		"deadLetterTargetArn": fmt.Sprintf("arn:aws:sqs:%s:%s:%s", p.region, p.accountID, fullName),
		"maxReceiveCount":     strconv.Itoa(dlqConf.MaxReceiveCount),
	}
	bytes, err := json.Marshal(redrivePolicy)
	if err != nil {
		return fmt.Errorf("failed to marshal redrive policy: %w", err)
	}

	queueAttrs["RedrivePolicy"] = string(bytes)

	return nil
}

func (p *ProviderImpl) Subscribe(ctx context.Context, queue Queue, topics []Topic) error {
	if len(topics) == 0 {
		return nil
	}
	topicArns := make([]string, len(topics))
	queueArn := fmt.Sprintf("arn:aws:sqs:%s:%s:%s", p.region, p.accountID, queue.FullName)

	for i, t := range topics {
		topicArn, err := p.CreateTopic(ctx, t)
		if err != nil {
			return fmt.Errorf("failed to create topic %q: %w", t.FullName, err)
		}

		si := &sns.SubscribeInput{
			Protocol:              aws.String("sqs"),
			TopicArn:              aws.String(topicArn),
			Endpoint:              aws.String(queueArn),
			Attributes:            map[string]string{"RawMessageDelivery": "true"},
			ReturnSubscriptionArn: true,
		}
		_, err = p.sns.Subscribe(ctx, si)
		if err != nil {
			return fmt.Errorf("failed to subscribe queue %q to topic %q: %w", queue.FullName, t.FullName, err)
		}

		topicArns[i] = topicArn
	}

	err := p.attachAccessPolicy(ctx, queue.URL, queueArn, topicArns)
	if err != nil {
		return fmt.Errorf("failed to attach access policy to queue %q: %w", queue.FullName, err)
	}

	return nil
}

func (p *ProviderImpl) attachAccessPolicy(ctx context.Context, queueURL, queueArn string, topicArns []string) error {
	policyDoc := PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{{
			Effect:    "Allow",
			Principal: map[string]string{"Service": "sns.amazonaws.com"},
			Action:    "sqs:SendMessage",
			Resource:  aws.String(queueArn),
			Condition: PolicyCondition{
				"ArnLike": map[string][]string{"aws:SourceArn": topicArns},
			},
		}},
	}
	bytes, err := json.Marshal(policyDoc)
	if err != nil {
		return err
	}

	_, err = p.sqs.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
		Attributes: map[string]string{string(types.QueueAttributeNamePolicy): string(bytes)},
		QueueUrl:   aws.String(queueURL),
	})
	if err != nil {
		return err
	}

	return nil
}
