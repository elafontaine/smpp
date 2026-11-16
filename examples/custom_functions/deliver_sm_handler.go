package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	. "github.com/elafontaine/smpp"
)

/*
https://www.plantuml.com/plantuml/uml/RS-zheCm30NWdQSu8mmCzxOJ0-9DK9HO9MfYezWKyVRzaP0ewchx77qUr5on9QTAi_hH2pDvYy9eUv1c6dsAn8OEVr3YW40fFgYCcglZlks_h_yn5_6aYaAUNebZ4f6D2hkKDjH1m69Jv1lMQ1EYDQTcd6mTBd2iAnMOm2R2xFoTxDSFvr67woxREseMGv0tmF7saJJLG1oMd9u0
     ┌───────────────┐            ┌───────────┐                                               ┌───────────────────┐
     │smsc_connection│            │esme_client│                                               │main_client_program│
     └───────┬───────┘            └─────┬─────┘                                               └─────────┬─────────┘
             │   1 send deliver_sm     ┌┴┐                                                              │          
             │───────────────────────> │ │                                                              │          
             │                         │ │                                                              │          
             │                         │ │2 process received deliver_sm internally (do nothing with it)┌┴┐         
             │                         │ │ ───────────────────────────────────────────────────────────>│ │         
             │                         │ │                                                             └┬┘         
             │                         │ │      3 return answer to deliver_sm (do nothing with it)      │          
             │                         │ │ <─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─          
             │                         └┬┘                                                              │          
             │ 4 answer to the packet   │                                                               │          
             │<─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                                                               │          
             │    ────┐                                                                                 │          
             │        │ 5 process answer                                                                │          
             │    <───┘  (not doing anything)                                                           │          
             │                                                                                          │          
             │                          │                                                               │          
             │                          │                                                               │          
     ┌───────┴───────┐            ┌─────┴─────┐                                               ┌─────────┴─────────┐
     │smsc_connection│            │esme_client│                                               │main_client_program│
     └───────────────┘            └───────────┘                                               └───────────────────┘
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

func step1(server_smpp_conn_obj *ESME) {
	deliverSM := NewDeliverSM().
	WithDestinationAddress(expectedDestinationAddress).
	WithDataCoding(3).
	WithMessage("Hello, how are you today ?").
	WithSourceAddress("5557654321")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := server_smpp_conn_obj.Send(ctx, deliverSM) // steps 1 to 4: returns with the answer
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't send on esme : %v", err))
		os.Exit(1)
	}
	if resp.Body.MandatoryParameter["message_id"] != expectedMessageId { // step 5
		fmt.Print(fmt.Errorf("The smsc_connection didn't receive expected message ID!"))
		os.Exit(1)
	}
}

func step2(receivedMessageChannel chan *pduNeedingAnswer) func(e *ESME, p PDU) error {
	return func(e *ESME, p PDU) error {
		current_state := e.GetEsmeState()
		DebugSmppLogger.Printf("Pdu received on client : %v", p)
		if !(current_state == BOUND_RX || current_state == BOUND_TRX) {
			resp_pdu := NewDeliverSMResp().WithSequenceNumber(p.Header.SequenceNumber).WithSMPPError(ESME_RINVBNDSTS)
			_ = e.Reply(resp_pdu)
			return fmt.Errorf("Not supposed to receive deliver_sm if binding isn't as a receiver!")
		} else { // Right type of binding
			if p.Body.MandatoryParameter["destination_addr"] == expectedDestinationAddress {
				// expected destination is right
				// send PDU into channel to offload esme and process the PDU in our own manner (maybe we got multiple esme ?)
				answerChannel := make(chan *PDU)
				receivedMessageChannel <- &pduNeedingAnswer{p, answerChannel}
				return e.Reply((<-answerChannel).WithSequenceNumber(p.Header.SequenceNumber)) // Step 3
			}
		}
		return nil
	}
}

func main() {
	// Server side preparations
	connType := "tcp"

	receivedMessageChannel := make(chan *pduNeedingAnswer)
	// internal esme_client doing "something" with received PDU and provide an answer to the esme_client (step 3)
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
	smsc.Start() // Start the control loop for the smsc_connection (useful for step 5)
	defer smsc.Close()

	// function that step 2 will be calling on the esme_client
	esme.CommandFunctions["deliver_sm"] = step2(receivedMessageChannel)
	esme.StartControlLoop()
	defer esme.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = esme.BindReceiver(ctx, "MySystemId", "Password")
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't bind esme : %v", err))
		os.Exit(1)
	}



	server_smpp_conn_obj := smsc.ESMEs.Load().([]*ESME)[0] // Have the server connection object send the deliver_sm
	step1(server_smpp_conn_obj)

	fmt.Println("Reached the end of the program!")
}
