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
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Blockchain struct {
	blocks []*Block
}

type Block struct {
	Position  int
	Data      BookCheckout
	Timestamp string
	Hash      string
	PrevHash  string
}

type BookCheckout struct {
	BookID       string `json:"book_id"`
	User         string `json:"user"`
	CheckoutDate string `json:"checkout_date"`
	IsGenesis    bool   `json:"is_genesis"`
}

type Book struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	PublishDate string `json:"publish_date"`
	ISBN        string `json:"isbn"`
}

var NewBlockChain *Blockchain

func (block *Block) validateHash(hash string) bool {
	block.generateHash()
	return block.Hash == hash
}

func (block *Block) generateHash() {

	// get []bytes value of data
	bytes, err := json.Marshal(block.Data)

	if err != nil {
		log.Panic(err)
		return
	}

	// concatenate all of the data from the block
	data := strconv.Itoa(block.Position) + block.Timestamp + string(bytes) + block.PrevHash

	// hash the data using sha256
	hash := sha256.New()
	hash.Write([]byte(data))
	block.Hash = hex.EncodeToString(hash.Sum(nil))
}

func CreateBlock(prevBlock *Block, bookCheckoutItem BookCheckout) *Block {
	// empty block
	block := &Block{}

	// popoulate block dat
	block.Position = prevBlock.Position + 1
	block.PrevHash = prevBlock.Hash
	block.Timestamp = time.Now().String()
	block.Data = bookCheckoutItem
	block.generateHash()

	return block
}

func ValidBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if prevBlock.Position+1 != block.Position {
		return false
	}

	if !(block.validateHash(block.Hash)) {
		return false
	}

	return true
}

func (blockchain *Blockchain) AddBlock(data BookCheckout) {
	if len(blockchain.blocks) > 0 {
		prevBlock := blockchain.blocks[len(blockchain.blocks)-1]

		block := CreateBlock(prevBlock, data)
		log.Println("Block to be added: ", block)

		if ValidBlock(block, prevBlock) {
			blockchain.blocks = append(blockchain.blocks, block)
		}
	} else {
		data.IsGenesis = true
		block := CreateBlock(&Block{}, data)
		log.Println("Block to be added: ", block)
		blockchain.blocks = append(blockchain.blocks, block)
	}
}

func GetBlockchain(w http.ResponseWriter, r *http.Request) {
	resp, err := json.MarshalIndent(NewBlockChain.blocks, "", " ")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		json.NewEncoder(w).Encode(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(resp))
}

func WriteBlock(w http.ResponseWriter, r *http.Request) {
	var bookCheckoutItem BookCheckout

	if err := json.NewDecoder(r.Body).Decode(&bookCheckoutItem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	log.Println("bookCheckoutItem to be added: ", bookCheckoutItem)
	NewBlockChain.AddBlock(bookCheckoutItem)

	resp, err := json.MarshalIndent(bookCheckoutItem, "", " ")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func NewBook(w http.ResponseWriter, r *http.Request) {
	var book Book

	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	// Generating a MD5 hash
	// https://stackoverflow.com/questions/2377881/how-to-get-a-md5-hash-from-a-string-in-golang
	h := md5.New()

	// book.ID is the MD5 hash of book.ISBN+book.PublishDate
	io.WriteString(h, book.ISBN+book.PublishDate)
	// book.ID = fmt.Sprintf("%x", h.Sum(nil))
	book.ID = hex.EncodeToString(h.Sum(nil))

	// https://stackoverflow.com/questions/44495489/what-is-the-difference-between-json-marshal-and-json-marshalindent-using-go
	/*
		type Entry struct {
			Key string `json:"key"`
		}

		e := Entry{Key: "value"}
		res, err := json.Marshal(e)
		fmt.Println(string(res), err)

		res, err = json.MarshalIndent(e, "", "  ")
		fmt.Println(string(res), err)


		The output is (try it on the Go Playground):

		{"key":"value"} <nil>
		{
		"key": "value"
		} <nil>
	*/
	resp, err := json.MarshalIndent(book, "", " ")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, BookCheckout{IsGenesis: true})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{}}

	// return &Blockchain{[]*Block{GenesisBlock()}}
}

func main() {

	NewBlockChain = NewBlockchain()

	r := mux.NewRouter()
	r.HandleFunc("/", GetBlockchain).Methods("GET")
	r.HandleFunc("/", WriteBlock).Methods("POST")
	r.HandleFunc("/new", NewBook).Methods("POST")

	// if there is any block in the blockchain when the app runs, print it out
	go func() {
		for _, block := range NewBlockChain.blocks {
			fmt.Printf("Previous hash: %x\n", block.PrevHash)
			bytes, err := json.MarshalIndent(block.Data, "", " ")
			if err != nil {
				log.Panic(err)
				return
			}
			fmt.Printf("Data: %v\n", string(bytes))
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Println()
		}
	}()

	log.Println("Listening on port 8000")

	log.Fatal(http.ListenAndServe(":8000", r))
}
