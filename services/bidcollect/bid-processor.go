package bidcollect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/flashbots/relayscan/common"
	"github.com/flashbots/relayscan/services/bidcollect/types"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Goals:
// 1. Dedup bids
// 2. Save bids to CSV
//   - One CSV for all bids
//   - One CSV for top bids only

type BidProcessorOpts struct {
	Log       *logrus.Entry
	UID       string
	OutDir    string
	OutputTSV bool
	RedisAddr string
}

type OutFiles struct {
	FAll *os.File
	FTop *os.File
}

type BidProcessor struct {
	opts *BidProcessorOpts
	log  *logrus.Entry

	outFiles     map[int64]*OutFiles // map[slot][bidUniqueKey]Bid
	outFilesLock sync.RWMutex

	bidCache     map[uint64]map[string]*types.CommonBid // map[slot][bidUniqueKey]Bid
	topBidCache  map[uint64]*types.CommonBid            // map[slot]Bid
	bidCacheLock sync.RWMutex

	csvSeparator  string
	csvFileEnding string

	redisClient *redis.Client
}

func NewBidProcessor(opts *BidProcessorOpts) (*BidProcessor, error) {
	c := &BidProcessor{
		log:         opts.Log,
		opts:        opts,
		outFiles:    make(map[int64]*OutFiles),
		bidCache:    make(map[uint64]map[string]*types.CommonBid),
		topBidCache: make(map[uint64]*types.CommonBid),
	}

	if opts.OutputTSV {
		c.csvSeparator = "\t"
		c.csvFileEnding = "tsv"
	} else {
		c.csvSeparator = ","
		c.csvFileEnding = "csv"
	}

	if opts.RedisAddr != "" {
		c.redisClient = redis.NewClient(&redis.Options{
			Addr:     opts.RedisAddr,
			Password: "", // no password set
			DB:       0,  // use default DB
		})

		// Make sure we can connect to redis to connect to redis
		if _, err := c.redisClient.Ping(context.Background()).Result(); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *BidProcessor) Start() {
	for {
		time.Sleep(30 * time.Second)
		c.housekeeping()
	}
}

func (c *BidProcessor) processBids(bids []*types.CommonBid) {
	c.bidCacheLock.Lock()
	defer c.bidCacheLock.Unlock()

	var isTopBid, isNewBid bool
	for _, bid := range bids {
		isNewBid, isTopBid = false, false
		if _, ok := c.bidCache[bid.Slot]; !ok {
			c.bidCache[bid.Slot] = make(map[string]*types.CommonBid)
		}

		// Check if bid is new top bid
		if topBid, ok := c.topBidCache[bid.Slot]; !ok {
			c.topBidCache[bid.Slot] = bid // first one for the slot
			isTopBid = true
		} else {
			// if current bid has higher value, use it as new top bid
			if bid.ValueAsBigInt().Cmp(topBid.ValueAsBigInt()) == 1 {
				c.topBidCache[bid.Slot] = bid
				isTopBid = true
			}
		}

		// process regular bids only once per unique key (slot+blockhash+parenthash+builderpubkey+value)
		if _, ok := c.bidCache[bid.Slot][bid.UniqueKey()]; !ok {
			// yet unknown bid, save it
			c.bidCache[bid.Slot][bid.UniqueKey()] = bid
			isNewBid = true
		}

		// Send to Redis
		if c.redisClient != nil {
			err := c.redisClient.Publish(context.Background(), "bidcollect/bids", bid.ToCSVLine(",")).Err()
			if err != nil {
				c.log.WithError(err).Error("failed to publish bid to redis")
			}
		}

		// Write to CSV
		c.writeBidToFile(bid, isNewBid, isTopBid)
	}
}

func (c *BidProcessor) writeBidToFile(bid *types.CommonBid, isNewBid, isTopBid bool) {
	fAll, fTop, err := c.getFiles(bid)
	if err != nil {
		c.log.WithError(err).Error("get get output file")
		return
	}
	if isNewBid {
		_, err = fmt.Fprint(fAll, bid.ToCSVLine(c.csvSeparator)+"\n")
		if err != nil {
			c.log.WithError(err).Error("couldn't write bid to file")
			return
		}
	}
	if isTopBid {
		_, err = fmt.Fprint(fTop, bid.ToCSVLine(c.csvSeparator)+"\n")
		if err != nil {
			c.log.WithError(err).Error("couldn't write bid to file")
			return
		}
	}
}

func (c *BidProcessor) getFiles(bid *types.CommonBid) (fAll, fTop *os.File, err error) {
	// hourlybucket
	sec := int64(types.BucketMinutes * 60)
	bucketTS := bid.ReceivedAtMs / 1000 / sec * sec // timestamp down-round to start of bucket
	t := time.Unix(bucketTS, 0).UTC()

	// files may already be opened
	c.outFilesLock.RLock()
	outFiles, outFilesOk := c.outFiles[bucketTS]
	c.outFilesLock.RUnlock()

	if outFilesOk {
		return outFiles.FAll, outFiles.FTop, nil
	}

	// Create output directory
	dir := filepath.Join(c.opts.OutDir, t.Format(time.DateOnly))
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	// Open ALL BIDS CSV
	fnAll := filepath.Join(dir, c.getFilename("all", bucketTS))
	fAll, err = os.OpenFile(fnAll, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, err
	}
	fi, err := fAll.Stat()
	if err != nil {
		c.log.WithError(err).Fatal("failed stat on output file")
	}
	if fi.Size() == 0 {
		_, err = fmt.Fprint(fAll, strings.Join(types.CommonBidCSVFields, c.csvSeparator)+"\n")
		if err != nil {
			c.log.WithError(err).Fatal("failed to write header to output file")
		}
	}

	// Open TOP BIDS CSV
	fnTop := filepath.Join(dir, c.getFilename("top", bucketTS))
	fTop, err = os.OpenFile(fnTop, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, err
	}
	fi, err = fTop.Stat()
	if err != nil {
		c.log.WithError(err).Fatal("failed stat on output file")
	}
	if fi.Size() == 0 {
		_, err = fmt.Fprint(fTop, strings.Join(types.CommonBidCSVFields, c.csvSeparator)+"\n")
		if err != nil {
			c.log.WithError(err).Fatal("failed to write header to output file")
		}
	}

	outFiles = &OutFiles{
		FAll: fAll,
		FTop: fTop,
	}
	c.outFilesLock.Lock()
	c.outFiles[bucketTS] = outFiles
	c.outFilesLock.Unlock()

	c.log.Infof("[bid-processor] created output file: %s", fnAll)
	c.log.Infof("[bid-processor] created output file: %s", fnTop)
	return fAll, fTop, nil
}

func (c *BidProcessor) getFilename(prefix string, timestamp int64) string {
	t := time.Unix(timestamp, 0).UTC()
	if prefix != "" {
		prefix += "_"
	}
	return fmt.Sprintf("%s%s_%s.%s", prefix, t.Format("2006-01-02_15-04"), c.opts.UID, c.csvFileEnding)
}

func (c *BidProcessor) housekeeping() {
	currentSlot := common.TimeToSlot(time.Now().UTC())
	maxSlotInCache := currentSlot - 3

	nDeleted := 0
	nBids := 0

	c.bidCacheLock.Lock()
	defer c.bidCacheLock.Unlock()
	for slot := range c.bidCache {
		if slot < maxSlotInCache {
			delete(c.bidCache, slot)
			nDeleted += 1
		} else {
			nBids += len(c.bidCache[slot])
		}
	}

	// Close and remove old files
	now := time.Now().UTC().Unix()
	filesBefore := len(c.outFiles)
	c.outFilesLock.Lock()
	for timestamp, outFiles := range c.outFiles {
		usageSec := types.BucketMinutes * 60 * 2
		if now-timestamp > int64(usageSec) { // remove all handles from 2x usage seconds ago
			c.log.Info("closing output files", timestamp)
			delete(c.outFiles, timestamp)
			_ = outFiles.FAll.Close()
			_ = outFiles.FTop.Close()
		}
	}
	nFiles := len(c.outFiles)
	filesClosed := len(c.outFiles) - filesBefore
	c.outFilesLock.Unlock()

	c.log.Infof("[bid-processor] cleanupBids - deleted slots: %d / total slots: %d / total bids: %d / files closed: %d, current: %d / memUsedMB: %d", nDeleted, len(c.bidCache), nBids, filesClosed, nFiles, common.GetMemMB())
}
