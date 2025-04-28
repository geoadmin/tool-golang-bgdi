// https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/s3/actions/bucket_basics.go

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type S3Event struct {
	S3 struct {
		Bucket struct {
			Arn  string `json:"Arn,omitempty"`
			Name string `json:"Name,omitempty"`
		} `json:"Bucket"`
		Object struct {
			Key string `json:"Key,omitempty"`
		} `json:"Object,omitempty"`
	} `json:"S3,omitempty"`
}

type SQSMessageBody struct {
	Records []S3Event `json:"Records"`
}

type SqsBasics struct {
	Client  *sqs.Client
	Context context.Context
}

func NewSqsBasics(ctx context.Context, awsConfig aws.Config) *SqsBasics {
	return &SqsBasics{
		Context: ctx,
		Client:  sqs.NewFromConfig(awsConfig),
	}
}

func (basics SqsBasics) PublishKeys(cfg partitionConfig, keys []string, metrics *metrics) error {
	// Keys is a list of up to 1000 s3 keys (filenames).
	for batchChunk := range slices.Chunk(keys, (cfg.SqsBatchSize * cfg.SqsMessageRecords)) {
		timestamp := time.Now()

		// SQS Batch Message content
		params := sqs.SendMessageBatchInput{
			QueueUrl: &cfg.SqsQueueURL,
		}

		// Build the content of a SQS message and add it to the batch.
		// Note: SQS max batch size = 10!
		for messageChunk := range slices.Chunk(batchChunk, cfg.SqsMessageRecords) {
			// Create a SQS message body and id
			body := SQSMessageBody{}
			id := uuid.New().String()

			for _, item := range messageChunk {
				// Create one s3 event (record) for each key and add it to the sqs message body
				record := S3Event{}
				record.S3.Bucket.Name = cfg.S3Bucket
				record.S3.Bucket.Arn = fmt.Sprintf("arn:aws:s3:::%s", cfg.S3Bucket)
				record.S3.Object.Key = item

				body.Records = append(body.Records, record)
			}

			// Convert the body to a json string
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return err
			}
			jsonBodyString := string(jsonBody)

			// Create the SQS message (batch entry) and add it to the batch
			entry := types.SendMessageBatchRequestEntry{
				Id:          &id,
				MessageBody: &jsonBodyString,
			}
			params.Entries = append(params.Entries, entry)
		}

		// Update metrics
		metrics.Durations.BuildSqsPayload += time.Since(timestamp)
		timestamp = time.Now()

		// Send batch
		output, err := basics.Client.SendMessageBatch(basics.Context, &params)
		metrics.Durations.SendSqsPayload += time.Since(timestamp)

		if err != nil {
			return err
		}

		// Batches may return successful even if some of the messages in the batch
		// fail. Thus we check for individual failures here.
		if len(output.Failed) > 0 {
			return fmt.Errorf("SQS publishing error: %+v", output.Failed)
		}
	}

	return nil
}
