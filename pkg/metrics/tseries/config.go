package tseries

import (
	"errors"
	"time"

	"github.com/alecthomas/units"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/urfave/cli"
)

const days = 24 * time.Hour

type Config struct {
	ENabled                  bool             `json:",omitempty"`
	Path                     string           `json:",omitempty"`
	MinBlockDuration         time.Duration    `json:",omitempty"`
	MaxBlockDuration         time.Duration    `json:",omitempty"`
	MaxBlockChunkSegmentSize units.Base2Bytes `json:",omitempty"`
	WALSegmentSize           units.Base2Bytes `json:",omitempty"`
	RetentionDuration        time.Duration    `json:",omitempty"`
	MaxBytes                 units.Base2Bytes `json:",omitempty"`
	NoLockfile               bool             `json:",omitempty"`
	AllowOverlappingBlocks   bool             `json:",omitempty"`
	WALCompression           bool             `json:",omitempty"`
}

func (c *Config) Parse(ctx *cli.Context) (err error) {
	c.Path = ctx.GlobalString("ts-path")
	c.MinBlockDuration = ctx.GlobalDuration("ts-min-block-duration")
	c.MaxBlockDuration = ctx.GlobalDuration("ts-max-block-duration")
	c.MaxBlockChunkSegmentSize, err = units.ParseBase2Bytes(
		ctx.GlobalString("ts-max-block-chunk-segment-size"),
	)
	c.WALSegmentSize, err = units.ParseBase2Bytes(
		ctx.GlobalString("ts-wal-segment-size"),
	)
	c.RetentionDuration = ctx.GlobalDuration("ts-retention-time")
	c.MaxBytes, err = units.ParseBase2Bytes(
		ctx.GlobalString("ts-retention-size"),
	)
	c.NoLockfile = ctx.GlobalBool("ts-no-lockfile")
	c.AllowOverlappingBlocks = ctx.GlobalBool("ts-allow-overlapping-blocks")
	c.WALCompression = ctx.GlobalBool("ts-wal-compression")
	return c.configure()
}

func (c *Config) Options() *tsdb.Options {
	return &tsdb.Options{
		WALSegmentSize:           int(c.WALSegmentSize),
		MaxBlockChunkSegmentSize: int64(c.MaxBlockChunkSegmentSize),
		RetentionDuration:        c.RetentionDuration.Milliseconds(),
		MaxBytes:                 int64(c.MaxBytes),
		NoLockfile:               c.NoLockfile,
		AllowOverlappingBlocks:   c.AllowOverlappingBlocks,
		WALCompression:           c.WALCompression,
		MinBlockDuration:         c.MinBlockDuration.Milliseconds(),
		MaxBlockDuration:         c.MaxBlockDuration.Milliseconds(),
	}
}

func (c *Config) configure() error {
	if c.RetentionDuration == 0 && c.MaxBytes == 0 {
		c.RetentionDuration = 15 * days
	}
	if c.MaxBlockDuration == 0 {
		max := 31 * days
		if c.RetentionDuration != 0 && c.RetentionDuration/10 < max {
			max = c.RetentionDuration / 10
		}
		c.MaxBlockDuration = max
	}
	if c.WALSegmentSize != 0 {
		if c.WALSegmentSize < 10*1024*1024 || c.WALSegmentSize > 256*1024*1024 {
			return errors.New("flag 'ts-wal-segment-size' must be set between 10MB and 256MB")
		}
	}
	if c.MaxBlockChunkSegmentSize != 0 {
		if c.MaxBlockChunkSegmentSize < 1024*1024 {
			return errors.New("flag 'ts-max-block-chunk-segment-size' must be set over 1MB")
		}
	}
	return nil
}

func (Config) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "ts-path",
			Usage:  "Base path for metrics storage.",
			EnvVar: "TT_TIMESERIES_STORE_PATH",
			Value:  "data/",
		},
		cli.DurationFlag{
			Name:   "ts-min-block-duration",
			EnvVar: "TT_TIMESERIES_MIN_BLOCK_DURATION",
			Usage:  "Minimum duration of a data block before being persisted. For use in testing.",
			Value:  2 * time.Hour,
		},
		cli.DurationFlag{
			Name:   "ts-max-block-duration",
			EnvVar: "TT_TIMESERIES_MAX_BLOCK_DURATION",
			Usage:  "Maximum duration compacted blocks may span.",
		},
		cli.StringFlag{
			Name:   "ts-max-block-chunk-segment-size",
			EnvVar: "TT_TIMESERIES_MAX_BLOCK_CHUNK_SEGMENT_SIZE",
			Usage:  "The maximum size for a single chunk segment in a block.",
		},
		cli.StringFlag{
			Name:   "ts-wal-segment-size",
			EnvVar: "TT_TIMESERIES_WAL_SEGMENT_SIZE",
			Usage:  "Size at which to split the tsdb WAL segment files.",
		},
		cli.DurationFlag{
			Name:   "ts-retention-time",
			EnvVar: "TT_TIMESERIES_RETENTION_TIME",
			Usage:  "ow long to retain samples in storage",
		},
		cli.StringFlag{
			Name:   "ts-retention-size",
			EnvVar: "TT_TIMESERIES_RETENTION_SIZE",
			Usage:  "Maximum number of bytes that can be stored for blocks",
		},
		cli.BoolFlag{
			Name:   "ts-no-lockfile",
			EnvVar: "TT_TIMESERIES_NO_LOCK_FILE",
		},
		cli.BoolFlag{
			Name:   "ts-allow-overlapping-blocks",
			EnvVar: "TT_TIMESERIES_ALLOW_OVERLAPING_BLOCK",
			Usage:  "Allow overlapping blocks, which in turn enables vertical compaction and vertical query merge.",
		},
		cli.BoolTFlag{
			Name:   "ts-wal-compression",
			EnvVar: "TT_TIMESERIES_WAL_COMPRESSION",
			Usage:  "Compress the tsdb WAL",
		},
	}
}
