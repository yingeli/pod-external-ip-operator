package imds

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Metadata struct {
	Compute Compute
}

type Compute struct {
	AzEnvironment     string
	Name              string
	ResourceGroupName string
	ResourceId        string
	SubscriptionId    string
	VmId              string
	VmScaleSetName    string
}

func GetMetadata() (Metadata, error) {
	var PTransport = &http.Transport{Proxy: nil}

	client := http.Client{Transport: PTransport}

	req, _ := http.NewRequest("GET", "http://169.254.169.254/metadata/instance", nil)
	req.Header.Add("Metadata", "True")

	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", "2019-03-11")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return Metadata{}, err
	}

	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Metadata{}, err
	}

	metadata := Metadata{}
	json.Unmarshal(resp_body, &metadata)

	return metadata, nil
}
