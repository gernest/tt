package tseries

import (
	"time"

	"github.com/urfave/cli"
)

type Config struct {
	Path                     string
	MinBlockDuration         time.Duration
	MaxBlockDuration         time.Duration
	MaxBlockChunkSegmentSize int64
	WALSegmentSize           int64
	RetentionDuration        time.Duration
	MaxBytes                 int64
	NoLockfile               bool
	AllowOverlappingBlocks   bool
	WALCompression           string
}

func (c *Config) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "ts-path",
			Usage:  "Base path for metrics storage.",
			EnvVar: "TT_TIMESERIES_STORE_PATH",
			Value:  "data/",
		},
		cli.DurationFlag{
			Name:  "ts-min-block-duration",
			Usage: "Minimum duration of a data block before being persisted. For use in testing.",
			Value: 2 * time.Hour,
		},
		cli.DurationFlag{
			Name:  "ts-max-block-duration",
			Usage: "Maximum duration compacted blocks may span.",
		},
		cli.StringFlag{
			Name:  "ts-max-block-chunk-segment-size",
			Usage: "The maximum size for a single chunk segment in a block.",
		},
		cli.StringFlag{
			Name:  "ts-wal-segment-size",
			Usage: "Size at which to split the tsdb WAL segment files.",
		},
		cli.StringFlag{
			Name:  "ts-retention-size",
			Usage: "Maximum number of bytes that can be stored for blocks",
		},
		cli.BoolFlag{
			Name: "ts-no-lockfile",
		},
		cli.BoolFlag{
			Name:  "ts-allow-overlapping-blocks",
			Usage: "Allow overlapping blocks, which in turn enables vertical compaction and vertical query merge.",
		},
		cli.BoolTFlag{
			Name:  "ts-wal-compression",
			Usage: "Compress the tsdb WAL",
		},
	}
}
