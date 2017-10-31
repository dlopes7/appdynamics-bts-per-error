package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/dlopes7/go-appdynamics-rest-api/appdrest"
)

var client *appdrest.Client
var wg sync.WaitGroup

// getTimeRanges returns a list of start and end time ranges ex: [1509473942000 1509475142000] [1509472741999 1509473941999]
func getTimeRanges(numberOfRequests int, eachRequestMinutes int) [][2]time.Time {
	times := make([][2]time.Time, numberOfRequests)

	end := time.Now().Unix()
	distance := int64(eachRequestMinutes * 60)

	for i := 0; i < numberOfRequests; i++ {
		start := end - distance
		times[i] = [2]time.Time{time.Unix(start, 0), time.Unix(end, 0)}
		end = start - 1
	}
	return times
}

func getErrorSnapshots(app int, timeInMinutes int) []*appdrest.Snapshot {
	var allSnaps []*appdrest.Snapshot

	fmt.Println("Calculating all time ranges...")
	eachRequestMinutes := 20
	numberOfRequests := timeInMinutes / eachRequestMinutes
	if numberOfRequests < 1 {
		numberOfRequests = 1
	}
	timeRanges := getTimeRanges(numberOfRequests, eachRequestMinutes)

	maxGoroutines := 20
	guard := make(chan int, maxGoroutines)

	fmt.Printf("Getting all snapshots for the last %d minutes, please wait...\n", timeInMinutes)
	application, err := client.Application.GetApplication(strconv.Itoa(app))
	if err != nil {
		panic(err.Error())
	}

	snapsFilters := &appdrest.SnapshotFilters{
		ErrorOccurred: true,
		NeedProps:     true,
	}
	for i, times := range timeRanges {
		fmt.Printf("%v%%\tGetting snapshots from %v to %v...\n", 100*(1+i)/len(timeRanges), times[0], times[1])
		wg.Add(1)
		guard <- 1
		go func(i int, times [2]time.Time) {
			defer wg.Done()
			snaps, err := client.Snapshot.GetSnapshots(application.ID, appdrest.TimeBETWEENTIMES, timeInMinutes, times[0], times[1], snapsFilters)
			if err != nil {
				panic(err.Error())
			}
			allSnaps = append(allSnaps, snaps...)
			<-guard
		}(i, times)
	}
	fmt.Println("Waiting for all the requests to finish....")
	wg.Wait()
	return allSnaps
}

func getAllBts(app int) map[int]string {
	fmt.Println("Getting the list of all Business Transactions, please wait...")
	mapBTs := make(map[int]string)
	bts, err := client.BusinessTransaction.GetBusinessTransactions(app)
	if err != nil {
		panic(err.Error())
	}

	for _, bt := range bts {
		mapBTs[bt.ID] = bt.Name
	}
	return mapBTs
}

func getBTsPerError(app int, minutes int) {

	mapBTs := getAllBts(app)
	btsPerError := make(map[string]map[string]int)
	snaps := getErrorSnapshots(app, minutes)
	for _, snap := range snaps {
		for _, errorDetail := range snap.ErrorDetails {
			if btsPerError[errorDetail.Name] != nil {
				btsPerError[errorDetail.Name][mapBTs[snap.BusinessTransactionID]]++
			} else {
				btsPerError[errorDetail.Name] = map[string]int{mapBTs[snap.BusinessTransactionID]: 0}
			}
		}
	}
	jsonString, err := json.MarshalIndent(btsPerError, "", "    ")
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("Writing the results to JSON")
	err = ioutil.WriteFile("resultado.json", jsonString, 0644)
	if err != nil {
		panic(err.Error())
	}

}

func getControllersFromJSON() *appdrest.Controller {
	file := "./conf.json"
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err.Error())
	}

	var controller *appdrest.Controller
	err = json.Unmarshal(raw, &controller)
	if err != nil {
		panic(err.Error())
	}
	return controller
}

func main() {
	controller := getControllersFromJSON()

	client = appdrest.NewClient(controller.Protocol, controller.Host, controller.Port, controller.User, controller.Password, controller.Account)
	appID := flag.Int("app", 3042, "The Application ID")
	minutes := flag.Int("minutes", 1440, "Number of minutes to process")
	flag.Parse()

	getBTsPerError(*appID, *minutes)

}
