package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const defaultSqsBatchSize = 10
const defaultSqsMessageRecords = 10
const maxSqsBatchSize = 10
const maxSqsMessageRecords = 100

// partition subcommand
var partitionCmd = &cobra.Command{
	Use:   "partition",
	Short: "Initiate partitioning of selected cloudfront log files",
	Long: `Fetch filenames from a cloudfront s3 log bucket of a given environment
and send them to a aws sqs-queue for partitioning.

Examples:
	cloudfront-logs partition --profile swisstopo-bgdi-dev --bucket swisstopo-bgdi-dev-cloudfront-logs-v2 \
	--prefix sys-data.dev.bgdi.ch --timestamp-from 2025-04-25 --timestamp-to 2025-04-25 --verbose --dry-run

	cloudfront-logs partition --profile swisstopo-bgdi-dev --bucket swisstopo-bgdi-dev-cloudfront-logs-v2 \
	--verbose --dry-run
`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {

		timeStart := time.Now()
		partitionConf, err := newPartionConfig(cmd)
		if err != nil {
			return err
		}

		printStart(partitionConf, timeStart)

		// Collect metrics
		ch := make(chan metrics)
		var m metrics
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			m = collectMetrics(ch, timeStart)
			wg.Done()
		}()

		// Do the partitioning work
		err = runPartition(partitionConf, ch)
		if err != nil {
			return err
		}

		close(ch)
		wg.Wait()

		m.Durations.Total = time.Since(timeStart)
		printEnd(m, partitionConf.Verbose)

		return nil
	},
}

var (
	timestampPattern = `^.*(?P<dateHour>\d\d\d\d\-\d\d\-\d\d\-\d\d).*$`
	timestampRe      = regexp.MustCompile(timestampPattern)
)

//-----------------------------------------------------------------------------

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.AddCommand(partitionCmd)

	partitionCmd.Flags().StringP("prefix", "p", "", "Prefix of s3 files we want to process.")
	partitionCmd.Flags().StringP("timestamp-from", "s", "", `Source-files with lower time-stamps are skipped.
	Format: yyyy[-mm[-dd]-[hh]]. Examples: 2025-04-23-01, 2025-03-10, 2025-02, 2024`)
	partitionCmd.Flags().StringP("timestamp-to", "t", "", `Source-files with higher OR EQUAL time-stamps are skipped.
	Format: yyyy[-mm[-dd]-[hh]]. Examples: 2025-05-01-13, 2025-04-01, 2025-02, 2025`)
	partitionCmd.Flags().Int64("sqs-message-records", defaultSqsMessageRecords, `Number of s3 records added to one
	SQS message. (max 100)`)
	partitionCmd.Flags().Int64("sqs-batch-size", defaultSqsBatchSize, `Number of SQS messages published in one SQS batch.
	(max 10)`)
	partitionCmd.Flags().BoolP("dry-run", "d", false, "Fetch files without publishing to queue.")
}

//-----------------------------------------------------------------------------

func runPartition(partitionConfig partitionConfig, ch chan metrics) error {
	context := context.Background()

	awsConfig, err := config.LoadDefaultConfig(
		context,
		config.WithRegion(partitionConfig.AwsRegion),
		config.WithSharedConfigProfile(partitionConfig.AwsProfile),
	)
	if err != nil {
		return err
	}

	s3Basics := NewS3Basics(context, awsConfig)
	sqsBasics := NewSqsBasics(context, awsConfig)

	// Get 3s list paginator
	paginator := s3Basics.GetListObjectsPaginator(partitionConfig)

	for paginator.HasMorePages() {
		m := metrics{}
		m.Counters.Pages++
		ts := time.Now()
		page, e := paginator.NextPage(context)
		if e != nil {
			return e
		}
		m.Counters.Files.Fetched += len(page.Contents)
		m.Durations.FetchKeys += time.Since(ts)

		ts = time.Now()
		keys, e := getKeysToPartition(page.Contents, &partitionConfig, &m)
		if e != nil {
			return e
		}
		m.Counters.Files.Skipped += len(page.Contents) - len(keys)
		m.Durations.GetKeysToPartition += time.Since(ts)

		ts = time.Now()
		if !partitionConfig.DryRun {
			err = sqsBasics.PublishKeys(partitionConfig, keys, &m)
			if err != nil {
				return err
			}
		}
		m.Durations.SendSqsPayload += time.Since(ts)
		m.Counters.Files.Partitioned += len(keys)

		ch <- m
	}

	return nil
}

func getKeysToPartition(contents []types.Object, conf *partitionConfig, metrics *metrics) ([]string, error) {
	keys := []string{}

	for _, obj := range contents {
		key := *obj.Key

		matches := timestampRe.FindStringSubmatch(key)
		switch {
		case matches != nil: // Timestamp found
			timestamp, err := parseTimestamp(matches[1])
			if err != nil {
				return []string{}, err
			}

			timeFrom := conf.TimeFrom
			timeTo := conf.TimeTo
			if (timeFrom.IsZero() || timeFrom.Equal(timestamp) || timeFrom.Before(timestamp)) &&
				(timeTo.IsZero() || timeTo.After(timestamp)) {
				keys = append(keys, key)
			}
		case strings.HasSuffix(key, "/"): // Prefix
			metrics.Counters.Prefixes++
		default: // Error
			return []string{}, fmt.Errorf("invalid key name: %s", key)
		}
	}

	return keys, nil
}

func parseTimestamp(tsString string) (time.Time, error) {
	layouts := []string{
		"2006-01-02-15",
		"2006-01-02",
		"2006-01",
		"2006",
	}
	var ts time.Time
	var err error

	for _, layout := range layouts {
		ts, err = time.Parse(layout, tsString)
		if err == nil {
			return ts, nil
		}
	}
	return ts, err
}
