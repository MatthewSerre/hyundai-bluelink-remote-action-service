package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	remote_action_v1 "github.com/MatthewSerre/hyundai-bluelink-protobufs/gen/go/protos/remote_action/v1"
	remote_actionv1 "github.com/MatthewSerre/hyundai-bluelink-protobufs/gen/go/protos/remote_action/v1"
	"google.golang.org/grpc"
)

var addr string = "0.0.0.0:50053"

type Server struct {
	remote_actionv1.RemoteActionServiceServer
}

type Auth struct {
	Username   string
	PIN        string
	JWT_Token  string
}

type Vehicle struct {
	RegistrationID string
	VIN string
	Generation string
}

type RemoteActionResponse struct {
	Result string
	FailMsg string
	// ResponseString *remote_actionv1.ResponseString
	ResponseString string
}

func main() {
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.Fatalf("Failed to listen on: %v\n", err)
	}

	log.Printf("Listening on %s\n", addr)

	s := grpc.NewServer()
	remote_actionv1.RegisterRemoteActionServiceServer(s, &Server{})

	if err = s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v\n", err)
	}
}

func (s *Server) ToggleLock(context context.Context, request *remote_actionv1.ToggleLockRequest) (*remote_actionv1.RemoteActionResponse, error) {
	log.Println("Processing toggle lock request...")

	res, err := toggleLock(
		Auth{Username: request.Username, JWT_Token: request.JwtToken, PIN: request.Pin},
		Vehicle{RegistrationID: request.RegistrationId, VIN: request.Vin, Generation: request.Generation},
		request.LockAction,
	)

	if err != nil {
		log.Println("error toggling lock status:", err)
		return &remote_actionv1.RemoteActionResponse{}, err
	}

	log.Println("Toggle lock request processed successfully!")

	log.Printf("%v\n", res.ResponseString)

	var x *remote_actionv1.ResponseString
	x = &remote_actionv1.ResponseString{
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

	return &remote_actionv1.RemoteActionResponse{
		Result: res.Result,
		FailMsg: res.FailMsg,
		ResponseString: x,
	}, nil
}

func toggleLock(auth Auth, vehicle Vehicle, lockAction remote_action_v1.LockAction) (RemoteActionResponse, error) {
	req, err := http.NewRequest("POST", "https://owners.hyundaiusa.com/bin/common/remoteAction", nil)
	if err != nil {
		log.Println("error initiating remote action req: ", err)
		return RemoteActionResponse{}, err
	}
	setReqHeaders(req, auth)
	q := req.URL.Query()
	q.Add("username", auth.Username)
	q.Add("token", auth.JWT_Token)

	var lockActionString string;

	switch lockAction {
	case remote_action_v1.LockAction_LOCK_ACTION_LOCK:
		lockActionString = "remotelock"
	case remote_action_v1.LockAction_LOCK_ACTION_UNLOCK:
		lockActionString = "remoteunlock"
	}

	if lockActionString == "remoteunlock" {
		q.Add("pin", auth.PIN)
	}
	q.Add("service", lockActionString)
	q.Add("url", "https://owners.hyundaiusa.com/us/en/page/blue-link.html")
	q.Add("regId", vehicle.RegistrationID)
	q.Add("vin", vehicle.VIN)
	q.Add("gen", vehicle.Generation)
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
	req.Header.Add("Cookie", "jwt_token=" + auth.JWT_Token + "; s_name=" + auth.Username)
}