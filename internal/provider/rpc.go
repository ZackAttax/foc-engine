package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"

	"github.com/b-j-roberts/foc-engine/internal/config"
)

type StarknetRpcCall struct {
	ID      int         `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type StarknetRpcResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

/*
Sample curl command to replicate

curl --location 'http://localhost:5050' \               ✔

	--header 'accept: application/json' \
	--header 'content-type: application/json' \
	--data '{
	    "id": 1,
	    "jsonrpc": "2.0",
	    "method": "starknet_blockNumber",
	    "params": []
	}'
	{"jsonrpc":"2.0","id":1,"result":1}
*/
func GetStarknetLatestBlockNumber() (uint64, error) {
	// Create a new StarknetRpcCall object
	call := StarknetRpcCall{
		ID:      1,
		Jsonrpc: "2.0",
		Method:  "starknet_blockNumber",
		Params:  []interface{}{},
	}

	// Marshal the call to JSON
	jsonData, err := json.Marshal(call)
	if err != nil {
		return 0, err
	}

	// Send the request to the Starknet RPC endpoint
	url := "http://" + config.Conf.Rpc.Host
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Unmarshal the response into a StarknetRpcResponse object
	var response StarknetRpcResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, err
	}

	// Check for errors in the response
	if response.Error != nil {
		return 0, fmt.Errorf("error from server: %v", response.Error)
	}

	// Extract the block number from the result
	blockNumber, ok := response.Result.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid result format")
	}

	return uint64(blockNumber), nil
}

type ContractClass struct {
	// TODO: SierraProgram []string `json:"sierra_program"`
	// TODO: SierraProgramDebugInfo
	// TODO: ContractClassVersion
	// TODO: EntryPointsByType
	Abi []interface{} `json:"abi"`
}

func GetStarknetClassAt(address string) (*ContractClass, error) {
	// Create a new StarknetRpcCall object
	call := StarknetRpcCall{
		ID:      1,
		Jsonrpc: "2.0",
		Method:  "starknet_getClassAt",
		Params: []interface{}{
			"latest",
			address,
		},
	}

	// Marshal the call to JSON
	jsonData, err := json.Marshal(call)
	if err != nil {
		return nil, err
	}

	// Send the request to the Starknet RPC endpoint
	url := "https://" + config.Conf.Rpc.Host
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Example:
	// "abi": "[{\"type\":\"impl\",\"name\":\"Point\",\"size\":1,\"offset\":0,\"members\":[{\"type\":\"felt\",\"name\":\"x\"}]}]" ->
	// "abi": [{"type":"impl","name":"Point","size":1,"offset":0,"members":[{"type":"felt","name":"x"}]}]
	// Remove all \ characters from the response body
	body = bytes.ReplaceAll(body, []byte("\\\""), []byte("\""))
	// Remove quotes around abi
	body = bytes.ReplaceAll(body, []byte("\"["), []byte("["))
	body = bytes.ReplaceAll(body, []byte("]\""), []byte("]"))

	// Unmarshal the response into a StarknetRpcResponse object
	var response StarknetRpcResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	// Check for errors in the response
	if response.Error != nil {
		return nil, fmt.Errorf("error from server: %v", response.Error)
	}

	// Extract the contract class from the result
	contractClass := response.Result

	// TODO
	// Output the contract class to a file for debugging
	filename := fmt.Sprintf("contract_class_%s.json", address)
	contractClassJson, err := json.MarshalIndent(contractClass, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling contract class to JSON:", err)
		return nil, err
	}
	err = os.WriteFile(filename, contractClassJson, 0644)
	if err != nil {
		fmt.Println("Error writing contract class to file:", err)
		return nil, err
	}

	// Convert the contract class to a ContractClass object
	abi, ok := contractClass.(map[string]interface{})["abi"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid result format, abi not found")
	}
	contractClassObj := &ContractClass{
		Abi: abi,
	}
	return contractClassObj, nil
}

func Mint(address string, amount *big.Int, unit string) error {
	// Create a new request body
	requestBody := map[string]interface{}{
		"address": address,
		"amount":  amount,
		"unit":    unit,
	}

	// Marshal the request body to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	// Send the request to the mint endpoint
	url := "http://" + config.Conf.Rpc.Host + "/mint"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to mint: %s", resp.Status)
	}

	return nil
}
