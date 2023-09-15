package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/r3labs/sse"

	browser "github.com/EDDYCJY/fake-useragent"
)

var relays = []string{
	"https://boost-relay.flashbots.net",
	"https://bloxroute.regulated.blxrbdn.com",
	"https://bloxroute.max-profit.blxrbdn.com",
	"https://relay.ultrasound.money",
	"https://agnostic-relay.net",
	"https://builder-relay-mainnet.blocknative.com",
	"https://aestus.live",
}

var beaconClientHost string

func init() {
	flag.StringVar(&beaconClientHost, "beacon-client", "http://localhost:5052", "beacon client host")
}

func main() {

	flag.Parse()
	url := fmt.Sprintf("%v/eth/v1/events?topics=payload_attributes", beaconClientHost)

	fmt.Println("Starting to listen for beacon events at ", url)

	client := sse.NewClient(url)

	previousData := payloadAttributeEvent{
		Data: payloadAttributeData{
			ProposalSlot:    0,
			ParentBlockHash: "",
			ProposerIndex:   "",
		},
	}

	err := client.SubscribeRaw(func(msg *sse.Event) {
		data := payloadAttributeEvent{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Print("could not process beacon event, msg data", "msg", msg.Data, "err", err)
			return
		}

		if previousData.Data.ProposalSlot == data.Data.ProposalSlot {
			return
		}

		if previousData.Data.ProposerIndex != "" {
			pubkey, err := validatorIndexToPubkey(beaconClientHost, previousData.Data.ProposerIndex)
			if err != nil {
				log.Print("could not get pubkey", "err", err)
				if previousData.Data.ProposalSlot != data.Data.ProposalSlot {
					previousData = data
				}
				return
			}

			registeredRelays := getRegistration(relays, pubkey)

			time.Sleep(500 * time.Millisecond)
			builderPubkey, relaySubmissions, err := checkSubmissions(data.Data.ParentBlockHash, relays)
			if err != nil {
				log.Print("could not check submissions", "err", err)
				return
			}

			isBXRegistered := false

			bxRelays := []string{}

			for _, relay := range registeredRelays {
				if strings.Contains(relay, "bloxroute") {
					isBXRegistered = true
					bxRelays = append(bxRelays, relay)
				}
			}

			if !isBXRegistered {
				if previousData.Data.ProposalSlot != data.Data.ProposalSlot {
					previousData = data
				}
				return
			}

			logMessage := fmt.Sprintf("Bids not received for slot %v\n", data.Data.ProposalSlot-1)
			logMessage += fmt.Sprintf("BlockHash: %v\n", data.Data.ParentBlockHash)
			logMessage += fmt.Sprintf("Builder pubkey: %v\n", builderPubkey)

			relays := ""
			for _, bxRelay := range bxRelays {
				if _, ok := relaySubmissions[bxRelay]; !ok {
					relays += fmt.Sprintf("%v\n%s/relay/v1/data/bidtraces/builder_blocks_received?block_hash=%v\n", bxRelay, bxRelay, data.Data.ParentBlockHash)
				}
			}
			if relays != "" {
				logMessage += fmt.Sprintf("Relays not received: \n%v\n", relays)
				fmt.Println(logMessage)
			}

		}

		if previousData.Data.ProposalSlot != data.Data.ProposalSlot {
			previousData = data
		}

	})

	if err != nil {
		log.Fatal(err)
	}
}

func checkSubmissions(blockhash string, relays []string) (string, map[string]bool, error) {
	client := http.Client{}
	relaySubmissions := make(map[string]bool)
	builderPubkey := ""
	for _, relay := range relays {
		req, err := http.NewRequest("GET", fmt.Sprintf("%v/relay/v1/data/bidtraces/builder_blocks_received?block_hash=%v", relay, blockhash), nil)
		if err != nil {
			return builderPubkey, nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("could not get bidtrace data from relay", "relay", relay, "err", err)
			continue
		}

		if resp.StatusCode != 200 {
			fmt.Println("could not get bidtrace data from relay", "relay", relay, "status", resp.StatusCode)
			continue
		}

		var data []blockSubmission
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			fmt.Println("could not decode bidtrace data from relay", "relay", relay, "err", err)
			continue
		}

		if len(data) > 0 {
			relaySubmissions[relay] = true
			builderPubkey = data[0].BuilderPubkey
		}
	}
	return builderPubkey, relaySubmissions, nil
}

func validatorIndexToPubkey(beaconClient, index string) (string, error) {

	url := fmt.Sprintf("%v/eth/v1/beacon/states/head/validators?id=%s", beaconClient, index)

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("could not get validator data", "err", err)
		return "", err
	}

	if resp.StatusCode != 200 {
		fmt.Println("could not get validator data", "status", resp.StatusCode)
		return "", err
	}

	var data validatorResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("could not decode validator data", "err", err)
		return "", err
	}

	if len(data.Data) > 0 {
		return data.Data[0].Validator.Pubkey, nil
	}

	return "", nil
}

func getRegistration(relays []string, pubkey string) []string {
	registeredRelays := []string{}
	for _, relayURL := range relays {
		buf := make([]byte, 4)
		ip := rand.Uint32()

		binary.LittleEndian.PutUint32(buf, ip)
		// randomIP := net.IP(buf).String()
		client := http.Client{}

		url := fmt.Sprintf("%s/relay/v1/data/validator_registration?pubkey=%s", relayURL, pubkey)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			fmt.Printf("could not get registration from %s\n", url)
			continue
		}

		req.Header.Set("User-Agent", browser.Random())

		res, err := client.Do(req)
		if err != nil {
			fmt.Printf("could not get registration from %s\n", url)
			continue
		}

		if res.StatusCode == http.StatusOK {
			registeredRelays = append(registeredRelays, relayURL)
		}
	}
	return registeredRelays
}
