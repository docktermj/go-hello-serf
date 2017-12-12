package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/serf/serf"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const MembersToNotify = 2

// ----------------------------------------------------------------------------
// Exemplar "internal" data that is shared
// ----------------------------------------------------------------------------

type oneAndOnlyNumber struct {
	Number     int `json:"number"`
	Generation int `json:"generation"`
	numMutex   sync.RWMutex
}

func InitTheNumber(val int) *oneAndOnlyNumber {
	return &oneAndOnlyNumber{
		Number: val,
	}
}

func (n *oneAndOnlyNumber) setValue(newVal int) {
	n.numMutex.Lock()
	defer n.numMutex.Unlock()
	n.Number = newVal
	n.Generation = n.Generation + 1
}

func (n *oneAndOnlyNumber) getValue() (int, int) {
	n.numMutex.RLock()
	defer n.numMutex.RUnlock()
	return n.Number, n.Generation
}

func (n *oneAndOnlyNumber) notifyValue(curVal int, curGeneration int) bool {
	if curGeneration > n.Generation {
		n.numMutex.Lock()
		defer n.numMutex.Unlock()
		n.Generation = curGeneration
		n.Number = curVal
		return true
	}
	return false
}

// ----------------------------------------------------------------------------
// Serf
// ----------------------------------------------------------------------------

// Setup the Serf Cluster
func setupSerfCluster(advertiseAddr string, clusterAddr string, eventChannel chan<- serf.Event) (*serf.Serf, error) {

	// Configuration values.

	configuration := serf.DefaultConfig()
	configuration.Init()
	configuration.MemberlistConfig.AdvertiseAddr = advertiseAddr
	configuration.EventCh = eventChannel

	// Create the Serf cluster with the configuration.

	cluster, err := serf.Create(configuration)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't create cluster")
	}

	// Try to join an existing Serf cluster.  If not, start a new cluster.

	_, err = cluster.Join([]string{clusterAddr}, true)
	if err != nil {
		log.Printf("Couldn't join cluster, starting own: %v\n", err)
	}

	return cluster, nil
}

// Get a list of members in the cluster.
func getClusterMembers(cluster *serf.Serf) []serf.Member {
	var result []serf.Member

	// Get all members in all states.

	members := cluster.Members()

	// Filter list. Don't add this instance nor failed instances.

	for _, member := range members {
		if member.Name != cluster.LocalMember().Name && member.Status == serf.StatusAlive {
			result = append(result, member)
		}
	}
	return result
}

// Notify a single member via HTTP request.
func notifyMember(ctx context.Context, addr string, db *oneAndOnlyNumber) error {
	val, gen := db.getValue()
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v:8080/notify/%v/%v?notifier=%v", addr, val, gen, ctx.Value("name")), nil)
	if err != nil {
		return errors.Wrap(err, "Couldn't create request")
	}
	req = req.WithContext(ctx)
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Couldn't make request")
	}
	return nil
}

// Given a list of serf members, send a message to "MembersToNotify"-random members.
func notifyMembers(ctx context.Context, otherMembers []serf.Member, db *oneAndOnlyNumber) {
	g, ctx := errgroup.WithContext(ctx)

	if len(otherMembers) <= 2 {
		for _, member := range otherMembers {
			curMember := member
			g.Go(func() error {
				return notifyMember(ctx, curMember.Addr.String(), db)
			})
		}
	} else {
		randIndex := rand.Int() % len(otherMembers)
		for i := 0; i < MembersToNotify; i++ {
			curIndex := i
			g.Go(func() error {
				return notifyMember(
					ctx,
					otherMembers[(randIndex+curIndex)%len(otherMembers)].Addr.String(),
					db)
			})
		}
	}

	err := g.Wait()
	if err != nil {
		log.Printf("Error when notifying other members: %v", err)
	}
}

func reverse(s string) string {
	var result string
	for i := len(s) - 1; i >= 0; i-- {
		result += string(s[i])
	}
	return result
}

// Example query responses.
func queryResponse(event serf.Event) {
	result := ""
	query := event.String()
	responder := event.(*serf.Query)
	switch query {
	case "query: bob":
		result = "Bob was here"
	case "query: mary":
		result = "Mary was here"
	case "query: time":
		result = time.Now().String()
	}
	responder.Respond([]byte(result))
}

// Handle any of the Serf event types.
func serfEventHandler(event serf.Event) {
	switch event.EventType() {
	case serf.EventMemberFailed:
		log.Printf("EventMemberFailed: %s\n", event.String())
	case serf.EventMemberJoin:
		log.Printf("EventMemberJoin: %s\n", event.String())
	case serf.EventMemberLeave:
		log.Printf("EventMemberLeave: %s\n", event.String())
	case serf.EventMemberReap:
		log.Printf("EventMemberReap: %s\n", event.String())
	case serf.EventMemberUpdate:
		log.Printf("EventMemberUpdate: %s\n", event.String())
	case serf.EventQuery:
		log.Printf("EventQuery: %s\n", event.String())
		queryResponse(event)
	case serf.EventUser:
		log.Printf("EventUser: %s\n", event.String())
	default:
		log.Printf("[WARN] on: Unhandled Serf Event: %#v", event)
	}
}

// ----------------------------------------------------------------------------
// HTTP
// ----------------------------------------------------------------------------

// Get the value in the database.
func httpGet(response http.ResponseWriter, request *http.Request, database *oneAndOnlyNumber) {
	myJson, _ := json.Marshal(database)
	fmt.Fprintf(response, "%s", myJson)
}

//  Set the value in the database.
func httpSet(response http.ResponseWriter, request *http.Request, database *oneAndOnlyNumber) {
	vars := mux.Vars(request)
	newVal, err := strconv.Atoi(vars["newVal"])
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(response, "%v", err)
		return
	}
	database.setValue(newVal)
	fmt.Fprintf(response, "%v", newVal)
}

// Notify other Serf members of a change in value.
func httpNotify(response http.ResponseWriter, request *http.Request, database *oneAndOnlyNumber) {
	vars := mux.Vars(request)
	curVal, err := strconv.Atoi(vars["curVal"])
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(response, "%v", err)
		return
	}
	curGeneration, err := strconv.Atoi(vars["curGeneration"])
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(response, "%v", err)
		return
	}

	if changed := database.notifyValue(curVal, curGeneration); changed {
		log.Printf(
			"NewVal: %v Gen: %v Notifier: %v",
			curVal,
			curGeneration,
			request.URL.Query().Get("notifier"))
	}
	response.WriteHeader(http.StatusOK)
}

// List the available HTTP routes and map to functions.
func httpRouter(database *oneAndOnlyNumber) {
	go func() {
		router := mux.NewRouter()
		router.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) { httpGet(w, r, database) })
		router.HandleFunc("/set/{newVal}", func(w http.ResponseWriter, r *http.Request) { httpSet(w, r, database) })
		router.HandleFunc("/notify/{curVal}/{curGeneration}", func(w http.ResponseWriter, r *http.Request) { httpNotify(w, r, database) })
		log.Fatal(http.ListenAndServe(":8080", router))
	}()
}

// ----------------------------------------------------------------------------
// HTTP
// ----------------------------------------------------------------------------

func main() {

	// Create a channel to receive Serf events.

	eventChannel := make(chan serf.Event, 256)

	// Initialize or join Serf cluster.

	serfCluster, err := setupSerfCluster(
		os.Getenv("ADVERTISE_ADDR"),
		os.Getenv("CLUSTER_ADDR"),
		eventChannel)
	if err != nil {
		log.Fatal(err)
	}
	defer serfCluster.Leave()

	// Initialized "internal data".

	theOneAndOnlyNumber := InitTheNumber(0)
	httpRouter(theOneAndOnlyNumber)

	// Create a base context with hostname, if possible.

	ctx := context.Background()
	if name, err := os.Hostname(); err == nil {
		ctx = context.WithValue(ctx, "name", name)
	}

	// Set the time between "ticks".

	debugDataPrinterTicker := time.Tick(time.Second * 15)

	numberBroadcastTick := time.Second * 2
	numberBroadcastTicker := time.Tick(numberBroadcastTick)

	// Handle "ticks" and events.

	for {
		select {

		// Handle events.

		case event := <-eventChannel:
			serfEventHandler(event)

		// Notification among serf members.

		case <-numberBroadcastTicker:
			members := getClusterMembers(serfCluster)
			ctx, _ := context.WithTimeout(ctx, numberBroadcastTick)
			go notifyMembers(ctx, members, theOneAndOnlyNumber)

		// Internal debugging.

		case <-debugDataPrinterTicker:
			members := serfCluster.Members()
			for memberNumber, member := range members {
				log.Printf("Member %d: %+v\n", memberNumber, member)
			}
			curVal, curGen := theOneAndOnlyNumber.getValue()
			log.Printf("State: %v Generation: %v\n", curVal, curGen)
		}
	}
}
