package controller

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/types"
)

// NewBlockchain returns a new initialized blockchain.
func NewBlockchain(conf peer.Configuration, log *zerolog.Logger) blockchain {
	return blockchain{
		conf: conf,
		log:  log,
	}
}

type blockchain struct {
	conf peer.Configuration
	log  *zerolog.Logger
}

func (b blockchain) BlockchainHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			b.blockchainGet(w, r)
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
		}
	}
}

func (b blockchain) blockchainGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var blocks []viewBlock
	if b.conf.UsePBFT {
		blocks = b.getPbftViewBlocks(w)
	} else {
		blocks = b.getPaxosViewBlocks(w)
	}
	viewData := struct {
		NodeAddr      string
		LastBlockHash string
		Blocks        []viewBlock
	}{
		NodeAddr:      b.conf.Socket.GetAddress(),
		LastBlockHash: "", //hex.EncodeToString(store.Get(storage.LastBlockKey)),
		Blocks:        blocks,
	}

	tmpl, err := template.New("html").ParseFiles(("httpnode/controller/blockchain.gohtml"))
	if err != nil {
		http.Error(w, "failed to parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	println("hello-----------------leave blockchainGet, block length:", len(blocks))
	tmpl.ExecuteTemplate(w, "blockchain.gohtml", viewData)
}

func (b blockchain) getPbftViewBlocks(w http.ResponseWriter) []viewBlock {
	blocks := []viewBlock{}
	addresses := b.conf.CothorityAddress
	hrStore := b.conf.HrStorage.GetHrBlockchainStore()
	fmt.Println("addresses", addresses)
	for _, address := range addresses {
		lastBlockHashHex := hex.EncodeToString(hrStore.Get(address, storage.LastBlockKey))
		fmt.Println("lastBlockHashHex", lastBlockHashHex, "address", address)
		endBlockHasHex := hex.EncodeToString(make([]byte, 32))

		for lastBlockHashHex != endBlockHasHex {
			lastBlockBuf := hrStore.Get(address, string(lastBlockHashHex))

			if lastBlockBuf == nil {
				break
			}

			var lastBlock types.PbftchainBlock

			err := lastBlock.Unmarshal(lastBlockBuf)
			if err != nil {
				http.Error(w, "failed to unmarshal block: "+err.Error(), http.StatusInternalServerError)
				return nil
			}
			recordBuf, _ := hex.DecodeString(lastBlock.Request.Filename)
			var record bytes.Buffer
			json.Indent(&record, recordBuf, "", "    ")

			blocks = append(blocks, viewBlock{
				Index:    lastBlock.Index,
				Hash:     hex.EncodeToString(lastBlock.Hash),
				ValueID:  lastBlock.Request.IDhr,
				Name:     record.String(),            //lastBlock.Value.Filename,
				Metahash: lastBlock.Request.Metahash, //lastBlock.Request.Metahash,
				PrevHash: hex.EncodeToString(lastBlock.PrevHash),
			})

			lastBlockHashHex = hex.EncodeToString(lastBlock.PrevHash)
		}
	}
	return blocks
}

func (b blockchain) getPaxosViewBlocks(w http.ResponseWriter) []viewBlock {
	store := b.conf.Storage.GetBlockchainStore()

	lastBlockHashHex := hex.EncodeToString(store.Get(storage.LastBlockKey))
	endBlockHasHex := hex.EncodeToString(make([]byte, 32))

	if lastBlockHashHex == "" {
		lastBlockHashHex = endBlockHasHex
	}

	blocks := []viewBlock{}

	for lastBlockHashHex != endBlockHasHex {
		lastBlockBuf := store.Get(string(lastBlockHashHex))

		var lastBlock types.BlockchainBlock

		err := lastBlock.Unmarshal(lastBlockBuf)
		if err != nil {
			http.Error(w, "failed to unmarshal block: "+err.Error(), http.StatusInternalServerError)
			return nil
		}

		recordBuf, _ := hex.DecodeString(lastBlock.Value.Filename)
		var record bytes.Buffer
		json.Indent(&record, recordBuf, "", "    ")
		blocks = append(blocks, viewBlock{
			Index:    lastBlock.Index,
			Hash:     hex.EncodeToString(lastBlock.Hash),
			ValueID:  lastBlock.Value.UniqID,
			Name:     record.String(),          //lastBlock.Value.Filename,
			Metahash: lastBlock.Value.Metahash, //lastBlock.Value.Metahash,
			PrevHash: hex.EncodeToString(lastBlock.PrevHash),
		})

		lastBlockHashHex = hex.EncodeToString(lastBlock.PrevHash)
	}
	return blocks
}

type viewBlock struct {
	Index    uint
	Hash     string
	ValueID  string
	Name     string
	Metahash string
	PrevHash string
}
