package getstream

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

const layout = "2006-01-02T15:04:05.999999"

// Activity is a getstream Activity
// Use it to post activities to Feeds
// It is also the response from Fetch and List Requests
type Activity struct {
	ID        string
	Actor     string
	Verb      string
	Object    string
	Target    string
	Origin    FeedID
	TimeStamp *time.Time

	ForeignID string
	Data      *json.RawMessage
	MetaData  map[string]interface{}

	To []Feed
}

// MarshalJSON is the custom marshal function for Activities
// It will be used by json.Marshal()
func (a Activity) MarshalJSON() ([]byte, error) {

	payload := make(map[string]interface{})

	for key, value := range a.MetaData {
		payload[key] = value
	}

	payload["actor"] = a.Actor
	payload["verb"] = a.Verb
	payload["object"] = a.Object
	payload["origin"] = a.Origin.Value()

	if a.ID != "" {
		payload["id"] = a.ID
	}
	if a.Target != "" {
		payload["target"] = a.Target
	}

	if a.Data != nil {
		payload["data"] = a.Data
	}

	if a.ForeignID != "" {
		payload["foreign_id"] = a.ForeignID
	}

	if a.TimeStamp == nil {
		payload["time"] = time.Now().Format(layout)
	} else {
		payload["time"] = a.TimeStamp.Format(layout)
	}

	var tos []string
	for _, feed := range a.To {
		to := feed.FeedID().Value()
		if feed.Token() != "" {
			to += " " + feed.Token()
		}
		tos = append(tos, to)
	}

	if len(tos) > 0 {
		payload["to"] = tos
	}

	return json.Marshal(payload)

}

// UnmarshalJSON is the custom unmarshal function for Activities
// It will be used by json.Unmarshal()
func (a *Activity) UnmarshalJSON(b []byte) (err error) {
	rawPayload := make(map[string]*payload)
	metadata := make(map[string]interface{})

	err = json.Unmarshal(b, &rawPayload)
	if err != nil {
		return err
	}

	for key, value := range rawPayload {
		lowerKey := strings.ToLower(key)

		if value == nil {
			continue
		}

		switch lowerKey {
		case "id":
			a.ID = value.String()
		case "actor":
			a.Actor = value.String()
		case "verb":
			a.Verb = value.String()
		case "foreign_id":
			a.ForeignID = value.String()
		case "object":
			a.Object = value.String()
		case "origin":
			a.Origin = FeedID(value.String())
		case "target":
			a.Target = value.String()
		case "time":
			a.TimeStamp = value.TimeStamp()
		case "data":
			a.Data = &value.RawMessage
		case "to":
			a.To = value.To()
		default:
			var v interface{}
			json.Unmarshal(value.RawMessage, &v)
			metadata[key] = v
		}
	}

	a.MetaData = metadata
	return nil

}

type payload struct {
	json.RawMessage
}

func (p *payload) String() string {
	var strValue string
	json.Unmarshal(p.RawMessage, &strValue)
	return strValue
}

func (p *payload) TimeStamp() *time.Time {
	var strValue string
	err := json.Unmarshal(p.RawMessage, &strValue)
	if err != nil {
		return nil
	}

	timeStamp, err := time.Parse(layout, strValue)
	if err != nil {
		return nil
	}

	return &timeStamp
}

func (p *payload) To() []Feed {
	var feed []Feed

	for _, to := range p.to1D() {
		m1 := matchFeed(to)
		if m1 != nil {
			feed = append(feed, m1)
		}

		m2 := matchFeedWithToken(to)
		if m2 != nil {
			feed = append(feed, m2)
		}
	}

	return feed
}

func (p *payload) to1D() []string {
	var to1D []string
	var to2D [][]string

	err := json.Unmarshal(p.RawMessage, &to1D)
	if err != nil {
		err = json.Unmarshal(p.RawMessage, &to2D)
		if err == nil {
			for _, to := range to2D {
				if len(to) == 2 {
					feedStr := to[0] + " " + to[1]
					to1D = append(to1D, feedStr)
				} else if len(to) == 1 {
					to1D = append(to1D, to[0])
				}
			}
		}
	}

	return to1D
}

func matchFeedWithToken(to string) *GeneralFeed {
	match, err := regexp.MatchString(`^\w+:\w+ .*?$`, to)
	if err != nil {
		return nil
	}

	if match {
		firstSplit := strings.Split(to, ":")
		secondSplit := strings.Split(firstSplit[1], " ")

		return &GeneralFeed{
			FeedSlug: firstSplit[0],
			UserID:   secondSplit[0],
			token:    secondSplit[1],
		}
	}

	return nil
}

func matchFeed(to string) *GeneralFeed {
	match, err := regexp.MatchString(`^\w+:\w+$`, to)
	if err != nil {
		return nil
	}

	if match {
		firstSplit := strings.Split(to, ":")

		return &GeneralFeed{
			FeedSlug: firstSplit[0],
			UserID:   firstSplit[1],
		}
	}
	return nil
}
