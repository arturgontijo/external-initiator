package blockchain

import (
	"encoding/json"

	common2 "github.com/smartcontractkit/external-initiator/blockchain/common"
	"github.com/smartcontractkit/external-initiator/blockchain/evm"
	"github.com/smartcontractkit/external-initiator/store"
	"github.com/smartcontractkit/external-initiator/subscriber"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/smartcontractkit/chainlink/core/logger"
)

const HMY = "harmony"

// The hmyManager implements the subscriber.JsonManager interface and allows
// for interacting with HMY nodes over RPC or WS.
type hmyManager struct {
	fq           *evm.FilterQuery
	p            subscriber.Type
	endpointName string
	jobid        string
}

// createHmyManager creates a new instance of hmyManager with the provided
// connection type and store.EthSubscription config.
func createHmyManager(p subscriber.Type, config store.Subscription) hmyManager {
	return hmyManager{
		fq:           evm.CreateEvmFilterQuery(config.Job, config.Ethereum.Addresses),
		p:            p,
		endpointName: config.EndpointName,
		jobid:        config.Job,
	}
}

// GetTriggerJson generates a JSON payload to the HMY node
// using the config in hmyManager.
//
// If hmyManager is using WebSocket:
// Creates a new "hmy_subscribe" subscription.
//
// If hmyManager is using RPC:
// Sends a "hmy_getLogs" request.
func (h hmyManager) GetTriggerJson() []byte {
	/*if h.p == subscriber.RPC && h.fq.FromBlock == "" {
		h.fq.FromBlock = "latest"
	}

	filter, err := h.fq.toMapInterface()
	if err != nil {
		return nil
	}

	filterBytes, err := json.Marshal(filter)
	if err != nil {
		return nil
	}

	msg := common2.JsonrpcMessage{
		Version: "2.0",
		ID:      json.RawMessage(`1`),
	}

	switch h.p {
	case subscriber.WS:
		msg.Method = "hmy_subscribe"
		msg.Params = json.RawMessage(`["logs",` + string(filterBytes) + `]`)
	case subscriber.RPC:
		msg.Method = "hmy_getLogs"
		msg.Params = json.RawMessage(`[` + string(filterBytes) + `]`)
	default:
		logger.Errorw(common2.ErrSubscriberType.Error(), "type", h.p)
		return nil
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil
	}

	return bytes*/
	return nil
}

// GetTestJson generates a JSON payload to test
// the connection to the HMY node.
//
// If hmyManager is using WebSocket:
// Returns nil.
//
// If hmyManager is using RPC:
// Sends a request to get the latest block number.
func (h hmyManager) GetTestJson() []byte {
	if h.p == subscriber.RPC {
		msg := common2.JsonrpcMessage{
			Version: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "hmy_blockNumber",
		}

		bytes, err := json.Marshal(msg)
		if err != nil {
			return nil
		}

		return bytes
	}

	return nil
}

// ParseTestResponse parses the response from the
// HMY node after sending GetTestJson(), and returns
// the error from parsing, if any.
//
// If hmyManager is using WebSocket:
// Returns nil.
//
// If hmyManager is using RPC:
// Attempts to parse the block number in the response.
// If successful, stores the block number in hmyManager.
func (h hmyManager) ParseTestResponse(data []byte) error {
	if h.p == subscriber.RPC {
		var msg common2.JsonrpcMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		var res string
		if err := json.Unmarshal(msg.Result, &res); err != nil {
			return err
		}
		h.fq.FromBlock = res
	}

	return nil
}

// ParseResponse parses the response from the
// HMY node, and returns a slice of subscriber.Events
// and if the parsing was successful.
//
// If hmyManager is using RPC:
// If there are new events, update hmyManager with
// the latest block number it sees.
func (h hmyManager) ParseResponse(data []byte) ([]subscriber.Event, bool) {
	promLastSourcePing.With(prometheus.Labels{"endpoint": h.endpointName, "jobid": h.jobid}).SetToCurrentTime()
	logger.Debugw("Parsing response", "ExpectsMock", common2.ExpectsMock)

	var msg common2.JsonrpcMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.Error("failed parsing msg: ", msg)
		return nil, false
	}

	var events []subscriber.Event

	switch h.p {
	case subscriber.WS:
		/*var res ethereum.ethSubscribeResponse
		if err := json.Unmarshal(msg.Params, &res); err != nil {
			logger.Error("unmarshal:", err)
			return nil, false
		}

		var evt models.Log
		if err := json.Unmarshal(res.Result, &evt); err != nil {
			logger.Error("unmarshal:", err)
			return nil, false
		}

		if evt.Removed {
			return nil, false
		}

		request, err := evm.logEventToOracleRequest(evt)
		if err != nil {
			logger.Error("failed to get oracle request:", err)
			return nil, false
		}

		_, err = json.Marshal(request)
		if err != nil {
			logger.Error("marshal:", err)
			return nil, false
		}*/

		// events = append(events, event)

	case subscriber.RPC:
		/*var rawEvents []models.Log
		if err := json.Unmarshal(msg.Result, &rawEvents); err != nil {
			logger.Error("unmarshal:", err)
			return nil, false
		}

		for _, evt := range rawEvents {
			if evt.Removed {
				continue
			}

			request, err := evm.logEventToOracleRequest(evt)
			if err != nil {
				logger.Error("failed to get oracle request:", err)
				return nil, false
			}

			_, err = json.Marshal(request)
			if err != nil {
				logger.Error("failed marshaling request:", err)
				continue
			}
			// events = append(events, event)

			// Check if we can update the "fromBlock" in the query,
			// so we only get new events from blocks we haven't queried yet
			// Increment the block number by 1, since we want events from *after* this block number
			curBlkn := &big.Int{}
			curBlkn = curBlkn.Add(big.NewInt(int64(evt.BlockNumber)), big.NewInt(1))

			fromBlkn, err := hexutil.DecodeBig(h.fq.FromBlock)
			if err != nil && !(h.fq.FromBlock == "latest" || h.fq.FromBlock == "") {
				logger.Error("Failed to get block number from event:", err)
				continue
			}

			// If our query "fromBlock" is "latest", or our current "fromBlock" is in the past compared to
			// the last event we received, we want to update the query
			if h.fq.FromBlock == "latest" || h.fq.FromBlock == "" || curBlkn.Cmp(fromBlkn) > 0 {
				h.fq.FromBlock = hexutil.EncodeBig(curBlkn)
			}
		}*/

	default:
		logger.Errorw(common2.ErrSubscriberType.Error(), "type", h.p)
		return nil, false
	}

	return events, true
}
