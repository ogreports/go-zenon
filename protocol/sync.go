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

package protocol

import (
	"math/rand"
	"time"

	"github.com/zenon-network/go-zenon/chain/nom"
	"github.com/zenon-network/go-zenon/p2p/discover"
)

const (
	forceSyncCycle = 4 * time.Second // Time interval to force syncs, even if few peers are available
)

type txsync struct {
	p   *peer
	txs []*nom.AccountBlock
}

// syncTransactions starts sending all currently pending transactions to the given peer.
func (pm *ProtocolManager) syncTransactions(p *peer) {
	txs := pm.txpool.GetTransactions()
	if len(txs) == 0 {
		return
	}
	select {
	case pm.txsyncCh <- &txsync{p, txs}:
	case <-pm.quitSync:
	}
}

// txsyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (pm *ProtocolManager) txsyncLoop() {
	var (
		pending = make(map[discover.NodeID]*txsync)
		sending = false               // whether a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)

	// send starts a sending a pack of transactions from the sync.
	send := func(s *txsync) {
		// Fill pack with transactions up to the target size.
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs); i++ {
			pack.txs = append(pack.txs, s.txs[i])
		}
		// Remove the transactions that will be sent.
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.p.ID())
		}
		// Send the pack in the background.
		log.Debug("sending transactions", "peer-id", s.p.Peer.ID(), "num-blocks", len(pack.txs))
		sending = true
		go func() { done <- pack.p.SendTransactions(pack.txs) }()
	}

	// pick chooses the next pending sync.
	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-pm.txsyncCh:
			pending[s.p.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				log.Info("tx send failed", "peer-id", pack.p.Peer.ID(), "reason", err)
				delete(pending, pack.p.ID())
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-pm.quitSync:
			return
		}
	}
}

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (pm *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	pm.fetcher.Start()
	defer pm.fetcher.Stop()
	defer pm.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	forceSync := time.Tick(forceSyncCycle)
	for {
		select {
		case <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			if pm.peers.Len() < pm.minPeers {
				break
			}
			go pm.synchronise(pm.peers.BestPeer())

		case <-forceSync:
			// Force a sync even if not enough peers are present
			if pm.peers.Len() < pm.minPeers {
				break
			}
			go pm.synchronise(pm.peers.BestPeer())

		case <-pm.quitSync:
			return
		}
	}
}

func (pm *ProtocolManager) syncInfo() *SyncInfo {
	// find height details
	currentHeight := pm.chainman.CurrentBlock().Height
	targetHeight := uint64(0)
	if best := pm.peers.BestPeer(); best != nil {
		targetHeight = best.Td()
	}

	// find state
	var state SyncState
	if currentHeight >= targetHeight {
		state = SyncDone
	} else {
		state = Syncing
	}

	// if there are not enough peers, return this instead
	if pm.peers.Len() < pm.minPeers {
		state = NotEnoughPeers
	}

	return &SyncInfo{
		State:         state,
		CurrentHeight: currentHeight,
		TargetHeight:  targetHeight,
	}
}

// synchronise tries to sync up our local block chain with a remote peer.
func (pm *ProtocolManager) synchronise(peer *peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}

	// Make sure the peer's TD is higher than our own. If not drop.
	if peer.Td() <= pm.chainman.CurrentBlock().Height {
		return
	}

	log.Debug("syncing", "peer-id", peer.Peer.ID(), "peer-height", peer.td, "our-height", pm.chainman.CurrentBlock().Height)
	// Otherwise, try to sync with the downloader
	pm.downloader.Synchronise(peer.id, peer.Head(), peer.Td())
}
