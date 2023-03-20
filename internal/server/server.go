package main

import (
	"context"
	"encoding/json"
	"io"
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
	ResponseString interface{}
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
	var lockActionString string;

	switch request.LockAction {
	case remoteActionV1.LockAction_LOCK_ACTION_LOCK:
		lockActionString = "remotelock"
	case remoteActionV1.LockAction_LOCK_ACTION_UNLOCK:
		lockActionString = "remoteunlock"
	}

	log.Printf("processing %v request", lockActionString)

	res, err := toggleLock(
		Auth{Username: request.Username, JWTToken: request.JwtToken, PIN: request.Pin},
		Vehicle{RegistrationID: request.RegistrationId, VIN: request.Vin, Generation: request.Generation},
		lockActionString,
	)

	if err != nil {
		log.Printf("failed to toggle lock with error: %v", err)
		return &remoteActionV1.RemoteActionResponse{}, err
	}

	log.Printf("%v request processed successfully", lockActionString)

	if obj, ok := res.ResponseString.(*remoteActionV1.ResponseString); ok {
		return &remoteActionV1.RemoteActionResponse{
			Result: res.Result,
			FailMsg: res.FailMsg,
			ResponseString: obj,
		}, nil
	} else {
		return &remoteActionV1.RemoteActionResponse{
			Result: res.Result,
			FailMsg: res.FailMsg,
			ResponseString: &remoteActionV1.ResponseString{
				ErrorSubCode: "",
				SystemName: "",
				FunctionName: "",
				ErrorMessage: "",
				ErrorCode: 0,
				ServiceName: "",
			},
		}, nil
	}
}

func toggleLock(auth Auth, vehicle Vehicle, lockAction string) (RemoteActionResponse, error) {
	req, err := http.NewRequest("POST", "https://owners.hyundaiusa.com/bin/common/remoteAction", nil)
	if err != nil {
		log.Println("error initiating remote action req: ", err)
		return RemoteActionResponse{}, err
	}
	setReqHeaders(req, auth)
	q := req.URL.Query()
	q.Add("username", auth.Username)
	q.Add("token", auth.JWTToken)
	q.Add("service", lockAction)
	q.Add("url", "https://owners.hyundaiusa.com/us/en/page/blue-link.html")
	q.Add("regId", vehicle.RegistrationID)
	q.Add("vin", vehicle.VIN)
	q.Add("gen", vehicle.Generation)
	if lockAction == "remoteunlock" {
		q.Add("pin", auth.PIN)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to create request with error: %v", err)
		return RemoteActionResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response with error: %v", err)
		return RemoteActionResponse{}, err
	}

	var r RemoteActionResponse
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		log.Printf("failed to decode response with error: %v", err)
		return RemoteActionResponse{}, err
	}

	return r, nil
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