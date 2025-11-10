package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/b-j-roberts/foc-engine/internal/deviceverification"
	routeutils "github.com/b-j-roberts/foc-engine/routes/utils"
)

func InitDeviceVerificationRoutes() {
	http.HandleFunc("/device-verification/verify-android", VerifyAndroid)
	http.HandleFunc("/device-verification/mark-android-claimed", MarkAndroidClaimed)
	http.HandleFunc("/device-verification/get-challenge", GetChallenge)
	http.HandleFunc("/device-verification/verify-ios", VerifyIOS)
	http.HandleFunc("/device-verification/mark-ios-claimed", MarkIOSClaimed)
}

// VerifyAndroidRequest represents the request body for Android verification
type VerifyAndroidRequest struct {
	IntegrityToken string `json:"integrityToken"`
	RequestHash    string `json:"requestHash"`
	AccountAddress string `json:"accountAddress"`
}

// VerifyAndroidResponse represents the response from Android verification
type VerifyAndroidResponse struct {
	DeviceRecall *deviceverification.DeviceRecall `json:"deviceRecall"`
	Verified     bool                             `json:"verified"`
	Error        string                           `json:"error,omitempty"`
}

func VerifyAndroid(w http.ResponseWriter, r *http.Request) {
	routeutils.SetupHeaders(w)

	if r.Method != http.MethodPost {
		routeutils.WriteErrorJson(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	reqBody, err := routeutils.ReadJsonBody[VerifyAndroidRequest](r)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if reqBody.IntegrityToken == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'integrityToken' field")
		return
	}

	if reqBody.AccountAddress == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'accountAddress' field")
		return
	}

	result, err := deviceverification.VerifyAndroidIntegrityToken(reqBody.IntegrityToken, "")
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := VerifyAndroidResponse{
		DeviceRecall: result.DeviceRecall,
		Verified:     result.Verified,
		Error:        result.Error,
	}

	json.NewEncoder(w).Encode(response)
}

// MarkAndroidClaimedRequest represents the request body for marking Android device as claimed
type MarkAndroidClaimedRequest struct {
	IntegrityToken string `json:"integrityToken"`
	AccountAddress string `json:"accountAddress"`
}

func MarkAndroidClaimed(w http.ResponseWriter, r *http.Request) {
	routeutils.SetupHeaders(w)

	if r.Method != http.MethodPost {
		routeutils.WriteErrorJson(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	reqBody, err := routeutils.ReadJsonBody[MarkAndroidClaimedRequest](r)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if reqBody.IntegrityToken == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'integrityToken' field")
		return
	}

	if reqBody.AccountAddress == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'accountAddress' field")
		return
	}

	err = deviceverification.MarkAndroidDeviceClaimed(reqBody.IntegrityToken)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	routeutils.WriteResultJson(w, "Device marked as claimed successfully")
}

// GetChallengeRequest represents the request body for getting a challenge
type GetChallengeRequest struct {
	AccountAddress string `json:"accountAddress"`
}

// GetChallengeResponse represents the response from challenge generation
type GetChallengeResponse struct {
	Challenge string `json:"challenge"`
}

func GetChallenge(w http.ResponseWriter, r *http.Request) {
	routeutils.SetupHeaders(w)

	if r.Method != http.MethodPost {
		routeutils.WriteErrorJson(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	reqBody, err := routeutils.ReadJsonBody[GetChallengeRequest](r)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if reqBody.AccountAddress == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'accountAddress' field")
		return
	}

	challenge, err := deviceverification.GenerateChallenge()
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Store challenge with 15 minute expiry
	expiry := time.Now().Add(15 * time.Minute)
	err = deviceverification.SaveChallenge(reqBody.AccountAddress, challenge, expiry)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := GetChallengeResponse{
		Challenge: challenge,
	}

	json.NewEncoder(w).Encode(response)
}

// VerifyIOSRequest represents the request body for iOS verification
type VerifyIOSRequest struct {
	AttestationObject string `json:"attestationObject"`
	KeyId             string `json:"keyId"`
	Challenge         string `json:"challenge"`
	AccountAddress    string `json:"accountAddress"`
}

// VerifyIOSResponse represents the response from iOS verification
type VerifyIOSResponse struct {
	HasClaimedReward bool   `json:"hasClaimedReward"`
	Verified         bool   `json:"verified"`
	Error            string `json:"error,omitempty"`
}

func VerifyIOS(w http.ResponseWriter, r *http.Request) {
	routeutils.SetupHeaders(w)

	if r.Method != http.MethodPost {
		routeutils.WriteErrorJson(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	reqBody, err := routeutils.ReadJsonBody[VerifyIOSRequest](r)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if reqBody.AttestationObject == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'attestationObject' field")
		return
	}

	if reqBody.Challenge == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'challenge' field")
		return
	}

	if reqBody.AccountAddress == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'accountAddress' field")
		return
	}

	result, err := deviceverification.VerifyIOSAttestation(
		reqBody.AttestationObject,
		reqBody.KeyId,
		reqBody.Challenge,
		reqBody.AccountAddress,
	)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := VerifyIOSResponse{
		HasClaimedReward: result.HasClaimedReward,
		Verified:         result.Verified,
		Error:            result.Error,
	}

	json.NewEncoder(w).Encode(response)
}

// MarkIOSClaimedRequest represents the request body for marking iOS device as claimed
type MarkIOSClaimedRequest struct {
	DeviceToken    string `json:"deviceToken"`
	AccountAddress string `json:"accountAddress"`
}

func MarkIOSClaimed(w http.ResponseWriter, r *http.Request) {
	routeutils.SetupHeaders(w)

	if r.Method != http.MethodPost {
		routeutils.WriteErrorJson(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	reqBody, err := routeutils.ReadJsonBody[MarkIOSClaimedRequest](r)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if reqBody.DeviceToken == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'deviceToken' field")
		return
	}

	if reqBody.AccountAddress == "" {
		routeutils.WriteErrorJson(w, http.StatusBadRequest, "Missing 'accountAddress' field")
		return
	}

	err = deviceverification.MarkIOSDeviceClaimed(reqBody.DeviceToken)
	if err != nil {
		routeutils.WriteErrorJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	routeutils.WriteResultJson(w, "Device marked as claimed successfully")
}
