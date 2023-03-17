package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	remoteActionV1 "github.com/MatthewSerre/hyundai-bluelink-protobufs/gen/go/protos/remote_action/v1"
	"google.golang.org/grpc"
)

var addr string = "0.0.0.0:50053"

type Server struct {
	remoteActionV1.RemoteActionServiceServer
}

type Auth struct {
	Username   string
	PIN        string
	JWTToken  string
}

type Vehicle struct {
	RegistrationID string
	VIN string
	Generation string
}

type RemoteActionResponse struct {
	Result string
	FailMsg string
	// this appears be a string, which makes sense given the name, but some other implementations I reviewed had it as an object; need to test more
	ResponseString string 
}

func main() {
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.Fatalf("failed to listen on: %v\n", err)
	}

	log.Printf("remote action server listening on %s\n", addr)

	s := grpc.NewServer()
	remoteActionV1.RegisterRemoteActionServiceServer(s, &Server{})

	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}

func (s *Server) ToggleLock(context context.Context, request *remoteActionV1.ToggleLockRequest) (*remoteActionV1.RemoteActionResponse, error) {
	log.Println("processing toggle lock request")

	res, err := toggleLock(
		Auth{Username: request.Username, JWTToken: request.JwtToken, PIN: request.Pin},
		Vehicle{RegistrationID: request.RegistrationId, VIN: request.Vin, Generation: request.Generation},
		request.LockAction,
	)

	if err != nil {
		log.Println("error toggling lock status:", err)
		return &remoteActionV1.RemoteActionResponse{}, err
	}

	log.Println("toggle lock request processed successfully")

	log.Printf("%v\n", res.ResponseString)

	var x *remoteActionV1.ResponseString
	x = &remoteActionV1.ResponseString{
		ErrorSubCode: "res.ResponseString.ErrorSubCode",
		SystemName: "res.ResponseString.SystemName",
		FunctionName: "res.ResponseString.FunctionName",
		ErrorMessage: "res.ResponseString.ErrorMessage",
		ErrorCode: 0,
		ServiceName: "res.ResponseString.ServiceName",
		// ErrorSubCode: res.ResponseString.ErrorSubCode,
		// SystemName: res.ResponseString.SystemName,
		// FunctionName: res.ResponseString.FunctionName,
		// ErrorMessage: res.ResponseString.ErrorMessage,
		// ErrorCode: res.ResponseString.ErrorCode,
		// ServiceName: res.ResponseString.ServiceName,
	}

	return &remoteActionV1.RemoteActionResponse{
		Result: res.Result,
		FailMsg: res.FailMsg,
		ResponseString: x,
	}, nil
}

func toggleLock(auth Auth, vehicle Vehicle, lockAction remoteActionV1.LockAction) (RemoteActionResponse, error) {
	var lockActionString string;

	switch lockAction {
	case remoteActionV1.LockAction_LOCK_ACTION_LOCK:
		lockActionString = "remotelock"
	case remoteActionV1.LockAction_LOCK_ACTION_UNLOCK:
		lockActionString = "remoteunlock"
	}

	req, err := http.NewRequest("POST", "https://owners.hyundaiusa.com/bin/common/remoteAction", nil)
	if err != nil {
		log.Println("error initiating remote action req: ", err)
		return RemoteActionResponse{}, err
	}
	setReqHeaders(req, auth)
	q := req.URL.Query()
	q.Add("username", auth.Username)
	q.Add("token", auth.JWTToken)
	q.Add("service", lockActionString)
	q.Add("url", "https://owners.hyundaiusa.com/us/en/page/blue-link.html")
	q.Add("regId", vehicle.RegistrationID)
	q.Add("vin", vehicle.VIN)
	q.Add("gen", vehicle.Generation)
	if lockActionString == "remoteunlock" {
		q.Add("pin", auth.PIN)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error creating remote lock req: ", err)
		return RemoteActionResponse{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	log.Printf("%v\n", body)
	if err != nil {
		log.Println("Error reading remote lock response: ", err)
		return RemoteActionResponse{}, err
	}
	log.Println("remote lock response: ", string(body))
	// decode body as json
	var remote_lock_response RemoteActionResponse
	err = json.Unmarshal(body, &remote_lock_response)
	if err != nil {
		log.Println("error decoding remote lock response: ", err)
		return RemoteActionResponse{}, err
	}

	log.Printf("%v\n", remote_lock_response)
	return remote_lock_response, nil
}

func setReqHeaders(req *http.Request, auth Auth) {
	// set request headers
	req.Header.Add("CSRF-Token", "undefined")
	req.Header.Add("accept-language", "en-US,en;q=0.9")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Referer", "https://owners.hyundaiusa.com/content/myhyundai/us/en/page/dashboard.html")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.2 Safari/605.1.15")
	req.Header.Add("Host", "owners.hyundaiusa.com")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Origin", "https://owners.hyundaiusa.com")
	req.Header.Add("Cookie", "JWTToken=" + auth.JWTToken + "; s_name=" + auth.Username)
}