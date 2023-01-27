package main

import (
	"context"
	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	ft "github.com/ipfs/go-unixfs"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	_ "github.com/mattn/go-sqlite3"
	"github.com/multiformats/go-multiaddr"
	"io"
	"log"
)

// IPFSNode
type IPFSNode interface {
	GetFile(_cid cid.Cid) (res []byte, err error)
	GetDAG(_cid cid.Cid) (fsNode *ft.FSNode, links []*format.Link, err error)
}

// IPFSNodeImpl is an implementation of the IPFSNode node.
type IPFSNodeImpl struct {
	ctx  context.Context
	peer *ipfslite.Peer
}

// NewIPFSNode builds a node that it connects to the IPFS network and instantiates an IPFSNodeImpl.
func NewIPFSNode(ctx context.Context) (*IPFSNodeImpl, error) {
	ds := ipfslite.NewInMemoryDatastore()
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return nil, err
	}
	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/4005")
	libp2p.EnableRelay()
	h, dht, err := ipfslite.SetupLibp2p(
		ctx,
		priv,
		nil,
		[]multiaddr.Multiaddr{listen},
		ds,
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, err
	}
	peer, err := ipfslite.New(ctx, ds, nil, h, dht, nil)
	if err != nil {
		return nil, err
	}
	peer.Bootstrap(ipfslite.DefaultBootstrapPeers())

	return &IPFSNodeImpl{
		ctx:  ctx,
		peer: peer,
	}, nil
}

// GetFile from IPFS.
func (n *IPFSNodeImpl) GetFile(_cid cid.Cid) ([]byte, error) {
	ctx, cancel := context.WithTimeout(n.ctx, ipfsTimeout)
	defer cancel()
	rsc, err := n.peer.GetFile(ctx, _cid)
	if err != nil {
		return nil, err
	}
	if err := rsc.Close(); err != nil {
		return nil, err
	}
	return io.ReadAll(rsc)
}

// GetDAG returns the links of a block and in case it's a Protobuf, also the FSNode.
func (n *IPFSNodeImpl) GetDAG(_cid cid.Cid) (node *ft.FSNode, links []*format.Link, err error) {
	log.Println("Get DAG for " + _cid.String())
	ctx, cancel := context.WithTimeout(n.ctx, ipfsTimeout)
	defer cancel()

	var dag format.Node
	dag, err = n.peer.DAGService.Get(ctx, _cid)
	if err != nil {
		return
	}
	links = dag.Links()
	if dag.Cid().Type() == cid.DagProtobuf {
		if pn, ok := dag.(*merkledag.ProtoNode); ok {
			node, err = ft.FSNodeFromBytes(pn.Data())
			return
		}
	}
	return
}
