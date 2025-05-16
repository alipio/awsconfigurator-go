package configurator_test

import (
	"context"
	"testing"

	"github.com/alipio/awsconfigurator-go/configurator"
	"github.com/alipio/awsconfigurator-go/configurator/mocks"
	"github.com/stretchr/testify/require"
)

func TestConfigurator(t *testing.T) {
	configPath := "../testdata/config.yaml"
	cfg, err := configurator.LoadConfig(configPath)
	require.NoError(t, err)

	provider := new(mocks.AwsProvider)

	ctx := context.Background()

	t1 := configurator.Topic{Name: "simple_topic", Prefix: "raven", IsFIFO: false, FullName: "raven_staging_simple_topic"}
	t2 := configurator.Topic{Name: "simple_fifo_topic", Prefix: "raven", IsFIFO: true, FullName: "raven_staging_simple_fifo_topic.fifo"}
	provider.On("CreateTopic", ctx, t1).Return("topic1Arn", nil).Once()
	provider.On("CreateTopic", ctx, t2).Return("topic2Arn", nil).Once()

	topics := []configurator.Topic{t1}
	q := configurator.Queue{
		Name:      "test_queue",
		SNSTopics: topics,
		DeadLetterQueue: configurator.DLQConfig{
			Enabled:         true,
			MaxReceiveCount: 7,
			Suffix:          "_failures",
		},
		MessageRetention:  172800,
		VisibilityTimeout: 300,
		IsFIFO:            false,
		FullName:          "raven_staging_test_queue",
	}
	provider.On("CreateQueue", ctx, q).Return("testQueueURL", nil).Once()

	configurator := configurator.New(provider, cfg)

	e := configurator.Run(ctx)
	require.NoError(t, e)

	provider.AssertExpectations(t)
}
