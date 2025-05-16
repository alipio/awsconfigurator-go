# awsconfigurator-go

![Supported Go Versions](https://img.shields.io/badge/Go-1.24-lightgrey.svg)
![License](https://img.shields.io/badge/license-MIT-blue)

Simple YAML-based configurator for AWS SNS/SQS services. This is a Go port of the [ex_aws_configurator](https://github.com/petlove/ex_aws_configurator/tree/main) tool that is used by many projects across Petlove to ensure that queues/topics are created with a consistent naming convention (more information below).

## Install the binary


    $ go install github.com/alipio/awsconfigurator-go/cmd/awsconfigurator@latest


## CLI usage

Run the following command (Make sure `$GOPATH/bin` is in your PATH)

    $ awsconfigurator -config=<path_to_config>

## Configuring AWS integration

This tool uses a thin wrapper around the [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) to access AWS, so you just have to set the following environment variables to get started: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and `AWS_SESSION_TOKEN`

## Configuring the AWS Configurator

Take a look at the sample `config-reference.yaml` for a complete reference. For now, a simple example will suffice:

```yaml
region: ${AWS_REGION}
prefix: raven
environment: staging
account_id: ${AWS_ACCOUNT_ID}

sns_topics:
  - name: simple_topic
  - name: simple_fifo_topic
    fifo: true

queues:
  - name: simple_queue
    fifo: true
    visibility_timeout_seconds: 60

  - name: another_queue
    sns_topics:
      - name: simple_topic
    dead_letter_queue:
      enabled: true
      max_receive_count: 3
    message_retention_seconds: 172800 # 2 days
    visibility_timeout_seconds: 300   # 5 minutes
```

You can embed environment variables in the config file and they will be expaded automatically.

The configuration format is pretty self-explanatory and almost identical to the one adopted by the other tools (`turtle`, `ex_aws_configurator` etc.).

Given the configuration above, the following steps are run by the tool in order:

1. It will create two topics: a regular one called **raven_staging_simple_topic** and a FIFO one called **raven_staging_simple_fifo_topic.fifo** (note the required `.fifo` suffix).
2. Next, it will create a FIFO queue called **raven_staging_simple_queue.fifo** with a visibility timeout of 60 seconds and no dead-letter queue.
3. Next, a regular queue called **raven_staging_another_queue** with the message retention set to 2 days and visibility timeout set to 5 minutes. In this case, the queue will have a dead-letter queue called **raven_staging_another_queue_failures** (the default DLQ suffix), configured with a redrive policy of no more than 3 retries maximum. The message retention period of this queue will be set to its maximum value of 1,209,600 seconds (14 days).
4. And finally, it will subscribe the latter queue to the topic **raven_staging_simple_topic**, setting up the queue's access policy accordingly.

All these resources will be created on the specified region according to the value of the environment variable `AWS_REGION`.

The name of those resources always follow the same pattern: `<prefix>_<environment>_<bare-name>` or `<prefix>_<environment>_<bare-name>.fifo` to keep it consistent and predictable.

Please take your time to inspect the tables below to see which options are required, which are optional and their default values:

**General options**

| Name          | Default | Required | Description                                                        |
|---------------|---------|----------|--------------------------------------------------------------------|
| `region`      | `""`    | yes      | AWS region                                                         |
| `prefix`      | `""`    | yes      | Used as prefix to compose topic/queue name                         |
| `environment` | `""`    | yes      | Target environment, this is used to compose topic/queue name       |
| `account_id`  | `""`    | yes      | Your AWS account id, this is used to calculate the topic/queue ARN |
| `sns_topics`  | `[]`    | no       | List of topic configurations                                       |
| `queues`      | `[]`    | no       | List of queues configurations                                      |

**Topic options**

| Name     | Default       | Required | Description                                                  |
|----------|---------------|----------|--------------------------------------------------------------|
| `name`   | `""`          | yes      | Bare name of the topic                                       |
| `prefix` | root `prefix` | no       | Only used so a queue can subscribe to topics from other apps |
| `fifo`   | false         | no       | Flag to specify if topic is FIFO                             |

**Queue options**

| Name                         | Default             | Required | Description                                                             |
|------------------------------|---------------------|----------|-------------------------------------------|
| `name`                       | `""`                | yes      | Bare name of the queue                    |
| `sns_topics`                 | `[]`                | no       | List of topics to subscribe this queue to |
| `dead_letter_queue`          | none                | no       | DLQ configuration (see below)             |
| `message_retention_seconds`  | 1,209,600 (14 days) | no       | Message retention period, in seconds      |
| `visibility_timeout_seconds` | 60                  | no       | The visibility timeout, in seconds        |
| `fifo`                       | false               | no       | Flag to specify if the queue is FIFO      |


**Dead-letter queue options**

| Name                | Default   | Required | Description                                                      |
|---------------------|-----------|----------|------------------------------------------------------------------|
| `enabled`           | false     | no       | Whether the DLQ configuration is enabled or not                  |
| `max_receive_count` | 7         | no       | Max. number of receives before a message is sent to the DLQ      |
| `suffix`            | _failures | no       | Suffix to append to queue/topic full name to form the DLQ's name |

If the required options are supplied but the list of queues and topics are empty, the program does nothing. In fact, you only need the bare name to create a topic/queue! All other options are set with their corresponding sensible defaults.

**Note**: If any error occurs, the program just exits with an error. This is done by design to stop the CI build, because it does not make sense to continue and deploy an application with missing dependencies.

If you prefer to have more control over the creation of the **topics** and **queues**, you should do the subscription step manually.

> [!IMPORTANT]
> All FIFO queues/topics are created with the attribute `ContentBasedDeduplication` set to `true` by default. This option cannot be customized for now.

## Contributing

Feel free to contribute, issues and pull requests are more than welcome!
