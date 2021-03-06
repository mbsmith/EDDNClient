package EDDNClient

import (
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	"strings"
)

var (
	errUnhandledSchema = errors.New("schema not supported")
)

// Root is the root of every JSON message received from EDDN.  This should
// not be used directly as this is lazily parsed to find the schema first.
type Root struct {
	SchemaRef string          `json:"$schemaRef"` // The schema of the message
	Header    Header          `json:"header"`     // The message header
	Message   json.RawMessage `json:"message"`    // The message unparsed until later
}

// Header type that is common to all messages.  This bit is only used by the parser
// however.  The types sent by the ChannelInterface will have their own
// Root/Header types that the receiver should use.
type Header struct {
	GatewayTimestamp string `json:"gatewayTimestamp,omitempty"` // Timestamp
	SoftwareName     string `json:"softwareName"`               // Software that sent the data
	SoftwareVersion  string `json:"softwareVersion"`            // Software version
	UploaderID       string `json:"uploaderID"`                 // ID of the uploader
}

func handleJournalMessage(msg interface{}) (out interface{}, err error) {

	if journalMsg, ok := msg.(map[string]interface{}); ok {

		if event, ok := journalMsg["event"]; ok {

			switch event {
			case "FSDJump":
				var jumpMsg JournalFSDJump
				err := mapstructure.Decode(journalMsg, &jumpMsg)

				if err != nil {
					return nil, err
				}

				return jumpMsg, nil

			case "Docked":
				var dockedMsg JournalDocked
				err := mapstructure.Decode(journalMsg, &dockedMsg)

				if err != nil {
					return nil, err
				}

				return dockedMsg, nil

			case "Scan":
				// Check if it's a star, or a body.
				if _, ok := journalMsg["StarType"]; ok {
					var scanMsg JournalScanStar
					err := mapstructure.Decode(journalMsg, &scanMsg)

					if err != nil {
						return nil, err
					}

					return scanMsg, nil
				}

				// We have a body
				var scanMsg JournalScanPlanet
				err := mapstructure.Decode(journalMsg, &scanMsg)

				if err != nil {
					return nil, err
				}

				return scanMsg, nil

			default:
				return nil, errors.New("invalid event, or event not found")
			}

		}

	}

	return nil, errors.New("msg is not a Journal type")
}

func parseJSON(data string) (parsed interface{}, err error) {
	r, _ := zlib.NewReader(strings.NewReader(data))
	defer r.Close()

	output, err := ioutil.ReadAll(r)

	if err != nil {
		fmt.Printf("Error: %v", err)
		return nil, err
	}

	// Parse the schema to find out what kind of message we're going to be
	// handling.
	var jsonData Root

	err = json.Unmarshal(output, &jsonData)

	if err != nil {
		fmt.Println("Error: ", err)
		return nil, err
	}

	switch jsonData.SchemaRef {
	case "http://schemas.elite-markets.net/eddn/commodity/1":
		fallthrough
	case "http://schemas.elite-markets.net/eddn/commodity/2":
		err := errors.New("commodity versions 1 and 2 not currently supported")
		return nil, err

	case "http://schemas.elite-markets.net/eddn/commodity/3":
		var commodityData Commodity
		json.Unmarshal(output, &commodityData)
		return commodityData, nil

	case "http://schemas.elite-markets.net/eddn/journal/1":
		var journalData Journal
		json.Unmarshal(output, &journalData)

		parsedMsg, err := handleJournalMessage(journalData.Message)

		if err != nil {
			return nil, err
		}

		journalData.Message = parsedMsg

		return journalData, nil

	case "http://schemas.elite-markets.net/eddn/outfitting/1":
		err := errors.New("outfitting version 1 is not currently supported")
		return nil, err

	case "http://schemas.elite-markets.net/eddn/outfitting/2":
		var outfittingData Outfitting
		json.Unmarshal(output, &outfittingData)
		return outfittingData, nil

	case "http://schemas.elite-markets.net/eddn/blackmarket/1":
		var blackmarketData Blackmarket
		json.Unmarshal(output, &blackmarketData)
		return blackmarketData, nil

	case "http://schemas.elite-markets.net/eddn/shipyard/1":
		err := errors.New("shipyard version 1 is not currently supported")
		return nil, err

	case "http://schemas.elite-markets.net/eddn/shipyard/2":
		var shipyardData Shipyard
		json.Unmarshal(output, &shipyardData)
		return shipyardData, nil

		// Handle special cases with test.  Disregard these.
	case "http://schemas.elite-markets.net/eddn/shipyard/2/test":
		fallthrough
	case "http://schemas.elite-markets.net/eddn/blackmarket/1/test":
		fallthrough
	case "http://schemas.elite-markets.net/eddn/outfitting/2/test":
		fallthrough
	case "http://schemas.elite-markets.net/eddn/journal/1/test":
		fallthrough
	case "http://schemas.elite-markets.net/eddn/commodity/3/test":
		fallthrough

	default:
		return nil, errUnhandledSchema
	}

}
