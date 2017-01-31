package saltboot

import (
	"encoding/json"
	"github.com/hortonworks/salt-bootstrap/saltboot/cautils"
	"github.com/hortonworks/salt-bootstrap/saltboot/model"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

type Credentials struct {
	Clients
	PublicIP *string `json:"PublicIP" yaml:"PublicIP"`
}

func ClientCredsHandler(w http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	var credentials Credentials
	err := decoder.Decode(&credentials)
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR] couldn't decode json: %s", err.Error())
		model.Response{Status: err.Error()}.WriteBadRequestHttp(w)
		return
	}

	// mkdir if needed
	log.Printf("[CAHandler] handleClientCreds executed")
	w.Header().Set("Content-Type", "application/json")
	pubIp := credentials.PublicIP

	if cautils.IsPathExisting("/etc/certs") == false {
		if err := os.Mkdir("/etc/certs", 0755); err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
	}
	caResp, _ := http.Get("http://" + credentials.Servers[0].Address + ":7070/saltboot/ca")
	caBytes, _ := ioutil.ReadAll(caResp.Body)
	caCrt, err := cautils.NewCertificateFromPEM(caBytes)
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
		model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
		return
	}
	err = caCrt.ToPEMFile("/etc/certs/ca.crt")
	if cautils.IsPathExisting("/etc/certs/client.key") == false {
		key, err := cautils.NewKey()
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}

		err = key.ToPEMFile("/etc/certs/client.key")
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
	}
	if cautils.IsPathExisting("/etc/certs/client.csr") == false {
		key, err := cautils.NewKeyFromPrivateKeyPEMFile("/etc/certs/client.key")
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}

		csr, err := cautils.NewCertificateRequest(key, pubIp)
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
		err = csr.ToPEMFile("/etc/certs/client.csr")
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
	}
	csr, err := cautils.NewCertificateRequestFromPEMFile("/etc/certs/client.csr")
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
		model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
		return
	}
	pem, _ := csr.ToPEM()
	data := make(url.Values)
	data.Add("csr", string(pem))
	//resp, err := http.PostForm("http://" + host + "/certificates", data)
	resp, _ := http.PostForm("http://"+credentials.Servers[0].Address+":7070/saltboot/csr", data)
	crtBytes, _ := ioutil.ReadAll(resp.Body)
	crt, err := cautils.NewCertificateFromPEM(crtBytes)
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
		model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
		return
	}
	err = crt.ToPEMFile("/etc/certs/client.crt")
	model.Response{Status: "OK"}.WriteHttp(w)
	return
}

func ClientCredsDistributeHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("[ClientCredsDistributeHandler] execute distribute hostname request")

	decoder := json.NewDecoder(req.Body)
	var credentials Credentials
	err := decoder.Decode(&credentials)
	if err != nil {
		log.Printf("[ClientCredsDistributeHandler] [ERROR] couldn't decode json: %s", err)
		model.Response{Status: err.Error()}.WriteBadRequestHttp(w)
		return
	}

	user, pass := GetAuthUserPass(req)
	responses := credentials.DistributeClientCredentials(user, pass)
	cResp := model.Responses{Responses: responses}
	log.Printf("[ClientCredsDistributeHandler] distribute request executed: %s" + cResp.String())
	json.NewEncoder(w).Encode(cResp)
}

func (credentials *Credentials) DistributeClientCredentials(user string, pass string) []model.Response {
	log.Printf("[Clients.DistributeClientCredentials] Request: %v", credentials)
	credReq := Credentials{
		Clients: Clients{
			Servers: credentials.Servers,
		},
		PublicIP: credentials.PublicIP,
	}
	jsonBody, _ := json.Marshal(credReq)
	resp := distributeImpl(Distribute, []string{credentials.Servers[0].Address}, jsonBody, ClientCredsEP, user, pass)
	for _, r := range resp {
		if r.StatusCode != http.StatusOK {
			return resp
		}
	}

	credReq.PublicIP = nil
	jsonBody, _ = json.Marshal(credReq)

	return append(resp, distributeImpl(Distribute, credentials.Clients.Clients, jsonBody, ClientCredsEP, user, pass)...)
}
