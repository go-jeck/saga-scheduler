package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jasonlvhit/gocron"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pborman/uuid"
)

type scheduler struct {
	Router    *mux.Router
	DB        *sqlx.DB
	client    client
	metricsDB metricsDB
}

type createContainerRequestData struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Server   string `json:"server,omitempty"`
	Alias    string `json:"alias,omitempty"`
}

type client interface {
	executeOperationRequest(req *http.Request) (*operation, error)
}

type agentClient struct{}

func (a agentClient) executeOperationRequest(req *http.Request) (*operation, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var op *operation

	err = json.Unmarshal(body, &op)
	if err != nil {
		return nil, err
	}

	return op, nil
}

func (s *scheduler) initialize(user, password, dbname, host, port, sslmode string) error {
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s", user, password, dbname, host, port, sslmode)
	var err error
	s.DB, err = sqlx.Connect("postgres", connectionString)
	if err != nil {
		return err
	}

	s.Router = mux.NewRouter()
	s.Router.HandleFunc("/api/v1/container", s.createNewLxcHandler).Methods("POST")
	s.client = agentClient{}
	s.metricsDB = prometheusMetricsDB{}
	return nil
}

func (s *scheduler) startCronJob() {
	gocron.Every(1).Minute().Do(s.doCron)
}

func (s *scheduler) doCron() {
	lxds, err := getLxds(s.DB)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(lxds); i++ {
		url := fmt.Sprintf("http://%s:9200/api/v1/container", lxds[i].Address)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			panic(err)
		}

		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		response, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}

		type lxdResponse struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		var resp []lxdResponse

		err = json.Unmarshal(body, &resp)
		if err != nil {
			panic(err)
		}

		for j := 0; j < len(resp); j++ {
			err = updateStatusByName(s.DB, resp[j].Name, resp[j].Status)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (s *scheduler) run(port string) {
	log.Fatal(http.ListenAndServe(port, s.Router))
}

func (s *scheduler) createNewLxcHandler(w http.ResponseWriter, r *http.Request) {
	var data createContainerRequestData
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	defer r.Body.Close()
	lxdInstance, err := s.metricsDB.getLowestLoadLxdInstance()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = lxdInstance.getLxdByIP(s.DB)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	newLxc := lxc{
		ID:         uuid.New(),
		LxdID:      lxdInstance.ID,
		Name:       data.Name,
		Type:       data.Type,
		Alias:      data.Alias,
		IsDeployed: 1,
	}

	err = newLxc.insertLxc(s.DB)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	op, err := s.createNewLxc(data, lxdInstance.Address)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	op.LxcID = newLxc.ID
	err = op.insertOperation(s.DB)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, op)
	return
}

func (s *scheduler) createNewLxc(data createContainerRequestData, lxdIPAddress string) (op *operation, err error) {
	url := fmt.Sprintf("http://%s:9200/api/v1/container", lxdIPAddress)
	payload, err := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	return s.client.executeOperationRequest(req)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
