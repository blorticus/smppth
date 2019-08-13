package smppth

import (
	"fmt"
	"reflect"
	"smpp"
	"testing"
)

func TestDefaultFactoryCreateEnquireLink(t *testing.T) {
	f := NewDefaultPduFactory()

	pdu := f.CreateEnquireLink()

	if pdu == nil {
		t.Errorf("Expected PDU, got nil")
	}

	if pdu.CommandID != smpp.CommandEnquireLink {
		t.Errorf("Expected pdu.CommandID is [%d] (enquire-link), got = [%d] (%s)", smpp.CommandEnquireLink, pdu.CommandID, pdu.CommandName())
	}
}

func TestDefaultFactoryCreateEnquireLinkFromRequest(t *testing.T) {
	f := NewDefaultPduFactory()

	enquireLinkPdu := smpp.NewPDU(smpp.CommandEnquireLink, 0, 10, []*smpp.Parameter{}, []*smpp.Parameter{})

	pdu := f.CreateEnquireLinkRespFromRequest(enquireLinkPdu)

	if pdu == nil {
		t.Errorf("Expected PDU, got nil")
	}

	if pdu.CommandID != smpp.CommandEnquireLinkResp {
		t.Errorf("Expected pdu.CommandID is [%d] (enquire-link-resp), got = [%d] (%s)", smpp.CommandEnquireLinkResp, pdu.CommandID, pdu.CommandName())
	}

	if pdu.SequenceNumber != enquireLinkPdu.SequenceNumber {
		t.Errorf("Expected Sequence Number = (%d), got = (%d)", enquireLinkPdu.SequenceNumber, pdu.SequenceNumber)
	}
}

func TestDefaultFactoryCreateSubmitSmNoParameters(t *testing.T) {
	f := NewDefaultPduFactory()

	factoryProducedPdu, err := f.CreateSubmitSm(map[string]string{})
	if err != nil {
		t.Errorf("Expected no error, got err = (%s)", err)
	}

	expectedSubmitSm := smpp.NewPDU(smpp.CommandSubmitSm, 0, 0, []*smpp.Parameter{
		smpp.NewFLParameter(uint8(0)),     // service_type
		smpp.NewFLParameter(uint8(0)),     // source_addr_ton
		smpp.NewFLParameter(uint8(0)),     // source_addr_npi
		smpp.NewCOctetStringParameter(""), // source_addr
		smpp.NewFLParameter(uint8(0)),     // dest_addr_ton
		smpp.NewFLParameter(uint8(0)),     // dest_addr_npi
		smpp.NewCOctetStringParameter(""), // destination_addr
		smpp.NewFLParameter(uint8(0)),     // esm_class
		smpp.NewFLParameter(uint8(0)),     // protocol_id
		smpp.NewFLParameter(uint8(0)),     // priority_flag
		smpp.NewFLParameter(uint8(0)),     // scheduled_delivery_time
		smpp.NewFLParameter(uint8(0)),     // validity_period
		smpp.NewFLParameter(uint8(0)),     // registered_delivery
		smpp.NewFLParameter(uint8(0)),     // replace_if_present_flag
		smpp.NewFLParameter(uint8(0)),     // data_coding
		smpp.NewFLParameter(uint8(0)),     // sm_defalt_msg_id
		smpp.NewFLParameter(uint8(28)),
		smpp.NewOctetStringFromString("This is a test short message"),
	}, []*smpp.Parameter{})

	err = compareSubmitSmPDUs(expectedSubmitSm, factoryProducedPdu)

	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestDefaultFactoryCreateSubmitSmWithParameters(t *testing.T) {
	f := NewDefaultPduFactory()

	factoryProducedPdu, err := f.CreateSubmitSm(map[string]string{
		"foo":              "bar",
		"source_addr_npi":  "1",
		"source_addr":      "source addr",
		"dest_addr_npi":    "2",
		"destination_addr": "dest addr",
	})
	if err != nil {
		t.Errorf("Expected no error, got err = (%s)", err)
	}

	expectedSubmitSm := smpp.NewPDU(smpp.CommandSubmitSm, 0, 0, []*smpp.Parameter{
		smpp.NewFLParameter(uint8(0)),                // service_type
		smpp.NewFLParameter(uint8(0)),                // source_addr_ton
		smpp.NewFLParameter(uint8(1)),                // source_addr_npi
		smpp.NewCOctetStringParameter("source addr"), // source_addr
		smpp.NewFLParameter(uint8(0)),                // dest_addr_ton
		smpp.NewFLParameter(uint8(2)),                // dest_addr_npi
		smpp.NewCOctetStringParameter("dest addr"),   // destination_addr
		smpp.NewFLParameter(uint8(0)),                // esm_class
		smpp.NewFLParameter(uint8(0)),                // protocol_id
		smpp.NewFLParameter(uint8(0)),                // priority_flag
		smpp.NewFLParameter(uint8(0)),                // scheduled_delivery_time
		smpp.NewFLParameter(uint8(0)),                // validity_period
		smpp.NewFLParameter(uint8(0)),                // registered_delivery
		smpp.NewFLParameter(uint8(0)),                // replace_if_present_flag
		smpp.NewFLParameter(uint8(0)),                // data_coding
		smpp.NewFLParameter(uint8(0)),                // sm_defalt_msg_id
		smpp.NewFLParameter(uint8(28)),
		smpp.NewOctetStringFromString("This is a test short message"),
	}, []*smpp.Parameter{})

	err = compareSubmitSmPDUs(expectedSubmitSm, factoryProducedPdu)

	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestDefaultFactoryCreateSubmitSmWithImproperlyTypedParameter(t *testing.T) {
	f := NewDefaultPduFactory()

	_, err := f.CreateSubmitSm(map[string]string{
		"foo":              "bar",
		"source_addr_npi":  "1",
		"source_addr":      "source addr",
		"dest_addr_npi":    "two",
		"destination_addr": "dest addr",
	})

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func compareSubmitSmPDUs(expected *smpp.PDU, got *smpp.PDU) error {
	if got == nil {
		return fmt.Errorf("Expected PDU, got nil")
	}

	if got.CommandID != smpp.CommandSubmitSm {
		return fmt.Errorf("Expected pdu.CommandID is [%d] (submit-sm), got = [%d] (%s)", smpp.CommandSubmitSm, got.CommandID, got.CommandName())
	}

	if len(got.MandatoryParameters) != len(expected.MandatoryParameters) {
		return fmt.Errorf("Expected %d mandatory parameters, got = (%d)", len(expected.MandatoryParameters), len(got.MandatoryParameters))
	}

	for i, expectedParameter := range expected.MandatoryParameters {
		expectedValue := expectedParameter.Value
		gotValue := got.MandatoryParameters[i].Value
		if reflect.ValueOf(expectedValue).Kind() != reflect.ValueOf(gotValue).Kind() {
			return fmt.Errorf("For mandatory parameter (%d), expected type = (%s), got = (%s)", i, reflect.ValueOf(expectedValue).Kind().String(), reflect.ValueOf(gotValue).Kind().String())
		}

		switch reflect.ValueOf(expectedValue).Kind() {
		case reflect.Uint8:
			if expectedValue.(uint8) != gotValue.(uint8) {
				return fmt.Errorf("For mandatory parameter (%d), expected (%d), got (%d)", i, expectedValue.(uint8), gotValue.(uint8))
			}
		case reflect.String:
			if expectedValue.(string) != gotValue.(string) {
				return fmt.Errorf("For mandatory parameter (%d), expected (%s), got (%s)", i, expectedValue.(string), gotValue.(string))
			}
		default:
			if string(expectedValue.([]byte)) != string(gotValue.([]byte)) {
				return fmt.Errorf("For mandatory parameter (%d), expected (%s), got (%s)", i, string(expectedValue.([]byte)), string(gotValue.([]byte)))
			}
		}

	}

	return nil
}
