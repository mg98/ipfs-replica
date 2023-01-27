package main

import (
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multicodec"
	rg "github.com/redislabs/redisgraph-go"
)

// newNode creates a new redis graph node struct.
func newNode(_cid cid.Cid) *rg.Node {
	return rg.NodeNew("Block", _cid.String(), map[string]interface{}{
		"cid":   _cid.String(),
		"codec": multicodec.Code(_cid.Type()).String(),
	})
}
