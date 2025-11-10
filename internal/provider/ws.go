package provider

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	
	"github.com/b-j-roberts/foc-engine/internal/config"
	"github.com/gorilla/websocket"
)

func ConnectStarknetWebSocket(processStarknetEventData func([]byte)) (*websocket.Conn, error) {
	// Connect to the WebSocket server
	fmt.Println("=== WebSocket Connection Debug ===")
	var wsURL string
	rpcHost := config.Conf.Rpc.Host
	fmt.Printf("DEBUG: Original RPC Host from config: %s\n", rpcHost)
	
	// Handle WebSocket URLs or convert HTTP URLs to WebSocket
	if strings.HasPrefix(rpcHost, "wss://") || strings.HasPrefix(rpcHost, "ws://") {
		// Already a WebSocket URL
		wsURL = rpcHost
		fmt.Printf("DEBUG: RPC Host already has WebSocket protocol, using as-is\n")
	} else if strings.HasPrefix(rpcHost, "https://") {
		wsURL = strings.Replace(rpcHost, "https://", "wss://", 1)
		fmt.Printf("DEBUG: Converted HTTPS to WSS\n")
	} else if strings.HasPrefix(rpcHost, "http://") {
		wsURL = strings.Replace(rpcHost, "http://", "ws://", 1)
		fmt.Printf("DEBUG: Converted HTTP to WS\n")
	} else {
		// Assume it's a hostname/path and prepend wss://
		wsURL = "wss://" + rpcHost
		fmt.Printf("DEBUG: Prepending wss:// to hostname/path\n")
	}
	
	fmt.Printf("DEBUG: Constructed WebSocket URL: %s\n", wsURL)
	
	u, err := url.Parse(wsURL)
	if err != nil {
		fmt.Printf("ERROR: Failed to parse WebSocket URL '%s': %v\n", wsURL, err)
		StarknetProvider = &Provider{
			RpcHost:       config.Conf.Rpc.Host,
			WebSocketConn: nil,
		}
		return nil, fmt.Errorf("failed to parse WebSocket URL: %v", err)
	}
	
	fmt.Printf("DEBUG: Parsed URL - Scheme: %s, Host: %s, Path: %s, RawQuery: %s\n", 
		u.Scheme, u.Host, u.Path, u.RawQuery)
	fmt.Printf("DEBUG: Full parsed URL string: %s\n", u.String())
	
	// Add headers for WebSocket connection
	headers := make(map[string][]string)
	origin := "https://" + u.Host
	headers["Origin"] = []string{origin}
	fmt.Printf("DEBUG: Setting Origin header to: %s\n", origin)
	fmt.Printf("DEBUG: Headers being sent: %+v\n", headers)
	
	fmt.Printf("DEBUG: Attempting WebSocket connection to: %s\n", u.String())
	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		fmt.Printf("ERROR: WebSocket connection failed\n")
		fmt.Printf("ERROR: URL attempted: %s\n", u.String())
		fmt.Printf("ERROR: Error details: %v\n", err)
		if resp != nil {
			fmt.Printf("ERROR: HTTP Response Status: %s\n", resp.Status)
			fmt.Printf("ERROR: HTTP Response Headers: %+v\n", resp.Header)
		} else {
			fmt.Printf("ERROR: No HTTP response received (connection may have failed before handshake)\n")
		}
		StarknetProvider = &Provider{
			RpcHost:       config.Conf.Rpc.Host,
			WebSocketConn: nil,
		}
		return nil, fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	fmt.Printf("DEBUG: WebSocket connection successful!\n")
	if resp != nil {
		fmt.Printf("DEBUG: HTTP Response Status: %s\n", resp.Status)
		fmt.Printf("DEBUG: HTTP Response Headers: %+v\n", resp.Header)
	}

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Error reading message from WebSocket:", err)
				return
			}
			ProcessWebSocketMessage(message, processStarknetEventData) // TODO: Refactor func param
		}
	}()
	fmt.Println("Connected to WebSocket server at", u.String())
	fmt.Println("=== End WebSocket Connection Debug ===")

	return conn, nil
}

// TODO: Can we include more here?
type StarknetWsResponse struct {
	ID      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
}

func ProcessWebSocketMessage(message []byte, processStarknetEventData func([]byte)) {
	var response StarknetWsResponse
	err := json.Unmarshal(message, &response)
	if err != nil {
		fmt.Println("Error unmarshalling WebSocket message:", err)
		return
	}
	switch response.Method {
	case "":
		fmt.Println("Received empty msg:", string(message))
	case "starknet_subscribeNewHeads":
		// TODO
		fmt.Println("Received new head subscription message:", string(message))
	case "starknet_subscriptionEvents":
		processStarknetEventData(message)
	default:
		fmt.Println("Unknown WebSocket message method:", response.Method)
	}
}

func SubscribeNewHeads() {
	call := StarknetRpcCall{
		ID:      1,
		Jsonrpc: "2.0",
		Method:  "starknet_subscribeNewHeads",
		Params:  map[string]interface{}{},
	}
	// Convert the call to JSON
	callBytes, err := json.Marshal(call)
	if err != nil {
		fmt.Println("Error marshalling call to JSON:", err)
		return
	}

	err = StarknetProvider.WebSocketConn.WriteMessage(websocket.TextMessage, callBytes)
	if err != nil {
		fmt.Println("Error writing message to WebSocket:", err)
		return
	}
	fmt.Println("Message sent to WebSocket:", call)
}

func SubscribeEvents(address string) error {
	var startingBlockNumber int = 0
	if config.Conf.Indexer.StartAt != nil {
		startingBlockNumber = *config.Conf.Indexer.StartAt
	}
	call := StarknetRpcCall{
		ID:      1,
		Jsonrpc: "2.0",
		Method:  "starknet_subscribeEvents",
		Params: map[string]interface{}{
			"block_id": map[string]interface{}{
				"block_number": startingBlockNumber,
			},
			"from_address": address,
		},
	}
	// Convert the call to JSON
	callBytes, err := json.Marshal(call)
	if err != nil {
		fmt.Println("Error marshalling call to JSON:", err)
		return err
	}

	if StarknetProvider.WebSocketConn == nil {
		fmt.Println("WebSocket connection is nil")
		return fmt.Errorf("WebSocket connection is nil")
	}
	err = StarknetProvider.WebSocketConn.WriteMessage(websocket.TextMessage, callBytes)
	if err != nil {
		fmt.Println("Error writing message to WebSocket:", err)
		return err
	}
	fmt.Println("Message sent to WebSocket:", call)

	return nil
}
