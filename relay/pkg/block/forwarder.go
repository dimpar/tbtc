package block

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ipfs/go-log"
	"github.com/keep-network/tbtc/relay/pkg/btc"
	"github.com/keep-network/tbtc/relay/pkg/chain"
)

const (
	// Size of the headers queue.
	headersQueueSize = 50

	// Maximum size of processed headers batch.
	headersBatchSize = 5

	// Maximum time for which the pulling process will wait for a single header
	// to be delivered by the headers queue.
	headerTimeout = 1 * time.Second

	// Block duration of a Bitcoin difficulty epoch.
	difficultyEpochDuration = 2016

	// Duration for which the forwarder should rest after performing
	// a push action.
	forwarderPushingSleepTime = 45 * time.Second

	// Duration for which the forwarder should rest after reaching the tip of
	// Bitcoin blockchain
	forwarderPullingSleepTime = 60 * time.Second
)

var logger = log.Logger("relay-block-forwarder")

// Forwarder takes blocks from the Bitcoin chain and forwards them to the
// given host chain.
type Forwarder struct {
	btcChain  btc.Handle
	hostChain chain.Handle

	processedHeaders int

	headersQueue chan *btc.Header
	errChan      chan error
	quit         chan bool
}

// RunForwarder creates an instance of the block forwarder and runs its
// processing loops. The lifecycle of the forwarder can be managed using the
// passed context.
func RunForwarder(
	ctx context.Context,
	btcChain btc.Handle,
	hostChain chain.Handle,
) *Forwarder {
	forwarder := &Forwarder{
		btcChain:     btcChain,
		hostChain:    hostChain,
		headersQueue: make(chan *btc.Header, headersQueueSize),
		errChan:      make(chan error, 1),
		quit:         make(chan bool, 1),
	}

	go forwarder.pullingLoop(ctx)
	go forwarder.pushingLoop(ctx)

	return forwarder
}

func (f *Forwarder) findBestBlock() (*btc.Header, error) {
	currentBestDigest, err := f.hostChain.GetBestKnownDigest()
	if err != nil {
		return nil, err
	}

	logger.Infof("best known digest returned from Ethereum network: %s",
		hex.EncodeToString(currentBestDigest[:]),
	)

	bestHeader, err := f.btcChain.GetHeaderByDigest(currentBestDigest)
	if err != nil {
		return nil, err
	}

	betterOrSameHeader, err := f.btcChain.GetHeaderByHeight(bestHeader.Height)
	if err != nil {
		return nil, err
	}

	// see if there's a better block at that height
	// if so, crawl backwards

	// TODO: Is it ever possible that bestHeader and betterOrSameHeader are not
	// equal?
	// TODO: Consider just comparing hashes - it should be enough
	for !headersEqual(bestHeader, betterOrSameHeader) {
		bestHeader, err = f.btcChain.GetHeaderByDigest(bestHeader.PrevHash)
		if err != nil {
			return nil, err
		}

		betterOrSameHeader, err = f.btcChain.GetHeaderByHeight(bestHeader.Height)
		if err != nil {
			return nil, err
		}
	}

	return bestHeader, nil
}

func (f *Forwarder) pullingLoop(ctx context.Context) {
	logger.Infof("running forwarder pulling loop")

	latestHeader, err := f.findBestBlock()
	if err != nil {
		f.errChan <- fmt.Errorf(
			"failure while trying to find best block for pulling loop: [%v]",
			err,
		)
		return
	}

	logger.Infof("starting pulling loop with header: hash %s at height %d",
		latestHeader.Hash.String(), latestHeader.Height)

	latestHeight := latestHeader.Height + 1
	lastAdded := &btc.Header{}

	for {
		select {
		case <-ctx.Done():
			logger.Infof("forwarder context is done")
			return
		case <-f.quit:
			return
		default:
			chainHeight, err := f.btcChain.GetBlockCount()
			if err != nil {
				f.errChan <- fmt.Errorf("could not get block count [%v]", err)
				return
			}

			if latestHeight <= chainHeight {
				newHeader, err := f.btcChain.GetHeaderByHeight(latestHeight)
				if err != nil {
					f.errChan <- fmt.Errorf(
						"could not get header by height at %d: [%v]",
						latestHeight,
						err,
					)
					return
				}

				// TODO: Consider just comparing hashes - should be enough
				if !headersEqual(newHeader, lastAdded) {
					f.headersQueue <- newHeader
					copyHeaders(lastAdded, newHeader)
					latestHeight++
				}
			} else {
				// Sleep for a while until the Bitcoin blockchain has more blocks
				select {
				case <-time.After(forwarderPullingSleepTime):
				case <-ctx.Done():
				case <-f.quit:
				}
			}
		}
	}
}

func (f *Forwarder) pushingLoop(ctx context.Context) {
	logger.Infof("running new block pushing loop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-f.quit:
			return
		default:
			logger.Infof("pulling new headers from queue")

			headers := f.pullHeadersFromQueue(ctx)
			if len(headers) == 0 {
				continue
			}

			logger.Infof(
				"pushing [%v] header(s) to host chain",
				len(headers),
			)

			if err := f.pushHeadersToHostChain(ctx, headers); err != nil {
				f.errChan <- fmt.Errorf("could not push headers: [%v]", err)
				return
			}

			logger.Infof(
				"suspending block pushing loop for [%v]",
				forwarderPushingSleepTime,
			)

			// Sleep for a while to achieve a limited rate.
			select {
			case <-time.After(forwarderPushingSleepTime):
			case <-ctx.Done():
			case <-f.quit:
			}
		}
	}
}

// ErrChan returns the error channel of the forwarder. Once an error
// appears here, the forwarder loop is immediately terminated.
func (f *Forwarder) ErrChan() <-chan error {
	return f.errChan
}

// QuitChan returns the quit channel of the forwarder. It is used to indicate
// that a pulling or pushing loop must stop because an error occurred.
func (f *Forwarder) QuitChan() chan<- bool {
	return f.quit
}

func headersEqual(first, second *btc.Header) bool {
	return first.Hash == second.Hash &&
		first.Height == second.Height &&
		first.PrevHash == second.PrevHash &&
		first.MerkleRoot == second.MerkleRoot &&
		bytes.Compare(first.Raw, second.Raw) == 0
}

func copyHeaders(dest, src *btc.Header) {
	*dest = *src
	dest.Raw = make([]byte, len(src.Raw))
	copy(dest.Raw, src.Raw)
}
