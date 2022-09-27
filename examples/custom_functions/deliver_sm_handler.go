package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	. "github.com/elafontaine/smpp"
)

/*
https://www.plantuml.com/plantuml/uml/RS-zheCm30NWdQSu8mmCzxOJ0-9DK9HO9MfYezWKyVRzaP0ewchx77qUr5on9QTAi_hH2pDvYy9eUv1c6dsAn8OEVr3YW40fFgYCcglZlks_h_yn5_6aYaAUNebZ4f6D2hkKDjH1m69Jv1lMQ1EYDQTcd6mTBd2iAnMOm2R2xFoTxDSFvr67woxREseMGv0tmF7saJJLG1oMd9u0

     ┌───────────────┐                            ┌───────────┐
     │smsc_connection│                            │esme_client│
     └───────┬───────┘                            └─────┬─────┘
             │            1 send deliver_sm             │
             │─────────────────────────────────────────>│
             │                                          │
             │                                          ────┐
             │                                              │ 2 process received deliver_sm internally (do nothing with it)
             │                                          <───┘
             │                                          │
             │         3 answer to the packet           │
             │<─────────────────────────────────────────│
             │                                          │
             ────┐                                      │
                 │ 4 process answer (not doing anything)│
             <───┘                                      │
     ┌───────┴───────┐                            ┌─────┴─────┐
     │smsc_connection│                            │esme_client│
     └───────────────┘                            └───────────┘
*/

type pduNeedingAnswer struct {
	PDU
	resp_pdu_chan chan *PDU
}

const (
	expectedDestinationAddress = "5551234567"
	expectedMessageId          = "1"
)

func init() {
	DebugSmppLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoSmppLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningSmppLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorSmppLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func step2(receivedMessageChannel chan *pduNeedingAnswer) func(e *ESME, p PDU) error {
	return func(e *ESME, p PDU) error {
		current_state := e.GetEsmeState()
		DebugSmppLogger.Printf("Pdu received on client : %v", p)
		if !(current_state == BOUND_RX || current_state == BOUND_TRX) {
			resp_pdu := NewDeliverSMResp().WithSequenceNumber(p.Header.SequenceNumber)
			resp_pdu.WithSMPPError(ESME_RINVBNDSTS)
			e.Send(&resp_pdu)
			return fmt.Errorf("Not supposed to receive deliver_sm if binding isn't as a receiver!")
		} else { // Right type of binding
			if p.Body.MandatoryParameter["destination_addr"] == expectedDestinationAddress {
				// expected destination is right
				// send PDU into channel to offload esme and process the PDU in our own manner (maybe we got multiple esme ?)
				answerChannel := make(chan *PDU)
				receivedMessageChannel <- &pduNeedingAnswer{p, answerChannel}
				e.Send(<-answerChannel) // Step 3
			}
		}
		return nil
	}
}

func step4(channel chan bool) func(*ESME, PDU) error {
	return func(e *ESME, p PDU) error {

		if p.Body.MandatoryParameter["message_id"] != expectedMessageId {
			fmt.Print(fmt.Errorf("The smsc_connection didn't receive expected message ID!"))
			os.Exit(1)
		}
		channel <- true
		return nil
	}
}

func main() {
	// Server side preparations
	connType := "tcp"

	receivedMessageChannel := make(chan *pduNeedingAnswer)
	// internal esme_client doing "something" with received PDU and provide an answer to the esme_client (step 2)
	go func(c chan *pduNeedingAnswer) {
		for {
			pdu_package, ok := <-c
			if !ok {
				break
			}
			pdu_resp := NewDeliverSMResp().WithMessageId(expectedMessageId)
			pdu_package.resp_pdu_chan <- &pdu_resp
		}
	}(receivedMessageChannel)

	serversocket, _ := net.Listen(connType, "0.0.0.0:0")
	smsc := NewSMSC(&serversocket, "MySystemId", "Password")

	// Client side example (using the prepared server side)
	esme, err := InstantiateEsme(serversocket.Addr(), connType)
	if err != nil {
		fmt.Print(fmt.Errorf("Issue starting ESME connection to server! : %v", err))
		os.Exit(1)
	}
	smsc.Start() // Start the control loop for the smsc_connection (useful for step 4)
	defer smsc.Close()

	// function that step 2 will be calling on the esme_client
	esme.CommandFunctions["deliver_sm"] = step2(receivedMessageChannel)
	esme.StartControlLoop()
	defer esme.Close()

	_, err = esme.BindReceiver("MySystemId", "Password") // Currently skipping the response pdu check
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't bind esme : %v", err))
		os.Exit(1)
	}

	deliverSM := NewDeliverSM().
		WithDestinationAddress(expectedDestinationAddress).
		WithDataCoding(3).
		WithMessage("Hello, how are you today ?").
		WithSourceAddress("5557654321")

	server_smpp_conn_obj := smsc.ESMEs.Load().([]*ESME)[0] // Have the server connection object send the deliver_sm
	step4processing_channel := make(chan bool)
	server_smpp_conn_obj.CommandFunctions["deliver_sm_resp"] = step4(step4processing_channel)

	InfoSmppLogger.Println("About to send")
	_, err = server_smpp_conn_obj.Send(&deliverSM) // Step 1, skipping the response pdu checks
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't send on esme : %v", err))
		os.Exit(1)
	}
	InfoSmppLogger.Println("Sent")
	select {
	case <-time.After(10 * time.Second):
		InfoSmppLogger.Fatalln("Timeout of the program.")
	case <-step4processing_channel:

	}

	fmt.Println("Reached the end of the program!")
}
