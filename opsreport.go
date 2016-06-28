package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const url string = "https://%s/api/v1/incidents?since=%sT00%%3A00%%3A00SGT&until=%sT23%%3A59%%3A59SGT&time_zone=%s&offset=%d"

type IncidentsResponse struct {
	Incidents []Incident `json:"incidents"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
	Total     int        `json:"total"`
}

type Incident struct {
	ID             string        `json:"id"`
	IncidentNumber int           `json:"incident_number"`
	CreatedOn      time.Time     `json:"created_on"`
	Status         string        `json:"status"`
	PendingActions []interface{} `json:"pending_actions"`
	HTMLURL        string        `json:"html_url"`
	IncidentKey    string        `json:"incident_key"`
	Service        struct {
		ID          string      `json:"id"`
		Name        string      `json:"name"`
		HTMLURL     string      `json:"html_url"`
		DeletedAt   interface{} `json:"deleted_at"`
		Description string      `json:"description"`
	} `json:"service"`
	EscalationPolicy struct {
		ID        string      `json:"id"`
		Name      string      `json:"name"`
		DeletedAt interface{} `json:"deleted_at"`
	} `json:"escalation_policy"`
	AssignedToUser     interface{} `json:"assigned_to_user"`
	TriggerSummaryData struct {
		Description string `json:"description"`
	} `json:"trigger_summary_data"`
	TriggerDetailsHTMLURL string        `json:"trigger_details_html_url"`
	TriggerType           string        `json:"trigger_type"`
	LastStatusChangeOn    time.Time     `json:"last_status_change_on"`
	LastStatusChangeBy    interface{}   `json:"last_status_change_by"`
	NumberOfEscalations   int           `json:"number_of_escalations"`
	ResolvedByUser        interface{}   `json:"resolved_by_user,omitempty"`
	AssignedTo            []interface{} `json:"assigned_to"`
	Urgency               string        `json:"urgency"`
	Acknowledgers         []struct {
		At     time.Time `json:"at"`
		Object struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Email   string `json:"email"`
			HTMLURL string `json:"html_url"`
			Type    string `json:"type"`
		} `json:"object"`
	} `json:"acknowledgers,omitempty"`
}

func writeIncident(key string, incident Incident) {
	status := incident.Status
	resolution := incident.ResolvedByUser
	if status == "resolved" {
		if resolution == nil {
			resolution = "API"
		} else {
			resolution = fmt.Sprintf("resolved by: %s", incident.ResolvedByUser)
		}
	} else if status == "acknowledged" {
		var buffer bytes.Buffer
		//just print the first Acknowledger
		buffer.WriteString(fmt.Sprintf("%s - %s", incident.Acknowledgers[0].Object.Name, incident.Acknowledgers[0].At))
		resolution = buffer.String()
	} else if status == "triggered" {
		resolution = "open"
	}

	fmt.Printf("%s,%d,%s,%s,%s,%s,%s\n", key, incident.IncidentNumber, incident.TriggerSummaryData.Description, incident.CreatedOn, incident.LastStatusChangeOn, status, resolution)
}

func callApi(endpoint string, timeZone string, token string, startDate string, endDate string, offset int, respStruct *IncidentsResponse) {
	url := fmt.Sprintf(url, endpoint, startDate, endDate, timeZone, offset)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

    tk := fmt.Sprintf("Token token=%s",  token) 
    fmt.Printf("Token: ", tk)
    
	req.Header.Set("Authorization", tk)

	fmt.Println("about to call API with url:", url)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Err: %s", err)
		os.Exit(-1)

	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(contents, respStruct)

	if err != nil {
		panic(err)
	}

	//fmt.Printf("myVariable = %#v \n", respStruct)

}

//usage: ./pdreport -endpoint <<your org's pd endpoint>> -tz <<your time zone>> -token <<yourAPI Token>> -since=2016-04-28 -until=2016-04-28 
func main() {

	endpoint := flag.String("endpoint", "", "pagerduty endpoint for your organization")
	timeZone := flag.String("tz", "", "tz db timezone e.g: Singapore")
	token := flag.String("token", "", "PD assigned API token")
	startDate := flag.String("since", "", "date in format: 2016-04-27")
	endDate := flag.String("until", "", "date in format: 2016-04-27")
	flag.Parse()
	fmt.Printf("Called with params: since:%s until:%s", *startDate, *endDate)

	respStruct := new(IncidentsResponse)
	groupedByServiceMap := make(map[string][]Incident)
	offset := 0
	callcount := 1

	//API call paginates so call repeatedly until there are no items left
	for {
		fmt.Println("Starting call: ", callcount)
		callApi(*endpoint, *timeZone, *token, *startDate, *endDate, offset, respStruct)
		if len(respStruct.Incidents) == 0 {
			fmt.Println("No more items.")
			break
		}
		for _, incident := range respStruct.Incidents {
			arr, present := groupedByServiceMap[incident.Service.Name]
			if !present {
				groupedByServiceMap[incident.Service.Name] = make([]Incident, 1)
			}
			newslice := append(arr, incident)
			groupedByServiceMap[incident.Service.Name] = newslice
		}
		//fmt.Println("offset:", respStruct.Offset)
		if respStruct.Total <= respStruct.Limit {
			break
		}
		if offset == 0 {
			offset = respStruct.Limit
		} else {
			offset = respStruct.Offset + len(respStruct.Incidents)
		}
		fmt.Println("finished call: ", callcount)
		callcount++
	}

	for key, valarr := range groupedByServiceMap {
		//fmt.Printf("%#v->\n", key)
		fmt.Printf("Category %s: count: %d\n", string(key), len(groupedByServiceMap[key]))
		for _, incident := range valarr {
			writeIncident(key, incident)
			//fmt.Printf("%#v \n", incident)
			//fmt.Printf("%#v \n", incident)
		}
	}
	//fmt.Printf("map = %#v \n", groupedByServiceMap)
	//fmt.Printf("map = %#v \n", groupedByServiceMap)
}
