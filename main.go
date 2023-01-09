package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Blockchain struct {
	blocks []*Block
}

type Block struct {
	Index     int
	Data      Book
	Timestamp string
	Hash      string
	PrevHash  string
}

type Book struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Author       string `json:"author"`
	PublishDate  string `json:"publish_date"`
	ISBN         string `json:"isbn"`
	User         string `json:"user"`
	CheckoutDate string `json:"checkout_date"`
}

var (
	NewBlockChain           *Blockchain = NewBlockchain()
	isGenesisBlockGenerated             = false
	mutex                               = &sync.RWMutex{}
)

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{}}
}

func (block *Block) ValidateHash(hash string) bool {
	block.GenerateHash()
	return block.Hash == hash
}

func (block *Block) GenerateHash() {
	bytes, err := json.Marshal(block.Data)

	if err != nil {
		log.Panic(err)
		return
	}

	// concatenate all of the data from the block
	data := strconv.Itoa(block.Index) + block.Timestamp + string(bytes) + block.PrevHash

	// hash the data using sha256
	hash := sha256.New()
	hash.Write([]byte(data))
	block.Hash = hex.EncodeToString(hash.Sum(nil))
}

func CreateBlock(prevBlock *Block, book Book) *Block {
	// empty block
	block := &Block{}

	// popoulate block dat
	block.Index = prevBlock.Index + 1
	block.PrevHash = prevBlock.Hash
	block.Timestamp = time.Now().String()
	block.Data = book
	block.GenerateHash()

	return block
}

func ValidBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if prevBlock.Index+1 != block.Index {
		return false
	}

	if !(block.ValidateHash(block.Hash)) {
		return false
	}

	return true
}

func (blockchain *Blockchain) AddBlock(book Book) {
	if isGenesisBlockGenerated {
		prevBlock := blockchain.blocks[len(blockchain.blocks)-1]

		block := CreateBlock(prevBlock, book)
		log.Println("Block to be added: ", block)

		if ValidBlock(block, prevBlock) {
			mutex.Lock()
			blockchain.blocks = append(blockchain.blocks, block)
			mutex.Unlock()
		}
	} else {
		block := CreateBlock(&Block{Index: -1}, book)
		log.Println("Block to be added: ", block)
		mutex.Lock()
		blockchain.blocks = append(blockchain.blocks, block)
		isGenesisBlockGenerated = true
		mutex.Unlock()
	}
}

func GetBlockchain(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	blocks := NewBlockChain.blocks
	mutex.RUnlock()
	resp, err := json.MarshalIndent(blocks, "", " ")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		json.NewEncoder(w).Encode(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(resp))
}

func NewBook(w http.ResponseWriter, r *http.Request) {
	var book Book

	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	book.ID = GenerateMD5Hash(book.ISBN + book.PublishDate)

	NewBlockChain.AddBlock(book)

	returnDto := map[string]interface{}{
		"message": "book has been added to the library blockchain",
	}

	// https://stackoverflow.com/questions/44495489/what-is-the-difference-between-json-marshal-and-json-marshalindent-using-go
	res, err := json.Marshal(returnDto)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func GenerateMD5Hash(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func init() {
	log.SetPrefix("[LOG] ")
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Llongfile)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", GetBlockchain).Methods("GET")
	r.HandleFunc("/new", NewBook).Methods("POST")

	log.Println("Listening on port 8000")

	log.Fatal(http.ListenAndServe(":8000", r))
}
