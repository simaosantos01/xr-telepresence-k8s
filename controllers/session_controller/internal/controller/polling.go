package controller

import (
	"encoding/json"
	"io"
	"net/http"

	telepresencev1 "mr.telepresence/controller/api/v1"
)

// todo: update json annotations
type PollingResponse struct {
	ClientCount          int16  `json:"clients"`
	ClientCountUpdatedAt string `json:"latestChange"`
}

func (r *SessionReconciler) evaluateSession(session *telepresencev1.Session) (bool, int16, error) {
	//url := "http://" + session.Name + "-clientpolling.clientpolling." + session.Namespace + ".svc.cluster.local:8080/"
	url := "http://localhost:8080"

	resp, err := http.Get(url)
	if err != nil {
		return false, -1, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, -1, err
	}

	var pollingData PollingResponse
	if err := json.Unmarshal(body, &pollingData); err != nil {
		return false, -1, err
	}

	if pollingData.ClientCount == 0 {
		return true, 0, nil
	}

	return false, pollingData.ClientCount, nil
}
