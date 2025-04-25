package cmd

import (
	"fmt"
	"strings"
	"time"
)

const numberOfSeparatorChars = 80

func printStart(conf partitionConfig, metrics *metrics) {
	lineSeparator := strings.Repeat("-", numberOfSeparatorChars)
	fmt.Println(lineSeparator)
	fmt.Printf("%s - Cloudfront logs partitioning starting\n\n", metrics.Timestamps.Start.Format("2006-01-02 15:04:05"))
	if conf.Verbose {
		fmt.Printf(`
Config:
    AWS-Profile        : %s
    AWS-Region         : %s
    S3-Bucket          : %s
    S3-Max-Keys        : %d
    S3-Object-Delimiter: %s
    S3-Prefix          : %s
    SQS-Queue-URL      : %s
    SQS-Batch-Size     : %d
    SQS-MessageRecords : %d
    Timestamp-From     : %s
    Timestamp-To       : %s

`,
			conf.AwsProfile,
			conf.AwsRegion,
			conf.S3Bucket,
			conf.S3MaxKeys,
			conf.S3ObjectDelimiter,
			conf.S3Prefix,
			conf.SqsQueueURL,
			conf.SqsBatchSize,
			conf.SqsMessageRecords,
			conf.TimeFrom.String(),
			conf.TimeTo.String(),
		)
	}
	fmt.Println(lineSeparator)
}

func printProgress(ticker *time.Ticker, metrics *metrics) {
	fmt.Print("\033[s") // save the cursor position

	for range ticker.C {
		// String built here to solve line-too-long linter problem when directly used in final print statement
		metricsString := fmt.Sprintf("prefixes: %3d, pages: %5d, files (fetched: %7d, partitioned %7d, skipped %7d)",
			metrics.Counters.Prefixes,
			metrics.Counters.Pages,
			metrics.Counters.Files.Fetched,
			metrics.Counters.Files.Partitioned,
			metrics.Counters.Files.Skipped,
		)

		fmt.Print("\033[G\033[K") // move the cursor left and clear the line
		fmt.Printf("%s - Partitioning running: %s processed in %s",
			time.Now().Format("2006-01-02 15:04:05"),
			metricsString,
			time.Since(metrics.Timestamps.Start).Round(time.Second).String(),
		)
	}
}

func printEnd(metrics *metrics, verbose bool) {
	lineSeparator := strings.Repeat("-", numberOfSeparatorChars)

	// String built here to solve line-too-long linter problem when directly used in final print statement
	metricsString := fmt.Sprintf("prefixes: %3d, pages: %5d, files (fetched: %7d, partitioned %7d, skipped %7d)",
		metrics.Counters.Prefixes,
		metrics.Counters.Pages,
		metrics.Counters.Files.Fetched,
		metrics.Counters.Files.Partitioned,
		metrics.Counters.Files.Skipped,
	)

	fmt.Println("")
	fmt.Println(lineSeparator)
	fmt.Printf("%s - Partitioning done:    %s processed in %s",
		time.Now().Format("2006-01-02 15:04:05"),
		metricsString,
		time.Since(metrics.Timestamps.Start).Round(time.Second).String(),
	)
	if verbose {
		fmt.Printf(`

        Durations:
        Fetch keys              : %s
        Get keys to be partition: %s
        Build SQS payload       : %s
        Total                   : %s

`,
			metrics.Durations.FetchKeys.Round(time.Millisecond),
			metrics.Durations.GetKeysToPartition.Round(time.Millisecond),
			metrics.Durations.BuildSqsPayload.Round(time.Millisecond),
			metrics.Durations.Total.Round(time.Millisecond),
		)
	}
	fmt.Println(lineSeparator)
}
