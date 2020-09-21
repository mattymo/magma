package sms_ll

import (
	"errors"
	"fmt"
)

type smsRPError struct {
	Field string
	Err   error
}

func SMSRPError(s string) error {
	return smsRPError{s, errors.New(s)}
}

func (e smsRPError) Error() string {
	return fmt.Sprintf("smsrp: %s", e.Field)
}

// Used for both RP-Destination and RP-Originator Address, since it's the
// same layout (TS 24.011 8.2.5.1 and 8.2.5.2). Note that while 8.2.5.1-2
// suggest that the first octet is an IEI, 7.3.1 notes that this is a Type
// 4 LV IE, which means there's no IEI present -- just a length and values.
type RPAddressElement struct {
	Length     byte // of the number, not the element! actual size is 1 greater
	NumberInfo byte // octet 3
	Number     []byte
}

func (rpadde RPAddressElement) MarshalBinary() ([]byte, error) {
	rpadde_len := 0
	b := make([]byte, 0, 11)
	b = append(b, byte(rpadde_len))
	if rpadde.Length != 0x0 {
		rpadde_len += 1 + len(rpadde.Number)
		b = append(b, rpadde.NumberInfo)
		b = append(b, rpadde.Number...)
		b[0] = byte(rpadde_len) // update with new size
		if rpadde_len != int(rpadde.Length) {
			return nil, SMSRPError("Address size mismatch")
		}
	}
	return b, nil
}

// Decode an address element. Returns the length of the address element if present.
func (rpadde *RPAddressElement) UnmarshalBinary(input []byte) (int, error) {
	// Empty addresses will be one byte long with a zero value length
	if len(input) == 1 {
		if input[0] != 0x0 {
			return -1, SMSRPError("Invalid RP Address of length 1")
		} else {
			rpadde.Length = input[0]
			return 1, nil
		}
	} else if len(input) < 2 {
		return -1, SMSRPError("Invalid RP Address")
	}

	rpadde.Length = input[0]

	rpadde.NumberInfo = input[1]
	rpadde.Number = input[2:int(rpadde.Length)]
	return int(rpadde.Length), nil
}

// The RP-Address-Element refers to the number of the SMSC. In our case, we
// generally don't use an SMSC, so we just set the number to 11.
func (rpadde *RPAddressElement) SetFakeNumber() error {
	rpadde.NumberInfo = 0xb9     // network specific number, private numbering plan
	rpadde.Number = []byte{0x11} // the decimal number 1 1
	rpadde.Length = 0x2

	return nil
}

// RP-User data element (TS 24.011 8.2.5.3)
type RPUserElement struct {
	IEI    byte // Not present for RP-DATA
	Length byte
	TPDU   []byte
}

func CreateRPUserElement(data []byte) (*RPUserElement, error) {
	rpue := new(RPUserElement)
	rpue.IEI = RP_UDE_IEI
	if len(data) > 232 { // TS24.011 8.2.5.3
		return nil, SMSRPError("UserData-Element too long (>232 bytes)")

	}
	rpue.Length = byte(len(data))
	rpue.TPDU = data

	return rpue, nil
}

func (rpue RPUserElement) MarshalBinary(msgType byte) ([]byte, error) {
	rpu_len := len(rpue.TPDU) + 1
	b := make([]byte, 0, rpu_len)
	if msgType == RP_ACK || msgType == RP_ERROR { // these start with IEI
		b = append(b, rpue.IEI)
	}
	b = append(b, rpue.Length)
	b = append(b, rpue.TPDU...)
	return b, nil
}

func (rpue *RPUserElement) UnmarshalBinary(msgType byte, input []byte) (int, error) {
	idx := 0
	if msgType == RP_ACK || msgType == RP_ERROR { // these start with IEI
		rpue.IEI = input[idx]
		idx++
	}
	rpue.Length = input[idx]
	idx++
	n := idx + int(rpue.Length)
	rpue.TPDU = input[idx:n]
	idx += n
	return n, nil
}

// RP-Cause element (TS 24.011 8.2.5.4)
type RPCauseElement struct {
	IEI        byte // Never serialized
	Length     byte
	Cause      byte
	Diagnostic byte // Optional
}

func (rpce RPCauseElement) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0, 2)
	b = append(b, rpce.Length)
	b = append(b, rpce.Cause)
	if int(rpce.Length) == 2 {
		b = append(b, rpce.Diagnostic)
	}
	return b, nil
}

func (rpce *RPCauseElement) UnmarshalBinary(input []byte) (int, error) {
	rpce.Length = input[0]
	rpce.Cause = input[1]
	if int(rpce.Length) == 2 {
		rpce.Diagnostic = input[2]
		return 3, nil
	}
	return 2, nil
}

type RPMessage struct {
	MTI       byte
	Reference byte

	// Mandantory. If UE->Network, must be length 0
	OriginatorAddress RPAddressElement

	// Mandantory. If Network->UE, must be length 0
	DestinationAddress RPAddressElement

	// Mandantory for RP-DATA, includes TPDU. If RP-ACK or RP-ERROR, must
	// include IEI.
	UserData RPUserElement

	// Mandantory for RP-ERROR
	Cause RPCauseElement
}

func (rpm RPMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0, 2) // it's at least length 2...
	b = append(b, rpm.MTI)
	b = append(b, rpm.Reference)

	rpmt, e := rpm.MsgType()
	if e != nil {
		return nil, e
	}

	switch rpmt {
	case RP_DATA:
		oa_data, e := rpm.OriginatorAddress.MarshalBinary()
		if e != nil {
			return nil, e
		}
		da_data, e := rpm.DestinationAddress.MarshalBinary()
		if e != nil {
			return nil, e
		}
		b = append(b, oa_data...)
		b = append(b, da_data...)

		ud_data, _ := rpm.UserData.MarshalBinary(RP_DATA)
		b = append(b, ud_data...)
	case RP_ERROR:
		cause_data, _ := rpm.Cause.MarshalBinary()
		b = append(b, cause_data...)
	case RP_ACK:
		// Do nothing
	default:
		return nil, SMSRPError("Invalid RP Message Type")
	}

	return b, nil

}

func (rpm *RPMessage) UnmarshalBinary(input []byte) error {
	if len(input) < 2 {
		return SMSRPError("SMS-RP Message too short")
	}

	idx := 0
	rpm.MTI = input[idx]
	idx++ // 1
	rpm.Reference = input[idx]
	idx++ // 2

	rpmt, e := rpm.MsgType()
	if e != nil {
		return e
	}

	switch rpmt {
	case RP_DATA:
		// The next two IEs should be adddresses in this case. So, get the lengths and pass to unmarshal
		rpm.OriginatorAddress = *new(RPAddressElement)
		rpm.DestinationAddress = *new(RPAddressElement)
		rpm.UserData = *new(RPUserElement)
		n, _ := rpm.OriginatorAddress.UnmarshalBinary(input[idx:])
		if rpm.Direction() == RP_MO && n != 1 {
			return SMSRPError("SMS-RP-DATA is MO, but OA length != 1")
		}
		idx += n
		n, _ = rpm.DestinationAddress.UnmarshalBinary(input[idx:])
		if rpm.Direction() == RP_MT && n != 1 {
			return SMSRPError("SMS-RP-DATA is MT, but DA length != 1")
		}
		idx += n

		n, _ = rpm.UserData.UnmarshalBinary(RP_DATA, input[idx:])
		idx += n
	case RP_ACK:
		// RP-ACK and RP-ERROR may optionally contain an RP-User-Data
		// element (TS24.001 7.3.3). If this is the case, it will be a
		// TLV IE, with the first octet starting with the RP-User-Data
		// IE ID (0x41).
		if len(input) > 2 && input[idx] == RP_UDE_IEI {
			rpm.UserData = *new(RPUserElement)
			n, _ := rpm.UserData.UnmarshalBinary(RP_ACK, input[idx:])
			idx += n
		}
	case RP_ERROR:
		// Do nothing
		rpm.Cause = *new(RPCauseElement)
		n, _ := rpm.Cause.UnmarshalBinary(input[idx:])
		idx += n
		// TODO: Add support for optional UserData TLV element.
	default:
		return SMSRPError(fmt.Sprintf("Invalid RP-SMS MTI: 0x%08b", rpm.MTI))
	}

	return nil
}

// If MTI is even, message is UE->Network
func (rpm RPMessage) Direction() byte {
	if rpm.MTI&0x1 == 0 {
		return RP_MO
	}
	return RP_MT
}

func (rpm RPMessage) MsgType() (byte, error) {
	switch rpm.MTI {
	case RP_MTI_MO_DATA, RP_MTI_MT_DATA:
		return RP_DATA, nil
	case RP_MTI_MO_ERR, RP_MTI_MT_ERR:
		return RP_ERROR, nil
	case RP_MTI_MO_ACK, RP_MTI_MT_ACK:
		return RP_ACK, nil
	default:
		return RP_INVALID, SMSRPError(fmt.Sprintf("Invalid RP-SMS MTI: 0x%08b", rpm.MTI))
	}
}

const (
	RP_DATA = iota
	RP_ERROR
	RP_ACK
	RP_INVALID // error type
	RP_MO      // UE/MS->Network
	RP_MT      // Network->UE/MS
)

const (
	// RP Message fields

	//RP-MTI (TS24.011 8.2.2) is technically defined as a 3 bit field in
	//the low order bits of the first octet of the RPDU. However, the five
	//high order bits are defined to always be 0, so here we treat these
	//fields as a full octet.
	RP_MTI_MO_DATA = 0x0
	RP_MTI_MO_ACK  = 0x2
	RP_MTI_MO_ERR  = 0x4
	RP_MTI_MO_SMMA = 0x6
	RP_MTI_MT_DATA = 0x1
	RP_MTI_MT_ACK  = 0x3
	RP_MTI_MT_ERR  = 0x5

	RP_UDE_IEI   = 0x41
	RP_CAUSE_IEI = 0x42
)

const (
	// RP Cause types (TS24.011 Table 8.4)
	RP_CAUSE_UNASSIGNED               = 0x1
	RP_CAUSE_OP_BARRED                = 0x8
	RP_CAUSE_CALL_BARRED              = 0xa
	RP_CAUSE_RESERVED                 = 0xb
	RP_CAUSE_SM_TRANSFER_REJECTED     = 0x15
	RP_CAUSE_MEM_EXCEEDED             = 0x16
	RP_CAUSE_DEST_OUT_OF_ORDER        = 0x1b
	RP_CAUSE_UNIDENTIFIED_SUB         = 0x1c
	RP_CAUSE_FACILITY_REJECTED        = 0x1d
	RP_CAUSE_UNKNOWN_SUB              = 0x1e
	RP_CAUSE_NET_OUT_OF_ORDER         = 0x26
	RP_CAUSE_TEMP_FAILURE             = 0x29
	RP_CAUSE_CONGESTION               = 0x2a
	RP_CAUSE_RESOURCE_UNAVAILABLE     = 0x2f
	RP_CAUSE_REQUESTED_FAC_NOT_SUB    = 0x32
	RP_CAUSE_REQUESTED_FAC_NOT_IMPL   = 0x45
	RP_CAUSE_INVALID_SM_TRANS_REF     = 0x51
	RP_CAUSE_SEM_INCORRECT_MESSAGE    = 0x5f
	RP_CAUSE_INVALID_MANDANTORY_INFO  = 0x60
	RP_CAUSE_MSG_TYPE_NOT_IMPL        = 0x61
	RP_CAUSE_MSG_TYPE_NOT_COMPATIBLE  = 0x62
	RP_CAUSE_INFO_ELEMENT_NONEXISTANT = 0x63
	RP_CAUSE_PROTOCOL_ERROR           = 0x6f
	RP_CAUSE_INTERWORKING             = 0x7f
)

var RP_CAUSE_STR = map[byte]string{
	RP_CAUSE_UNASSIGNED:               "Unassigned (unallocated) number",
	RP_CAUSE_OP_BARRED:                "Operator determined barring",
	RP_CAUSE_CALL_BARRED:              "Call barred",
	RP_CAUSE_RESERVED:                 "Reserved",
	RP_CAUSE_SM_TRANSFER_REJECTED:     "Short message transfer rejected",
	RP_CAUSE_MEM_EXCEEDED:             "Memory capacity exceeded",
	RP_CAUSE_DEST_OUT_OF_ORDER:        "Destination out of order",
	RP_CAUSE_UNIDENTIFIED_SUB:         "Unidentified subscriber",
	RP_CAUSE_FACILITY_REJECTED:        "Facility rejected",
	RP_CAUSE_UNKNOWN_SUB:              "Unknown subscriber",
	RP_CAUSE_NET_OUT_OF_ORDER:         "Network out of order",
	RP_CAUSE_TEMP_FAILURE:             "Temporary failure",
	RP_CAUSE_CONGESTION:               "Congestion",
	RP_CAUSE_RESOURCE_UNAVAILABLE:     "Resources unavailable, unspecified",
	RP_CAUSE_REQUESTED_FAC_NOT_SUB:    "Requested facility not subscribed",
	RP_CAUSE_REQUESTED_FAC_NOT_IMPL:   "Requested facility not implemented",
	RP_CAUSE_INVALID_SM_TRANS_REF:     "Invalid short message transfer reference value",
	RP_CAUSE_SEM_INCORRECT_MESSAGE:    "Semantically incorrect message",
	RP_CAUSE_INVALID_MANDANTORY_INFO:  "Invalid mandantory information",
	RP_CAUSE_MSG_TYPE_NOT_IMPL:        "Message type not non-existent or not implemented",
	RP_CAUSE_MSG_TYPE_NOT_COMPATIBLE:  "Message not compatible with short message protocol state",
	RP_CAUSE_INFO_ELEMENT_NONEXISTANT: "Information element non-existent or not implemented",
	RP_CAUSE_PROTOCOL_ERROR:           "Protocol error, unspecified",
	RP_CAUSE_INTERWORKING:             "Interworking, unspecified",
}
