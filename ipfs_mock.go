package main

import (
	"errors"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	ft "github.com/ipfs/go-unixfs"
	_ "github.com/mattn/go-sqlite3"
)

const (
	rawCID           = "bafkreictqudr3cfhwkobq4jqmlzvkhr6nnopgv47hr3l3dy4cg5es4o7d4"    // 0x00FF00FF
	otherRawCID      = "bafkreifypznnfdih7jirykt32aleub7q3pommoypvqo6ovk3upelzmgq7u"    // 0xFFFFFFFF
	yetAnotherRawCID = "bafk2bzaceatgshdb7uzpl26uxsxen3lduhr635j6qhxbntilygtvraucju26a" // 0xFF00FF00
	fileCID          = "bafybeihis42cbqzrlacahelswxbxhs62jn45gisz72beo7i6lhu2nmbezq"    // [ rawCID, rawCID, otherRawCID ]
	directoryCID     = "QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D"                 // [ fileCID, yetAnotherRawCID ]
)

// MockIPFSNode is a mocked implementation of the IPFSNode node used for testing.
type MockIPFSNode struct{}

func NewMockIPFSNode() IPFSNode {
	return &MockIPFSNode{}
}

func (n *MockIPFSNode) GetFile(_cid cid.Cid) (res []byte, err error) {
	switch _cid.String() {
	case rawCID:
		return []byte{0x00, 0xFF, 0x00, 0xFF}, nil
	case otherRawCID:
		return []byte{0xFF, 0xFF, 0xFF, 0xFF}, nil
	case yetAnotherRawCID:
		return []byte{0xFF, 0x00, 0xFF, 0x00}, nil
	default:
		return nil, errors.New("invalid cid")
	}
}

func (n *MockIPFSNode) GetDAG(_cid cid.Cid) (*ft.FSNode, []*format.Link, error) {
	switch _cid.String() {
	case fileCID:
		return nil, []*format.Link{
			{Cid: cid.MustParse(rawCID)},
			{Cid: cid.MustParse(rawCID)},
			{Cid: cid.MustParse(otherRawCID)},
		}, nil
	case directoryCID:
		return nil, []*format.Link{
			{Cid: cid.MustParse(fileCID)},
			{Cid: cid.MustParse(yetAnotherRawCID)},
		}, nil
	default:
		return nil, nil, errors.New("invalid cid")
	}
}
