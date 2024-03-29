package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Block contains the blockchain data.
type Block struct {
	Pos       int
	Data      MedRecordCheckout
	Timestamp string
	Hash      string
	PrevHash  string
}

// MedRecordCheckout contains the checked out Medical record
type MedRecordCheckout struct {
	MedRecordID       string `json:"medrecord_id"`
	User         string `json:"user"`
	AppointmentDate string `json:"appointment_date"`
	AppointmentData string `json:"appointment_data"`
	IsGenesis    bool   `json:"is_genesis"`
}

// MedRecord contains data structure of the  medical record
type MedRecord struct {
	ID          string `json:"id"`
	Patient       string `json:"patient"`
	Doctor      string `json:"doctor"`
	RecordDate string `json:"record_date"`
	EMRID       string `json:"emrid:`
}

func (b *Block) generateHash() {
	// get string val of the Data
	bytes, _ := json.Marshal(b.Data)
	// concatenate the dataset
	data := string(b.Pos) + b.Timestamp + string(bytes) + b.PrevHash
	hash := sha256.New()
	hash.Write([]byte(data))
	b.Hash = hex.EncodeToString(hash.Sum(nil))
}

func CreateBlock(prevBlock *Block, checkoutItem MedRecordCheckout) *Block {
	block := &Block{}
	block.Pos = prevBlock.Pos + 1
	block.Timestamp = time.Now().String()
	block.Data = checkoutItem
	block.PrevHash = prevBlock.Hash
	block.generateHash()

	return block
}

// Blockchain is an ordered list of blocks
type Blockchain struct {
	blocks []*Block
}

// BlockChain is a global variable that'll return the mutated Blockchain struct
var BlockChain *Blockchain

// AddBlock adds a Block to a Blockchain
func (bc *Blockchain) AddBlock(data MedRecordCheckout) {
	// get previous block
	prevBlock := bc.blocks[len(bc.blocks)-1]
	// create new block
	block := CreateBlock(prevBlock, data)
	//  validate integrity of blocks
	if validBlock(block, prevBlock) {
		bc.blocks = append(bc.blocks, block)
	}
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, MedRecordCheckout{IsGenesis: true})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func validBlock(block, prevBlock *Block) bool {
	// Confirm the hashes
	if prevBlock.Hash != block.PrevHash {
		return false
	}
	// confirm the block's hash is valid
	if !block.validateHash(block.Hash) {
		return false
	}
	// Check the position to confirm its been incremented
	if prevBlock.Pos+1 != block.Pos {
		return false
	}
	return true
}

func (b *Block) validateHash(hash string) bool {
	b.generateHash()
	if b.Hash != hash {
		return false
	}
	return true
}

func getBlockchain(w http.ResponseWriter, r *http.Request) {
	jbytes, err := json.MarshalIndent(BlockChain.blocks, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	// write JSON string
	io.WriteString(w, string(jbytes))
}

func writeBlock(w http.ResponseWriter, r *http.Request) {
	var checkoutItem MedRecordCheckout
	if err := json.NewDecoder(r.Body).Decode(&checkoutItem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not write Block: %v", err)
		w.Write([]byte("could not write block"))
		return
	}
	// create block
	BlockChain.AddBlock(checkoutItem)
	resp, err := json.MarshalIndent(checkoutItem, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not write block"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func newMedRecord(w http.ResponseWriter, r *http.Request) {
	var medrecord MedRecord
	if err := json.NewDecoder(r.Body).Decode(&medrecord); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not create: %v", err)
		w.Write([]byte("could not create new MedRecord"))
		return
	}
	// We'll create an ID, concatenating Data and medical record date
	// @Todo implement a microservice that brokers patient data

	h := md5.New()
	io.WriteString(h, medrecord.EMRID+medrecord.RecordDate)
	medrecord.ID = fmt.Sprintf("%x", h.Sum(nil))

	// send back payload
	resp, err := json.MarshalIndent(medrecord, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not save medrecord data"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func main() {
	// initialize the blockchain and store in var
	// @Todo update program to broadcast to multiple peers connect to Tendermint API for consensus
	// @Todo Enable blockchain persistance
	// @Todo Client application to interface with Blockchain
	// @Todo CLI commands

	BlockChain = NewBlockchain()

	// register router
	r := mux.NewRouter()
	r.HandleFunc("/", getBlockchain).Methods("GET")
	r.HandleFunc("/", writeBlock).Methods("POST")
	r.HandleFunc("/new", newMedRecord).Methods("POST")

	// dump the state of the Blockchain to the console
	go func() {
		//for {
		for _, block := range BlockChain.blocks {
			fmt.Printf("Prev. hash: %x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data: %v\n", string(bytes))
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Println()
		}
		//}
	}()
	log.Println("Listening on port 3000")

	log.Fatal(http.ListenAndServe(":3000", r))
}