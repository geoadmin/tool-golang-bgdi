package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type partitionConfig struct {
	AwsProfile        string
	AwsRegion         string
	S3Bucket          string
	S3Prefix          string
	S3ObjectDelimiter string
	S3MaxKeys         int32
	SqsQueueURL       string
	SqsMessageRecords int
	SqsBatchSize      int
	TimeFrom          time.Time
	TimeTo            time.Time
	DryRun            bool
	Verbose           bool
}

func newPartionConfig(cmd *cobra.Command) (partitionConfig, error) {
	conf := partitionConfig{}
	conf.AwsProfile = cmd.Flag("profile").Value.String()
	conf.AwsRegion = "eu-central-1"
	conf.S3Bucket = cmd.Flag("bucket").Value.String()
	conf.S3Prefix = cmd.Flag("prefix").Value.String()
	conf.S3MaxKeys = 0

	switch conf.AwsProfile {
	case "swisstopo-bgdi-dev":
		conf.SqsQueueURL = "https://sqs.eu-central-1.amazonaws.com/839910802816/cloudfront-logs-partitioning-queue-manual"
	case "swisstopo-bgdi":
		conf.SqsQueueURL = "https://sqs.eu-central-1.amazonaws.com/993448060988/cloudfront-logs-partitioning-queue-manual"
	default:
		return conf, fmt.Errorf("invalid aws-profile %s. See --help for allowed values", conf.AwsProfile)
	}

	if len(cmd.Flag("timestamp-from").Value.String()) > 0 {
		timeFrom, err := parseTimestamp(cmd.Flag("timestamp-from").Value.String())
		if err != nil {
			return conf, err
		}
		conf.TimeFrom = timeFrom
	}

	if len(cmd.Flag("timestamp-to").Value.String()) > 0 {
		timeTo, err := parseTimestamp(cmd.Flag("timestamp-to").Value.String())
		if err != nil {
			return conf, err
		}
		conf.TimeTo = timeTo
	}

	messageRecords, err := cmd.Flags().GetInt64("sqs-message-records")
	if err != nil {
		return conf, err
	}
	if messageRecords > maxSqsMessageRecords {
		return conf, fmt.Errorf("sqs batch size %d too big. Max sqs batch size=10", messageRecords)
	}
	conf.SqsMessageRecords = int(messageRecords)

	batchSize, err := cmd.Flags().GetInt64("sqs-batch-size")

	if err != nil {
		return conf, err
	}
	if batchSize > maxSqsBatchSize {
		return conf, fmt.Errorf("sqs batch size %d too big. Max sqs batch size=10", batchSize)
	}
	conf.SqsBatchSize = int(batchSize)

	dryRun, err := cmd.Flags().GetBool("dry-run")

	if err != nil {
		return conf, err
	}
	conf.DryRun = dryRun

	verbose, err := cmd.Flags().GetBool("verbose")

	if err != nil {
		return conf, err
	}
	conf.Verbose = verbose

	return conf, nil
}
