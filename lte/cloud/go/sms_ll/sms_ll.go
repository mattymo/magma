package sms_ll

import (
	"github.com/warthog618/sms"
	"github.com/warthog618/sms/encoding/tpdu"
	"time"
)

// Generate fully encoded SMS PDUs for delivery to a UE (MS). Will handle
// encoding and chunking of messages as appropriate. We first generate TPDUs,
// then RP-DATA headers, and finally CP-DATA headers, resulting in a set of
// byte arrays that can be directly delivered to a UE (MS).
// Inputs:
// 	message: A UTF-8 string representing the SMS.
//	from_num: A E.164 encoded source number.
//	timestamp: The sender timestamp for the SMS (generally, use current server time)
//	ref_base: The starting reference number for this set of SMS messages. Should be a counter per IMSI.
// Outputs:
//	- Array of byte array representing the set of fully-encoded CP-DATA(RP-DATA(TPDU)) messages generated
//	- Error	(if any)
func GenerateSMSDelivers(message string, from_num string, timestamp time.Time, ref_base byte) ([][]byte, error) {
	tpdus := CreateTPDUs(message, from_num, timestamp)

	output := make([][]byte, 0)

	for i := range tpdus {
		tp, e := tpdus[i].MarshalBinary()
		if e != nil {
			return nil, e
		}
		reference := byte(int(ref_base) + i)
		rpm, e := CreateRPDataMessage(true, reference, tp)
		if e != nil {
			return nil, e
		}

		b, e := rpm.MarshalBinary()
		if e != nil {
			return nil, e
		}

		cpm, e := CreateCPDataMessage(b)
		if e != nil {
			return nil, e
		}

		b, e = cpm.MarshalBinary()
		if e != nil {
			return nil, e
		}

		output = append(output, b)
	}

	return output, nil
}

func CreateTPDUs(message string, from_num string, timestamp time.Time) []tpdu.TPDU {
	tpdus, _ := sms.Encode([]byte(message), sms.AsDeliver, sms.From(from_num))
	for i := range tpdus {
		tpdus[i].FirstOctet |= tpdu.FoMMS // Android won't accept if this bit isn't set.
		tpdus[i].FirstOctet |= tpdu.FoSRI // Request a delivery report.
		tpdus[i].SCTS = tpdu.Timestamp{Time: timestamp}
	}

	return tpdus
}

// Helper for creating a RP-DATA message. If is_mt is true, make this mobile terminated.
func CreateRPDataMessage(is_mt bool, reference byte, data []byte) (*RPMessage, error) {
	rpm := new(RPMessage)
	rpm.MTI = RP_MTI_MT_DATA
	rpm.Reference = reference

	var oa, da *RPAddressElement
	if is_mt { // MT-SMS should have empty dest address
		oa, _ = CreateRPAddressElement(false)
		da, _ = CreateRPAddressElement(true)
	} else { // MO-SMS should have empty orig address
		oa, _ = CreateRPAddressElement(true)
		da, _ = CreateRPAddressElement(false)
	}
	rpm.OriginatorAddress = *oa
	rpm.DestinationAddress = *da

	rpue, e := CreateRPUserElement(data)
	if e != nil {
		return nil, e
	}
	rpm.UserData = *rpue
	return rpm, nil
}

// Helper for generating RP Addresses.
func CreateRPAddressElement(empty bool) (*RPAddressElement, error) {
	rpadde := new(RPAddressElement)
	if empty {
		// Create a zero-length element
		rpadde.Length = 0
	} else {
		// Create a normal "fake" element
		rpadde.SetFakeNumber()
	}
	return rpadde, nil
}

func CreateCPDataMessage(rpdu []byte) (*CPMessage, error) {
	cpm := new(CPMessage)
	e := cpm.SetTransactionId(0x5)
	if e != nil {
		return nil, e
	}
	e = cpm.SetProtocolDisc(CP_PROTOCOL_DISC)
	if e != nil {
		return nil, e
	}
	cpm.MessageType = CP_DATA
	cpm.Length = byte(len(rpdu))
	cpm.RPDU = rpdu
	return cpm, nil
}
