package main

import (
	bsmsg "github.com/ipfs/go-bitswap/message"
	"github.com/ipfs/go-cid"
	_ "github.com/trudi-group/ipfs-metric-exporter/metricplugin"
	"time"
)

// BitswapMessage copies the struct from metricplugin.BitswapMessage but adapts the type of ConnectedAddresses to
// circumvent the issue described in https://github.com/multiformats/go-multiaddr/issues/189.
type BitswapMessage struct {
	WantlistEntries    []bsmsg.Entry `json:"wantlist_entries"`
	FullWantList       bool          `json:"full_wantlist"`
	Blocks             []cid.Cid     `json:"blocks"`
	BlockPresences     []any         `json:"block_presences"`
	ConnectedAddresses []string      `json:"connected_addresses"`
}

// Event copies the struct from metricplugin.Event. See comment on BitswapMessage.
type Event struct {
	Timestamp      time.Time      `json:"timestamp"`
	Peer           string         `json:"peer"`
	BitswapMessage BitswapMessage `json:"bitswap_message,omitempty"`
}
