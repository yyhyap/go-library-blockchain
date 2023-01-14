# A blockchain with Proof of Work written in Go
A blockchain for library book checkout system, built using Go and Gorilla Mux HTTP router. Additionally, there is a proof of work logic for each block added to the blockchain.

<br />

## REST APIs
### Get the current blockchain
```
http://<host>:<port>
```
A GET request for getting the JSON response body for the blochchain.

### Adding a new block
```
http://<host>:<port>/new
```
A POST request for adding a new block to the blockchain. A example JSON for the request: 
```json
{
    "title": "Sample 1",
    "author": "test_author",
    "publish_date": "2022-04-16",
    "isbn": "123456",
    "user": "test_user",
    "checkout_date": "2022-04-15"
}
```

<br />

## Difficulty
In this blockchain project, the difficulty for the proof of work will increase when the amount of block added to the blockchain is increasing. The difficulty logic can be set at: 
```go
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
```
For example, the current code increase the difficulty level by 1 when 10 blocks are added to the blockchain.