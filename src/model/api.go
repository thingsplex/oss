package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	BASE_URL      = "https://publicapi.oss.no/api/"
	url_auth      = BASE_URL + "Auth"
	url_token     = BASE_URL + "Token"
	url_meter     = BASE_URL + "Meter"
	url_telemetry = BASE_URL + "Telemetry/"
)

type (
	ResetToken struct {
		Token    string `json:"token"`
		DevToken string `json:"devToken,omitempty"`
	}

	Auth struct {
		AuthId       string `json:"authId"`
		Token        string `json:"token"`
		EncodedToken string `json:"encodedToken"`
	}

	Meters struct {
		Meters []struct {
			MeterNumber  string `json:"meterNumber"`
			MeterAddress struct {
				StreetAddress1 string      `json:"streetAddress1, omitempty"`
				StreetAddress2 interface{} `json:"streetAddress2, omitempty"`
				PostalArea     string      `json:"postalArea, omitempty"`
				PostCode       string      `json:"postCode, omitempty"`
				Country        string      `json:"country, omitempty"`
				Region         interface{} `json:"region, omitempty"`
				County         interface{} `json:"county, omitempty"`
			} `json:"meterAddress, omitempty"`
		} `json:"meters, omitempty"`
	}

	Telemetry []struct {
		Timestamp   time.Time `json:"timestamp"`
		ActivePower struct {
			Input struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"input"`
			Output struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"output"`
		} `json:"activePower"`
		ReactivePower struct {
			Input struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"input"`
			Output struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"output"`
		} `json:"reactivePower"`
		PhaseOne struct {
			Voltage struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"voltage"`
			Current struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"current"`
		} `json:"phaseOne"`
		PhaseTwo struct {
			Voltage struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"voltage"`
			Current struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"current"`
		} `json:"phaseTwo"`
		PhaseThree struct {
			Voltage struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"voltage"`
			Current struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"current"`
		} `json:"phaseThree"`
		CumulativeActivePower struct {
			Input struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"input"`
			Output struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"output"`
		} `json:"cumulativeActivePower"`
		CumulativeReactivePower struct {
			Input struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"input"`
			Output struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"output"`
		} `json:"cumulativeReactivePower"`
	}
)

func processHTTPResponse(resp *http.Response, err error, holder interface{}) error {
	if err != nil {
		log.Error(fmt.Errorf("API does not respond"))
		return err
	}
	defer resp.Body.Close()
	// check http return code
	if resp.StatusCode != 200 {
		//bytes, _ := ioutil.ReadAll(resp.Body)
		log.Error("Bad HTTP return code ", resp.StatusCode)
		return fmt.Errorf("Bad HTTP return code %d", resp.StatusCode)
	}
	if err = json.NewDecoder(resp.Body).Decode(holder); err != nil {
		return err
	}
	return nil
}

func (auth *Auth) GetAuthCode(email string) (string, error) {
	log.Debug("Getting auth code. User should receive email shortly.")

	body := strings.NewReader(fmt.Sprintf(`{
		"email": "%s",
		"label": "string"
	}`, email))
	log.Debug("Body: ", body)

	req, err := http.NewRequest("POST", url_auth, body)
	log.Debug("Req: ", req)

	if err != nil {
		log.Error(fmt.Errorf("Can't post login request, error: %v", err))
		return "", err
	}

	req.Header.Set("Content-Type", "application/json-patch+json")
	req.Header.Set("accept", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	log.Debug("Resp: ", resp)

	if err := processHTTPResponse(resp, err, &auth); err != nil {
		// if response topic is not set , sending back to default application event topic
		log.Error(err)
		return "", err
	}
	log.Debug("authId: ", auth.AuthId)
	return auth.AuthId, nil
}

func (auth *Auth) GetAuthToken(authId string, authSecret string) (string, error) {
	log.Debug("Getting token.")

	url := fmt.Sprintf("%s/%s/%s", url_auth, authId, authSecret)
	req, err := http.NewRequest("GET", url, nil)
	log.Debug("Req: ", req)

	if err != nil {
		log.Error(fmt.Errorf("Can't get token, error: ", err))
		return "", err
	}
	req.Header.Set("accept", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	log.Debug("Resp: ", resp)
	if err := processHTTPResponse(resp, err, &auth); err != nil {
		// if response topic is not set , sending back to default application event topic
		log.Error(err)
		return "", err
	}
	log.Debug("token: ", auth.Token)

	// Received first token. Use this to get second token.
	log.Debug("Getting second token.")

	body := strings.NewReader(fmt.Sprintf(`{
		"expires": "2022-04-15T07:03:23.532Z",
		"friendlyName": "string"
	}`))
	log.Debug("Body: ", body)

	req, err = http.NewRequest("POST", url_token, body)
	log.Debug("Req: ", req)

	if err != nil {
		log.Error(fmt.Errorf("Can't post second token request, error: %v", err))
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.Token))
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err = http.DefaultClient.Do(req)
	log.Debug("Resp: ", resp)
	if err := processHTTPResponse(resp, err, &auth); err != nil {
		// if response topic is not set , sending back to default application event topic
		log.Error(err)
		return "", err
	}
	log.Debug("EncodedToken: ", auth.EncodedToken)

	return auth.EncodedToken, nil
}

func GetMeters(accessToken string) (Meters, error) {
	meters := new(Meters)
	err := get(accessToken, url_meter, meters) // get meters
	return *meters, err
}

func GetTelemetry(accessToken string, startDate string, endDate string, meterId string, resolution int) (Telemetry, error) {
	telemetry := new(Telemetry)
	newStartDate := url.QueryEscape(startDate)
	newEndDate := url.QueryEscape(endDate)
	newResolution := url.QueryEscape(strconv.Itoa(resolution))

	url := fmt.Sprintf("%s%s/%s/%s/%s", url_telemetry, meterId, newStartDate, newEndDate, newResolution)
	err := get(accessToken, url, telemetry)
	return *telemetry, err
}

func get(accessToken string, url string, target interface{}) error {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Error(fmt.Errorf("Can't GET from ", url))
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := http.DefaultClient.Do(req)
	err = processHTTPResponse(resp, err, target)
	return err
}
