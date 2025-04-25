package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Struct which holds all metrics used for status updates
type metrics struct {
	Counters struct {
		Files struct {
			Fetched     int
			Partitioned int
			Skipped     int
		}
		Pages    int
		Prefixes int
	}
	Durations struct {
		FetchKeys          time.Duration
		GetKeysToPartition time.Duration
		BuildSqsPayload    time.Duration
		SendSqsPayload     time.Duration
		Total              time.Duration
	}
	Timestamps struct {
		Start time.Time
	}
}

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
	Run: func(cmd *cobra.Command, _ []string) {
		metrics := &metrics{}
		metrics.Timestamps.Start = time.Now()

		partitionConf, err := newPartionConfig(cmd)
		if err != nil {
			fmt.Println(err)
			return
		}

		printStart(partitionConf, metrics)

		// Start a goroutine to printout status updates.
		ticker := time.NewTicker(time.Second)
		go printProgress(ticker, metrics)

		// Do the partitioning work
		err = runPartition(partitionConf, metrics)
		if err != nil {
			fmt.Println(err)
			return
		}

		ticker.Stop()
		metrics.Durations.Total = time.Since(metrics.Timestamps.Start)
		printEnd(metrics, partitionConf.Verbose)
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

func runPartition(partitionConfig partitionConfig, metrics *metrics) error {
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
		metrics.Counters.Pages++
		ts := time.Now()
		page, e := paginator.NextPage(context)
		if e != nil {
			return e
		}
		metrics.Counters.Files.Fetched += len(page.Contents)
		metrics.Durations.FetchKeys += time.Since(ts)

		ts = time.Now()
		keys, e := getKeysToPartition(page.Contents, &partitionConfig, metrics)
		if e != nil {
			return e
		}
		metrics.Counters.Files.Skipped += len(page.Contents) - len(keys)
		metrics.Durations.GetKeysToPartition += time.Since(ts)

		ts = time.Now()
		if !partitionConfig.DryRun {
			err = sqsBasics.PublishKeys(partitionConfig, keys, metrics)
			if err != nil {
				return err
			}
		}
		metrics.Durations.SendSqsPayload += time.Since(ts)
		metrics.Counters.Files.Partitioned += len(keys)
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
