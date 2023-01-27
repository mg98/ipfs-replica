package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ipfs/go-cid"
	rg "github.com/redislabs/redisgraph-go"
	"log"
	"os"
	"strings"
)

type IPFSFetcher struct {
	ctx          context.Context
	node         IPFSNode
	graph        *rg.Graph
	DownloadPath string
}

func NewIPFSFetcher(ctx context.Context, node IPFSNode, graph *rg.Graph, downloadPath string) *IPFSFetcher {
	if err := os.Mkdir(downloadPath, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
		log.Fatalf("error creating data folder: %v", err)
	}
	return &IPFSFetcher{
		ctx:          ctx,
		node:         node,
		graph:        graph,
		DownloadPath: downloadPath,
	}
}

// Download will download the contents of the CID. This initiates a recursive process that creates the according
// nodes and edges to the db graph and eventually creates the raw data as blobs on the disk.
func (f *IPFSFetcher) Download(_cid cid.Cid, index int, parentNode *rg.Node) {
	log.Println("Download " + _cid.String())

	// create node
	node := newNode(_cid)
	qr, err := f.graph.Query("MERGE " + node.Encode())
	if err != nil {
		log.Fatalf("failed to merge node of CID %s: %v", _cid.String(), err)
	}
	if qr.NodesCreated() > 0 {
		log.Println("Node added: " + _cid.String())
	}

	// create edge to its parent
	if parentNode != nil {
		if _, err := f.graph.Query(fmt.Sprintf(
			"MATCH (a:Block{cid:'%s'}), (b:Block{cid:'%s'}) MERGE (a)-[:has{index:%d}]->(b)",
			parentNode.GetProperty("cid"),
			node.GetProperty("cid"),
			index,
		)); err != nil {
			log.Fatal(err)
		}
		log.Println("Edge added: " + parentNode.Alias + " has " + _cid.String())
	}

	// if node already existed, we are done here
	if qr.NodesCreated() == 0 {
		return
	}

	/**
	In CIDv0, everything is a DAG-PB and further decoding is necessary to interpret the data (else-block).
	In CIDv1, raw contents (and raw contents only) are encoded as RAW.
	*/
	if _cid.Type() == cid.Raw {
		if _, err := f.graph.Query(fmt.Sprintf("MATCH (b:Block {cid: '%s'}) SET b.type = 'Raw'", _cid.String())); err != nil {
			log.Fatalf("failed to update type for node with CID %s: %v", _cid.String(), err)
		}

		if _, err := jobs.Execute(func() {
			f.DownloadRawObject(_cid)
		}); err != nil {
			log.Fatal(err)
		}
	} else {
		// get dag links and possibly attached raw data
		fsNode, links, err := f.node.GetDAG(_cid)
		if err != nil && (errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) || strings.Contains(err.Error(), "context deadline exceeded")) {
			log.Printf("Timeout for CID %s. Skip!", _cid.String())
		} else if err != nil {
			log.Printf("GetDAG for cid %s failed with error: %v\n", _cid.String(), err)
		}

		if fsNode != nil {
			if _, err := f.graph.Query(fmt.Sprintf(
				"MATCH (b:Block {cid: '%s'}) SET b.type = '%s'",
				_cid.String(),
				fsNode.Type().String(),
			)); err != nil {
				log.Fatalf("failed to update type for node with cid %s: %v", _cid.String(), err)
			}
		}

		// Policy: If it has links, treat it as a dag; otherwise, treat it as a raw. Lol.
		if len(links) > 0 {
			// a set is sufficient and will reduce redundant and expensive network requests
			linkedCids := NewSet[cid.Cid]()
			for _, link := range links {
				linkedCids.Add(link.Cid)
			}
			for i, ref := range linkedCids.Values() {
				// recursively call Download on all refs
				f.Download(ref, i, node)
			}
		} else if fsNode != nil && len(fsNode.Data()) > 0 {
			if _, err := jobs.Execute(func() {
				f.SaveRawObject(_cid, fsNode.Data())
			}); err != nil {
				log.Fatal(err)
			}
		}
	}
}

// DownloadRawObject downloads the CID's raw content to a binary file on the disk.
func (f *IPFSFetcher) DownloadRawObject(_cid cid.Cid) {
	// check if file already exists
	if _, err := os.Stat(f.DownloadPath + "/" + _cid.String()); err == nil {
		return
	}

	// get file contents as binary
	file, err := f.node.GetFile(_cid)
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) || strings.Contains(err.Error(), "context deadline exceeded")) {
		log.Printf("Timeout for CID %s. Skip!", _cid.String())
		return
	} else if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(f.DownloadPath+"/"+_cid.String(), file, 0644); err != nil {
		log.Fatal("failed to write cid contents to file: ", err)
	}

	log.Printf("New file downloaded (CID: %s, Size: %d).\n", _cid.String(), len(file))
}

// SaveRawObject save the CID's raw content to a binary file on the disk.
func (f *IPFSFetcher) SaveRawObject(_cid cid.Cid, raw []byte) {
	// check if file already exists
	if _, err := os.Stat(f.DownloadPath + _cid.String()); err == nil {
		return
	}

	if err := os.WriteFile(f.DownloadPath+"/"+_cid.String(), raw, 0644); err != nil {
		log.Fatal("failed to write cid contents to file: ", err)
	}

	log.Printf("New file downloaded (CID: %s, Size: %d).\n", _cid.String(), len(raw))
}
