// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package fetcher contains the block announcement based synchonisation.
package fetcher

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/zenon-network/go-zenon/chain/nom"
	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/common/types"
)

const (
	arriveTimeout = 500 * time.Millisecond // Time allowance before an announced block is explicitly requested
	gatherSlack   = 100 * time.Millisecond // Interval used to collate almost-expired announces with fetches
	fetchTimeout  = 5 * time.Second        // Maximum alloted time to return an explicitly requested block
	maxUncleDist  = 7                      // Maximum allowed backward distance from the chain head
	maxQueueDist  = 32                     // Maximum allowed distance from the chain head to queue
	hashLimit     = 256                    // Maximum number of unique blocks a peer may have announced
	blockLimit    = 64                     // Maximum number of unique blocks a per may have delivered
)

var (
	log           = common.FetcherLogger
	errTerminated = errors.New("terminated")
)

// blockRetrievalFn is a callback type for retrieving a block from the local chain.
type blockRetrievalFn func(types.Hash) *nom.DetailedMomentum

// blockRequesterFn is a callback type for sending a block retrieval request.
type blockRequesterFn func([]types.Hash) error

// blockValidatorFn is a callback type to verify a block's header for fast propagation.
type blockValidatorFn func(block *nom.Momentum, parent *nom.Momentum) error

// blockBroadcasterFn is a callback type for broadcasting a block to connected peers.
type blockBroadcasterFn func(block *nom.DetailedMomentum, propagate bool)

// chainHeightFn is a callback type to retrieve the current chain height.
type chainHeightFn func() uint64

// chainInsertFn is a callback type to insert a batch of blocks into the local chain.
type chainInsertFn func([]*nom.DetailedMomentum) (int, error)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// announce is the hash notification of the availability of a new block in the
// network.
type announce struct {
	hash types.Hash // Hash of the block being announced
	time time.Time  // Timestamp of the announcement

	origin string           // Header of the peer originating the notification
	fetch  blockRequesterFn // Fetcher function to retrieve
}

// inject represents a schedules import operation.
type inject struct {
	origin   string
	detailed *nom.DetailedMomentum
}

// Fetcher is responsible for accumulating block announcements from various peers
// and scheduling them for retrieval.
type Fetcher struct {
	// Various event channels
	notify chan *announce
	inject chan *inject
	filter chan chan []*nom.DetailedMomentum
	done   chan types.Hash
	quit   chan struct{}

	// Announce states
	announces map[string]int             // Per peer announce counts to prevent memory exhaustion
	announced map[types.Hash][]*announce // Announced blocks, scheduled for fetching
	fetching  map[types.Hash]*announce   // Announced blocks, currently fetching

	// Block cache
	queue  *prque.Prque           // Queue containing the import operations (block number sorted)
	queues map[string]int         // Per peer block counts to prevent memory exhaustion
	queued map[types.Hash]*inject // Set of already queued blocks (to dedup imports)

	// Callbacks
	getBlock       blockRetrievalFn   // Retrieves a block from the local chain
	validateBlock  blockValidatorFn   // Checks if a block's headers have a valid proof of work
	broadcastBlock blockBroadcasterFn // Broadcasts a block to connected peers
	chainHeight    chainHeightFn      // Retrieves the current chain's height
	insertChain    chainInsertFn      // Injects a batch of blocks into the chain
	dropPeer       peerDropFn         // Drops a peer for misbehaving

	// Testing hooks
	fetchingHook func([]types.Hash)  // Method to call upon starting a block fetch
	importedHook func(*nom.Momentum) // Method to call upon successful block import
}

// New creates a block fetcher to retrieve blocks based on hash announcements.
func New(getBlock blockRetrievalFn, validateBlock blockValidatorFn, broadcastBlock blockBroadcasterFn, chainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn) *Fetcher {
	return &Fetcher{
		notify:         make(chan *announce),
		inject:         make(chan *inject),
		filter:         make(chan chan []*nom.DetailedMomentum),
		done:           make(chan types.Hash),
		quit:           make(chan struct{}),
		announces:      make(map[string]int),
		announced:      make(map[types.Hash][]*announce),
		fetching:       make(map[types.Hash]*announce),
		queue:          prque.New(),
		queues:         make(map[string]int),
		queued:         make(map[types.Hash]*inject),
		getBlock:       getBlock,
		validateBlock:  validateBlock,
		broadcastBlock: broadcastBlock,
		chainHeight:    chainHeight,
		insertChain:    insertChain,
		dropPeer:       dropPeer,
	}
}

// Start boots up the announcement based synchoniser, accepting and processing
// hash notifications and block fetches until termination requested.
func (f *Fetcher) Start() {
	go f.loop()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (f *Fetcher) Stop() {
	close(f.quit)
}

// Notify announces the fetcher of the potential availability of a new block in
// the network.
func (f *Fetcher) Notify(peer string, hash types.Hash, time time.Time, fetcher blockRequesterFn) error {
	block := &announce{
		hash:   hash,
		time:   time,
		origin: peer,
		fetch:  fetcher,
	}
	select {
	case f.notify <- block:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Enqueue tries to fill gaps the the fetcher's future import queue.
func (f *Fetcher) Enqueue(peer string, block *nom.DetailedMomentum) error {
	op := &inject{
		origin:   peer,
		detailed: block,
	}
	select {
	case f.inject <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Filter extracts all the blocks that were explicitly requested by the fetcher,
// returning those that should be handled differently.
func (f *Fetcher) Filter(blocks []*nom.DetailedMomentum) []*nom.DetailedMomentum {
	// Send the filter channel to the fetcher
	filter := make(chan []*nom.DetailedMomentum)

	select {
	case f.filter <- filter:
	case <-f.quit:
		return nil
	}
	// Request the filtering of the block list
	select {
	case filter <- blocks:
	case <-f.quit:
		return nil
	}
	// Retrieve the blocks remaining after filtering
	select {
	case blocks := <-filter:
		return blocks
	case <-f.quit:
		return nil
	}
}

// Loop is the main fetcher loop, checking and processing various notification
// events.
func (f *Fetcher) loop() {
	// Iterate the block fetching until a quit is requested
	fetch := time.NewTimer(0)
	for {
		// Clean up any expired block fetches
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > fetchTimeout {
				f.forgetHash(hash)
			}
		}
		// Import any queued blocks that could potentially fit
		height := f.chainHeight()
		for !f.queue.Empty() {
			op := f.queue.PopItem().(*inject)
			momentum := op.detailed.Momentum

			// If too high up the chain or phase, continue later
			number := momentum.Height
			if number > height+1 {
				f.queue.Push(op, -float32(momentum.Height))
				break
			}
			// Otherwise if fresh and still unknown, try and import
			hash := momentum.Hash
			if number+maxUncleDist < height || f.getBlock(hash) != nil {
				f.forgetBlock(hash)
				continue
			}
			f.insert(op.origin, op.detailed)
		}
		// Wait for an outside event to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case notification := <-f.notify:
			// A block was announced, make sure the peer isn't DOSing us
			count := f.announces[notification.origin] + 1
			if count > hashLimit {
				log.Info("Peer exceeded outstanding announces", "peer", notification.origin, "hash-limit", hashLimit)
				break
			}
			// All is well, schedule the announce if block's not yet downloading
			if _, ok := f.fetching[notification.hash]; ok {
				break
			}
			f.announces[notification.origin] = count
			f.announced[notification.hash] = append(f.announced[notification.hash], notification)
			if len(f.announced) == 1 {
				f.reschedule(fetch)
			}

		case op := <-f.inject:
			// A direct block insertion was requested, try and fill any pending gaps
			f.enqueue(op.origin, op.detailed)

		case hash := <-f.done:
			// A pending import finished, remove all traces of the notification
			f.forgetHash(hash)
			f.forgetBlock(hash)

		case <-fetch.C:
			// At least one block's timer ran out, check for needing retrieval
			request := make(map[string][]types.Hash)

			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arriveTimeout-gatherSlack {
					// Pick a random peer to retrieve from, reset all others
					announce := announces[rand.Intn(len(announces))]
					f.forgetHash(hash)

					// If the block still didn't arrive, queue for fetching
					if f.getBlock(hash) == nil {
						request[announce.origin] = append(request[announce.origin], hash)
						f.fetching[hash] = announce
					}
				}
			}
			// Send out all block requests
			for peer, hashes := range request {
				if len(hashes) > 0 {
					list := "["
					for _, hash := range hashes {
						list += fmt.Sprintf("%x, ", hash[:4])
					}
					list = list[:len(list)-2] + "]"

					log.Debug("fetching", "peer", peer, "hashes", list)
				}
				// Create a closure of the fetch and schedule in on a new thread
				fetcher, hashes := f.fetching[hashes[0]].fetch, hashes
				go func() {
					if f.fetchingHook != nil {
						f.fetchingHook(hashes)
					}
					fetcher(hashes)
				}()
			}
			// Schedule the next fetch if blocks are still pending
			f.reschedule(fetch)

		case filter := <-f.filter:
			// Blocks arrived, extract any explicit fetches, return all else
			var blocks []*nom.DetailedMomentum
			select {
			case blocks = <-filter:
			case <-f.quit:
				return
			}

			var explicit []*nom.DetailedMomentum
			var download []*nom.DetailedMomentum
			for _, detailed := range blocks {
				block := detailed.Momentum
				hash := block.Hash

				// Filter explicitly requested blocks from hash announcements
				if f.fetching[hash] != nil && f.queued[hash] == nil {
					// Discard if already imported by other means
					if f.getBlock(hash) == nil {
						explicit = append(explicit, detailed)
					} else {
						f.forgetHash(hash)
					}
				} else {
					download = append(download, detailed)
				}
			}

			select {
			case filter <- download:
			case <-f.quit:
				return
			}
			// Schedule the retrieved blocks for ordered import
			for _, block := range explicit {
				if announce := f.fetching[block.Momentum.Hash]; announce != nil {
					f.enqueue(announce.origin, block)
				}
			}
		}
	}
}

// reschedule resets the specified fetch timer to the next announce timeout.
func (f *Fetcher) reschedule(fetch *time.Timer) {
	// Short circuit if no blocks are announced
	if len(f.announced) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	fetch.Reset(arriveTimeout - time.Since(earliest))
}

// enqueue schedules a new future import operation, if the block to be imported
// has not yet been seen.
func (f *Fetcher) enqueue(peer string, detailed *nom.DetailedMomentum) {
	block := detailed.Momentum
	hash := block.Hash

	// Ensure the peer isn't DOSing us
	count := f.queues[peer] + 1
	if count > blockLimit {
		log.Info("Peer discarded block", "peer", peer, "momentum", block.Height, "hash", hash.Bytes()[:4], "exceeded allowance", blockLimit)
		return
	}
	// Discard any past or too distant blocks
	if dist := int64(block.Height) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
		log.Info("Peer discarded block", "peer", peer, "momentum", block.Height, "hash", hash.Bytes()[:4], "distance", dist)
		return
	}
	// Schedule the block for future importing
	if _, ok := f.queued[hash]; !ok {
		op := &inject{
			origin:   peer,
			detailed: detailed,
		}
		f.queues[peer] = count
		f.queued[hash] = op
		f.queue.Push(op, -float32(block.Height))

		log.Debug("Peer queued block", "peer", peer, "momentum", block.Height, "hash", hash.Bytes()[:4], "total", f.queue.Size())
	}
}

// insert spawns a new goroutine to run a block insertion into the chain. If the
// block's number is at the same height as the current import phase, if updates
// the phase states accordingly.
func (f *Fetcher) insert(peer string, detailed *nom.DetailedMomentum) {
	momentum := detailed.Momentum
	hash := momentum.Hash

	// Run the import on a new thread
	log.Info("Peer importing momentum", "peer", peer, "momentum", momentum.Height, "hash", hash[:4])
	go func() {
		defer func() { f.done <- hash }()

		// If the parent's unknown, abort insertion
		parent := f.getBlock(momentum.PreviousHash)
		if parent == nil {
			return
		}
		// Quickly validate the header and propagate the momentum if it passes
		switch err := f.validateBlock(momentum, parent.Momentum); err {
		case nil:
			// All ok, quickly propagate to our peers
			go f.broadcastBlock(detailed, true)

		default:
			// Something went very wrong, drop the peer
			log.Info("momentum verification failed", "peer", peer, "momentum", momentum.Height, "hash", hash[:4], "reason", err)
			f.dropPeer(peer)
			return
		}
		// Run the actual import and log any issues
		if _, err := f.insertChain([]*nom.DetailedMomentum{detailed}); err != nil {
			log.Warn("momentum import failed", "peer", peer, "momentum", momentum.Height, "hash", hash[:4], "reason", err)
			return
		}
		// If import succeeded, broadcast the momentum
		go f.broadcastBlock(detailed, false)

		// Invoke the testing hook if needed
		if f.importedHook != nil {
			f.importedHook(momentum)
		}
	}()
}

// forgetHash removes all traces of a block announcement from the fetcher's
// internal state.
func (f *Fetcher) forgetHash(hash types.Hash) {
	// Remove all pending announces and decrement DOS counters
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)

	// Remove any pending fetches and decrement the DOS counters
	if announce := f.fetching[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.fetching, hash)
	}
}

// forgetBlock removes all traces of a queued block frmo the fetcher's internal
// state.
func (f *Fetcher) forgetBlock(hash types.Hash) {
	if insert := f.queued[hash]; insert != nil {
		f.queues[insert.origin]--
		if f.queues[insert.origin] == 0 {
			delete(f.queues, insert.origin)
		}
		delete(f.queued, hash)
	}
}
