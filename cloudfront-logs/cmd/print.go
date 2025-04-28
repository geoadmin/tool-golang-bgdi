package cmd

import (
	"fmt"
	"strings"
	"time"
)

const numberOfSeparatorChars = 80

func printStart(conf partitionConfig, timeStart time.Time) {
	lineSeparator := strings.Repeat("-", numberOfSeparatorChars)
	fmt.Println(lineSeparator)
	fmt.Printf("%s - Cloudfront logs partitioning starting\n\n", timeStart.Format("2006-01-02 15:04:05"))
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

func printProgress(metrics *metrics) {
	fmt.Print("\033[s") // save the cursor position

	fmt.Print("\033[G\033[K") // move the cursor left and clear the line
	fmt.Printf("%s - %3d prefixes, %5d pages, %8d files-fetched, %8d files-partitioned, %8d files-skipped, Duration: %s",
		time.Now().Format("2006-01-02 15:04:05"),
		len(metrics.Prefixes),
		metrics.Counters.Pages,
		metrics.Counters.Files.Fetched,
		metrics.Counters.Files.Partitioned,
		metrics.Counters.Files.Skipped,
		time.Since(metrics.Timestamps.Start).Round(time.Millisecond),
	)
}

func printEnd(metrics metrics, verbose bool) {
	lineSeparator := strings.Repeat("-", numberOfSeparatorChars)

	fmt.Println("")
	fmt.Println(lineSeparator)
	fmt.Printf("%s - Partitioning done in %s",
		time.Now().Format("2006-01-02 15:04:05"),
		metrics.Durations.Total.Round(time.Millisecond),
	)

	if verbose {
		fmt.Printf(`
	Counters
		Prefixes                   : %8d
		Pages                      : %8d
		Files-fetched              : %8d
		Files-partitioned          : %8d
		Files-skipped              : %8d

	Durations:
		Fetch keys                 : %8s
		Get keys to be partitioned : %8s
		Build SQS payload          : %8s
		Total                      : %8s

`,
			len(metrics.Prefixes),
			metrics.Counters.Pages,
			metrics.Counters.Files.Fetched,
			metrics.Counters.Files.Partitioned,
			metrics.Counters.Files.Skipped,
			metrics.Durations.Total.Round(time.Millisecond),
			metrics.Durations.GetKeysToPartition.Round(time.Millisecond),
			metrics.Durations.BuildSqsPayload.Round(time.Millisecond),
			metrics.Durations.Total.Round(time.Millisecond),
		)
	}
	fmt.Println("	Prefixes")
	for _, prefix := range metrics.Prefixes {
		fmt.Printf("		%s\n", prefix)
	}

	fmt.Println(lineSeparator)
}
