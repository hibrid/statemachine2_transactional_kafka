package simulator

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// Messages contains a slice of Order
type Messages struct {
	Messages []Message
}

// Message contains an Message
type Message struct {
	AccountID     string `json:"accountId"`
	TransactionID string `json:"transactionId"`
	EventType     string `json:"eventType"`
}

type eventType string

func (e eventType) String() string {
	return string(e)
}

func getRandomEventType() eventType {
	eventTypes := []eventType{"ENTITLEMENT_CREATION", "ENTITLEMENT_CANCEL"}
	return eventTypes[rand.Intn(len(eventTypes))]
}

func generateRandomAccountID() string {
	return "account-" + fmt.Sprint(rand.Intn(1000000))
}

func generateRandomTransactionID() string {
	accountID, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return accountID.String()
}

func generateRandomMessage() Message {
	return Message{
		AccountID:     generateRandomAccountID(),
		TransactionID: generateRandomTransactionID(),
		EventType:     getRandomEventType().String(),
	}
}

type rate float64

func (r *rate) asFloat64() float64 {
	return float64(*r)
}

type nextMessage float64

func (r *nextMessage) asDuration() time.Duration {
	//not exact but we lose presicion converting to duration which
	//is of type int64. This should be closer to the target duration
	roundedValue := math.Round(float64(*r))
	return time.Duration(roundedValue)
}
