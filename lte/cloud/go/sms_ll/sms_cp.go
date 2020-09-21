/*
Copyright 2020 The Magma Authors.

This source code is licensed under the BSD-style license found in the
LICENSE file in the root directory of this source tree.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package sms_ll

import (
	"errors"
	"fmt"
)

type smsCPError struct {
	Field string
	Err   error
}

func SMSCPError(s string) error {
	return smsCPError{s, errors.New(s)}
}

func (e smsCPError) Error() string {
	return fmt.Sprintf("smscp: %s", e.Field)
}

// Handles creation of SMS-CM messages (3GPP TS 24.011 7.2)

// CP Message represents
type CPMessage struct {
	// Contains Transaction ID and Protocol Disc
	FirstOctet byte

	// CP-DATA, CP-ACK, or CP-ERROR
	MessageType byte

	// Only present for CP-ERROR
	Cause byte

	// Only present for CP-DATA
	Length byte

	// Only present for CP-DATA
	RPDU []byte
}

func (cpm CPMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0)
	b = append(b, cpm.FirstOctet)
	b = append(b, cpm.MessageType)

	switch cpm.MessageType {
	case CP_DATA:
		b = append(b, cpm.Length)
		b = append(b, cpm.RPDU...)
	case CP_ERROR:
		b = append(b, cpm.Cause)
	case CP_ACK:
		// No additional data, do nothing
	default:
		return nil, errors.New("Invalid CP MessageType")
	}

	return b, nil
}

func (cpm CPMessage) TransactionId() byte {
	fo := cpm.FirstOctet
	return fo >> 4
}

func (cpm CPMessage) ProtocolDisc() byte {
	fo := cpm.FirstOctet
	fo &= 0xf
	return fo
}

func (cpm *CPMessage) SetTransactionId(input byte) error {
	if int(input) < 0 || int(input) > 15 {
		return SMSCPError("Transaction ID must be 0-15")
	}

	cpm.FirstOctet &^= 0xf0
	cpm.FirstOctet |= (input << 4)

	return nil
}

func (cpm *CPMessage) SetProtocolDisc(input byte) error {
	if input != CP_PROTOCOL_DISC {
		return SMSCPError("Invalid protocol discriminator (must be 0x9)")
	}

	cpm.FirstOctet &^= 0xf
	cpm.FirstOctet |= input

	return nil
}

func (cpm *CPMessage) UnmarshalBinary(input []byte) error {
	// must be at least two octets long
	if len(input) < 2 {
		return SMSCPError("SMS-CP Message too short")
	}

	cpm.FirstOctet = byte(input[0])
	cpm.MessageType = byte(input[1])

	switch cpm.MessageType {
	case CP_DATA:
		cpm.Length = byte(input[2])
		cpm.RPDU = input[3:]
	case CP_ERROR:
		if _, ok := CP_CAUSE_STR[input[2]]; ok {
			cpm.Cause = input[2]
		} else {
			return SMSCPError("Invalid cause")
		}
	case CP_ACK:
		// Do nothing -- no more data
	default:
		return SMSCPError("Invalid IE type")
	}

	return nil
}

const (
	// CP Message bit fields

	// Protocol discriminator (3GPP TS 24.007 11.2.3.1.1)
	// For SMS-related messages, this is always 0x9 (half-octet)
	CP_PROTOCOL_DISC = 0x9

	// Message types
	CP_DATA  = 0x1
	CP_ACK   = 0x3
	CP_ERROR = 0x5

	// IE Types
	CP_IEI_USER  = 0x1
	CP_IEI_CAUSE = 0x2
)

const (
	// CP Cause error types (24.011 8.1.4.2, Table 8.2)
	CP_CAUSE_NETWORK_FAILURE                = 0x11
	CP_CAUSE_CONGESTION                     = 0x16
	CP_CAUSE_INVALID_TI                     = 0x51
	CP_CAUSE_SEMANTICALLY_INCORRECT         = 0x5f
	CP_CAUSE_INVALID_MANDANTORY_INFORMATION = 0x60
	CP_CAUSE_MESSAGE_TYPE_NONEXISTANT       = 0x61
	CP_CAUSE_MESSAGE_NOT_COMPATIBLE         = 0x62
	CP_CAUSE_INFO_ELEMENT_NONEXISTANT       = 0x63
	CP_CAUSE_PROTOCOL_ERROR                 = 0x6f
)

var CP_CAUSE_STR = map[byte]string{
	CP_CAUSE_NETWORK_FAILURE:                "Network failure",
	CP_CAUSE_CONGESTION:                     "Congestion",
	CP_CAUSE_INVALID_TI:                     "Invalid Transaction Identifier value",
	CP_CAUSE_SEMANTICALLY_INCORRECT:         "Semantically incorrect message",
	CP_CAUSE_INVALID_MANDANTORY_INFORMATION: "Invalid mandantory information",
	CP_CAUSE_MESSAGE_TYPE_NONEXISTANT:       "Message type non-existent or not implemented",
	CP_CAUSE_MESSAGE_NOT_COMPATIBLE:         "Message not compatible with the short message protocol state",
	CP_CAUSE_INFO_ELEMENT_NONEXISTANT:       "Information element non-existent or not implemented",
	CP_CAUSE_PROTOCOL_ERROR:                 "Protocol error, unspecified",
}
