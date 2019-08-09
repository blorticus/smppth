package smppth

import (
	"fmt"
	"reflect"
	"smpp"
	"strconv"
)

type PduFactory struct {
	nextSequenceNumber        uint32
	defaultSubmitSmParameters map[string]interface{}
}

func NewPduFactory() *PduFactory {
	return &PduFactory{
		nextSequenceNumber: 2,
		defaultSubmitSmParameters: map[string]interface{}{
			"source_addr_npi":  uint8(0),
			"source_addr":      "",
			"dest_addr_npi":    uint8(0),
			"destination_addr": "",
			"short_message":    "This is a test short message",
		},
	}
}

func (factory *PduFactory) CreateEnquireLink() *smpp.PDU {
	factory.nextSequenceNumber++
	return smpp.NewPDU(smpp.CommandEnquireLink, 0, factory.nextSequenceNumber, []*smpp.Parameter{}, []*smpp.Parameter{})
}

func (factory *PduFactory) CreateEnquireLinkRespFromRequest(requestPDU smpp.PDU) *smpp.PDU {
	return smpp.NewPDU(smpp.CommandEnquireLinkResp, 0, requestPDU.SequenceNumber, []*smpp.Parameter{}, []*smpp.Parameter{})
}

func (factory *PduFactory) CreateSubmitSmUsingTextParameters(parameters map[string]string) (*smpp.PDU, error) {
	usingParameters := make(map[string]interface{})

	for key, defaultValue := range factory.defaultSubmitSmParameters {
		desiredValue, keyIsInOverrideMap := parameters[key]
		if keyIsInOverrideMap {
			requiredTypeForValue := reflect.TypeOf(defaultValue).Kind().String()
			coercedValue, ableToCoerceDesiredValue := factory.attemptCoersionFromStringToNamedType(desiredValue, requiredTypeForValue)

			if !ableToCoerceDesiredValue {
				return nil, fmt.Errorf("Unable to coerce parameter (%s) from (%s) to type (%s)", key, desiredValue, requiredTypeForValue)
			}

			usingParameters[key] = coercedValue
		} else {
			usingParameters[key] = defaultValue
		}
	}

	shortMessage := usingParameters["short_message"].(string)

	return smpp.NewPDU(smpp.CommandSubmitSm, 0, 1, []*smpp.Parameter{
		smpp.NewFLParameter(uint8(0)),                                               // service_type
		smpp.NewFLParameter(uint8(0)),                                               // source_addr_ton
		smpp.NewFLParameter(usingParameters["source_addr_npi"].(uint8)),             // source_addr_npi
		smpp.NewCOctetStringParameter(usingParameters["source_addr"].(string)),      // source_addr
		smpp.NewFLParameter(uint8(0)),                                               // dest_addr_ton
		smpp.NewFLParameter(usingParameters["dest_addr_npi"].(uint8)),               // dest_addr_npi
		smpp.NewCOctetStringParameter(usingParameters["destination_addr"].(string)), // destination_addr
		smpp.NewFLParameter(uint8(0)),                                               // esm_class
		smpp.NewFLParameter(uint8(0)),                                               // protocol_id
		smpp.NewFLParameter(uint8(0)),                                               // priority_flag
		smpp.NewFLParameter(uint8(0)),                                               // scheduled_delivery_time
		smpp.NewFLParameter(uint8(0)),                                               // validity_period
		smpp.NewFLParameter(uint8(0)),                                               // registered_delivery
		smpp.NewFLParameter(uint8(0)),                                               // replace_if_present_flag
		smpp.NewFLParameter(uint8(0)),                                               // data_coding
		smpp.NewFLParameter(uint8(0)),                                               // sm_defalt_msg_id
		smpp.NewFLParameter(uint8(len(shortMessage))),
		smpp.NewOctetStringFromString(shortMessage),
	}, []*smpp.Parameter{}), nil
}

func (factory *PduFactory) CreateSubmitSmRespFromRequest(requestPDU *smpp.PDU, messageID string) *smpp.PDU {
	return smpp.NewPDU(smpp.CommandSubmitSm, 0, 1, []*smpp.Parameter{
		smpp.NewCOctetStringParameter(messageID),
	}, []*smpp.Parameter{})
}

func (factory *PduFactory) attemptCoersionFromStringToNamedType(value string, coercedType string) (interface{}, bool) {
	switch coercedType {
	case "uint8":
		coercedValue, err := strconv.ParseUint(value, 10, 8)
		if err != nil {
			return uint8(0), false
		}

		return uint8(coercedValue), true

	case "string":
		return value, true
	}

	return value, false
}
