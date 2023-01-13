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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Blockchain struct {
	blocks []*Block
}

type Block struct {
	Index      int
	Data       *Book
	Timestamp  string
	Hash       string
	PrevHash   string
	Nonce      string
	Difficulty int
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
	NewBlockChain     *Blockchain = NewBlockchain()
	difficulty                    = 1
	difficultyCounter             = 0
	mutex                         = &sync.RWMutex{}
	difficultyMutex               = &sync.RWMutex{}
)

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{}}
}

func (block *Block) ValidateBlockHash(hash string) bool {
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
	data := strconv.Itoa(block.Index) + block.PrevHash + block.Timestamp + string(bytes) + block.Nonce + strconv.Itoa(block.Difficulty)

	// hash the data using sha256
	hash := sha256.New()
	hash.Write([]byte(data))
	block.Hash = hex.EncodeToString(hash.Sum(nil))
}

func CreateBlock(prevBlock *Block, book *Book) *Block {
	// empty block
	block := &Block{}

	// popoulate block dat
	block.Index = prevBlock.Index + 1
	block.PrevHash = prevBlock.Hash
	block.Timestamp = time.Now().String()
	block.Data = book

	difficultyMutex.RLock()
	block.Difficulty = difficulty
	difficultyMutex.RUnlock()

	for i := 0; ; i++ {
		hex := fmt.Sprintf("%x", i)
		block.Nonce = hex
		block.GenerateHash()
		if IsNewBlockHashValid(block.Hash, block.Difficulty) {
			break
		}
	}

	return block
}

func IsNewBlockHashValid(hash string, blockDifficulty int) bool {
	prefix := strings.Repeat("0", blockDifficulty)
	return strings.HasPrefix(hash, prefix)
}

func (blockchain *Blockchain) IsValidNewBlock(block *Block) bool {
	prevBlock := blockchain.blocks[len(blockchain.blocks)-1]

	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if prevBlock.Index+1 != block.Index {
		return false
	}

	if !(block.ValidateBlockHash(block.Hash)) {
		return false
	}

	return true
}

func (blockchain *Blockchain) AddBlock(book *Book) error {
	mutex.RLock()
	prevBlock := blockchain.blocks[len(blockchain.blocks)-1]
	mutex.RUnlock()

	block := CreateBlock(prevBlock, book)

	mutex.Lock()
	defer mutex.Unlock()
	if blockchain.IsValidNewBlock(block) {
		blockchain.blocks = append(blockchain.blocks, block)
		SetDifficulty()
		return nil
	} else {
		return fmt.Errorf("new block: %v is not valid, previous block: %v", block, blockchain.blocks[len(blockchain.blocks)-1])
	}
}

func SetDifficulty() {
	difficultyMutex.Lock()
	defer difficultyMutex.Unlock()
	if difficultyCounter == 9 {
		difficulty++
		difficultyCounter = 0
	} else {
		difficultyCounter++
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

func NewBook(w http.ResponseWriter, r *http.Request) {
	var book Book

	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	book.ID = GenerateMD5Hash(book.ISBN + book.PublishDate)

	err := NewBlockChain.AddBlock(&book)

	returnDto := map[string]interface{}{}

	if err != nil {
		log.Printf("error while adding block: %v", err.Error())
		returnDto["error"] = err.Error()
	} else {
		returnDto["message"] = "book has been added to the library blockchain"
	}

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
	genesisBlock := &Block{
		Index:     -1,
		Data:      nil,
		Timestamp: time.Now().String(),
		PrevHash:  "",
		Nonce:     "",
	}
	genesisBlock.GenerateHash()

	mutex.Lock()
	NewBlockChain.blocks = append(NewBlockChain.blocks, genesisBlock)
	mutex.Unlock()

	r := mux.NewRouter()
	r.HandleFunc("/", GetBlockchain).Methods("GET")
	r.HandleFunc("/new", NewBook).Methods("POST")

	log.Println("Listening on port 8000")

	log.Fatal(http.ListenAndServe(":8000", r))
}
