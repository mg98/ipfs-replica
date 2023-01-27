package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	"github.com/korovkin/limiter"
	rg "github.com/redislabs/redisgraph-go"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

const ipfsTestDataPath = ".test-data"

var mockedFetcher *IPFSFetcher
var graphTest rg.Graph

func init() {
	conn, err := redis.Dial("tcp", rgHost)
	if err != nil {
		log.Fatal(err)
	}
	graphTest = rg.GraphNew("ipfs_test", conn)
	graphTest.Delete()
	mockedFetcher = NewIPFSFetcher(
		context.Background(),
		NewMockIPFSNode(),
		&graphTest,
		ipfsTestDataPath,
	)
}

func TestIPFSFetcher_DownloadRawObject(t *testing.T) {
	if err := os.Mkdir(ipfsTestDataPath, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
		panic(err)
	}
	defer os.RemoveAll(ipfsTestDataPath)
	mockedFetcher.DownloadRawObject(cid.MustParse(rawCID))
	defer graphTest.Query("MATCH (b:Block) DELETE b")
	const filePath = ipfsTestDataPath + "/" + rawCID

	t.Run("file is created", func(t *testing.T) {
		_, err := os.Stat(filePath)
		assert.Nil(t, err)
	})
	t.Run("file has expected contents", func(t *testing.T) {
		blob, err := os.ReadFile(filePath)
		assert.Nil(t, err)
		assert.Equal(t, []byte{0x00, 0xFF, 0x00, 0xFF}, blob)
	})
	t.Run("can safely re-download this file", func(t *testing.T) {
		mockedFetcher.DownloadRawObject(cid.MustParse(rawCID))
		blob, err := os.ReadFile(filePath)
		assert.Nil(t, err)
		assert.Equal(t, []byte{0x00, 0xFF, 0x00, 0xFF}, blob)
	})
}

func TestIPFSFetcher_Download(t *testing.T) {
	if err := os.Mkdir(ipfsTestDataPath, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
		panic(err)
	}
	defer os.RemoveAll(ipfsTestDataPath)
	defer graphTest.Query("MATCH (b:Block) DELETE b")

	t.Run("raw object with no parent", func(t *testing.T) {
		const filePath = ipfsTestDataPath + "/" + rawCID
		jobs = limiter.NewConcurrencyLimiter(1)
		mockedFetcher.Download(cid.MustParse(rawCID), 0, nil)
		defer os.Remove(filePath)
		jobs.WaitAndClose()

		t.Run("create node and file for raw object", func(t *testing.T) {
			// check if node exists
			res, err := graphTest.Query(fmt.Sprintf("MATCH (b:Block { cid: '%s' }) RETURN b", rawCID))
			assert.Nil(t, err)
			assert.True(t, res.Next())
			b, ok := res.Record().Get("b")
			assert.True(t, ok)
			assert.Equal(t, "raw", b.(*rg.Node).GetProperty("codec"))
			assert.Equal(t, rawCID, b.(*rg.Node).GetProperty("cid"))
			assert.False(t, res.Next())

			// check if file exists
			bs, err := os.ReadFile(filePath)
			assert.Nil(t, err)
			assert.Equal(t, []byte{0x00, 0xFF, 0x00, 0xFF}, bs)
		})

		t.Run("handle duplicate encounter", func(t *testing.T) {
			jobs = limiter.NewConcurrencyLimiter(1)
			mockedFetcher.Download(cid.MustParse(rawCID), 0, nil)
			jobs.WaitAndClose()

			// check if node exists with no duplicate
			res, err := graphTest.Query(fmt.Sprintf("MATCH (b:Block { cid: '%s' }) RETURN b", rawCID))
			assert.Nil(t, err)
			assert.True(t, res.Next())
			b, ok := res.Record().Get("b")
			assert.True(t, ok)
			assert.Equal(t, "raw", b.(*rg.Node).GetProperty("codec"))
			assert.Equal(t, rawCID, b.(*rg.Node).GetProperty("cid"))
			assert.False(t, res.Next())

			// check if file still exists
			bs, err := os.ReadFile(filePath)
			assert.Nil(t, err)
			assert.Equal(t, []byte{0x00, 0xFF, 0x00, 0xFF}, bs)
		})
	})

	t.Run("file with 3 raw objects", func(t *testing.T) {
		jobs = limiter.NewConcurrencyLimiter(1)
		mockedFetcher.Download(cid.MustParse(fileCID), 0, nil)
		jobs.WaitAndClose()

		res, err := graphTest.Query(fmt.Sprintf("MATCH (f:Block { cid: '%s' }) RETURN f", fileCID))
		assert.Nil(t, err)
		assert.True(t, res.Next())
		f, ok := res.Record().Get("f")
		assert.True(t, ok)
		assert.Equal(t, "dag-pb", f.(*rg.Node).GetProperty("codec"))
		assert.Equal(t, fileCID, f.(*rg.Node).GetProperty("cid"))
		assert.False(t, res.Next())
	})

	t.Run("directory with file and raw object", func(t *testing.T) {
		jobs = limiter.NewConcurrencyLimiter(1)
		mockedFetcher.Download(cid.MustParse(directoryCID), 0, nil)
		jobs.WaitAndClose()

		t.Run("directory node exists uniquely", func(t *testing.T) {
			res, err := graphTest.Query(fmt.Sprintf("MATCH (b:Block { cid: '%s' }) RETURN b", directoryCID))
			assert.Nil(t, err)
			assert.True(t, res.Next())
			b, ok := res.Record().Get("b")
			assert.True(t, ok)
			assert.Equal(t, "dag-pb", b.(*rg.Node).GetProperty("codec"))
			assert.Equal(t, directoryCID, b.(*rg.Node).GetProperty("cid"))
			assert.False(t, res.Next())
		})

		t.Run("file node exists uniquely", func(t *testing.T) {
			res, err := graphTest.Query(fmt.Sprintf("MATCH (b:Block { cid: '%s' }) RETURN b", fileCID))
			assert.Nil(t, err)
			assert.True(t, res.Next())
			b, ok := res.Record().Get("b")
			assert.True(t, ok)
			assert.Equal(t, "dag-pb", b.(*rg.Node).GetProperty("codec"))
			assert.Equal(t, fileCID, b.(*rg.Node).GetProperty("cid"))
			assert.False(t, res.Next())
		})

		t.Run("yet another raw node exists uniquely", func(t *testing.T) {
			res, err := graphTest.Query(fmt.Sprintf("MATCH (b:Block { cid: '%s' }) RETURN b", yetAnotherRawCID))
			assert.Nil(t, err)
			assert.True(t, res.Next())
			b, ok := res.Record().Get("b")
			assert.True(t, ok)
			assert.Equal(t, "raw", b.(*rg.Node).GetProperty("codec"))
			assert.Equal(t, yetAnotherRawCID, b.(*rg.Node).GetProperty("cid"))
			assert.False(t, res.Next())
		})

		t.Run("directory node has file node", func(t *testing.T) {
			res, err := graphTest.Query(fmt.Sprintf(
				"MATCH (d:Block { cid: '%s' })-[:has]->(f:Block { cid: '%s' }) RETURN d, f",
				directoryCID,
				fileCID,
			))
			assert.Nil(t, err)
			assert.True(t, res.Next())
			assert.False(t, res.Next())
		})

		t.Run("directory node has yet another file node", func(t *testing.T) {
			res, err := graphTest.Query(fmt.Sprintf(
				"MATCH (d:Block { cid: '%s' })-[:has]->(f:Block { cid: '%s' }) RETURN d, f",
				directoryCID,
				yetAnotherRawCID,
			))
			assert.Nil(t, err)
			assert.True(t, res.Next())
			assert.False(t, res.Next())
		})
	})
}
