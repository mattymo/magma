package sms_ll

import (
	"encoding/hex"
	"testing"
	"time"
)

// Test cases:
// 1) Marshal a specific SMS-Deliver and ensure the binary matches what we expect
// 2) Marshal a long SMS and ensure binary matches what we expect
// 3) Given a delivery report, decode the TPDU

// Pick a consistent timestamp for tests

func TestEncodeSingleSMS(t *testing.T) {
	msg := "Here's a test."
	ts := time.Date(2020, 9, 14, 16, 30, 50, 12345, time.UTC)
	num := "18658675309"
	ref := byte(7)
	expected := "590127010702b9110020240b918156685703f90000029041610305000ec8b2bc7c9a83c2207a794e7701"
	b, e := GenerateSMSDelivers(msg, num, ts, ref)
	if e != nil {
		t.Errorf("Error: %s", e)
	}
	if len(b) != 1 {
		t.Errorf("Too many PDUs generated")
	}
	if hex.EncodeToString(b[0]) != expected {
		t.Errorf("Incorrect PDU generated")
	}

}

func TestEncodingMultipleSMS(t *testing.T) {
	msg := "Here's a test of a veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeerrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrryyyyyyyyyyyyyyyyyy long message that's super super long."
	ts := time.Date(2020, 9, 14, 16, 30, 50, 12345, time.UTC)
	num := "18658675309"
	ref := byte(1)
	expected := []string{"5901a6010102b911009f640b918156685703f9000002904161030500a0050003010201906579f934078541f4f29c0e7a9b416190bd5c2e97cbe572b95c2e97cbe572b95c2e97cbe572b95c2e97cbe572b95c2e97cbe572b95c2e97cbe572b95c96cbe572b95c2e97cbe572b95c2e97cbe572b95c2e97cbe572b95c2e97cbe572b95c2e97cbe572b95c2ecfe7f3f97c3e9fcfe7f3f97c3e9fcfe741ecb7fb0c6a97e7f3f0b90ca2a3c3", "590133010202b911002c640b918156685703f90000029041610305001c050003010202e8a739685e8797e5a0791d5e9683d86ff7d905"}

	b, e := GenerateSMSDelivers(msg, num, ts, ref)
	if e != nil {
		t.Errorf("Error: %s", e)
	}
	if len(b) != 2 {
		t.Errorf("Wrong number of PDUs generated")
	}
	for i := range b {
		if hex.EncodeToString(b[i]) != expected[i] {
			t.Errorf("Incorrect PDU generated")
		}
	}
}

func TestDecodeDeliveryFailure(t *testing.T) {
	input := "d9010404010160"
	cp_hex, _ := hex.DecodeString(input)
	cpm := new(CPMessage)
	e := cpm.UnmarshalBinary(cp_hex)
	if e != nil {
		t.Errorf("Failed to decode valid CP-DATA")
	}
	if cpm.MessageType != CP_DATA {
		t.Errorf("Failed to decode valid CP-DATA")
	}
	if int(cpm.Length) != 4 || len(cpm.RPDU) != 4 {
		t.Errorf("CP-DATA length incorrect")
	}
	if cpm.Cause != 0x0 {
		t.Errorf("CP-DATA has cause set to non-zero")
	}

	rpm := new(RPMessage)
	e = rpm.UnmarshalBinary(cpm.RPDU)
	if e != nil {
		t.Errorf("Failed to decode valid RP-ERROR")
	}
	msg_type, e := rpm.MsgType()
	if e != nil {
		t.Errorf("Failed to decode valid RP-ERROR")
	}
	if msg_type != RP_ERROR || rpm.Cause.Cause != RP_CAUSE_INVALID_MANDANTORY_INFO {
		t.Errorf("Failed to decode valid RP-ERROR")
	}
}

func TestDecodeDeliveryReport(t *testing.T) {
	input := "d90106020141020000"
	cp_hex, _ := hex.DecodeString(input)
	cpm := new(CPMessage)
	e := cpm.UnmarshalBinary(cp_hex)
	if e != nil {
		t.Errorf("Failed to decode valid CP-DATA")
	}
	if cpm.MessageType != CP_DATA {
		t.Errorf("Failed to decode valid CP-DATA")
	}
	if int(cpm.Length) != 6 && len(cpm.RPDU) != 6 {
		t.Errorf("CP-DATA length incorrect")
	}
	if cpm.Cause != 0x0 {
		t.Errorf("CP-DATA has cause set to non-zero")
	}

	rpm := new(RPMessage)
	e = rpm.UnmarshalBinary(cpm.RPDU)
	if e != nil {
		t.Errorf("Failed to decode valid RP-ACK")
	}
	msg_type, e := rpm.MsgType()
	if e != nil {
		t.Errorf("Failed to decode valid RP-ACK")
	}
	if msg_type != RP_ACK {
		t.Errorf("Failed to decode valid RP-ACK")
	}
	if rpm.UserData.IEI != RP_UDE_IEI {
		t.Errorf("Failed to decode valid RP-ACK User Data IEI")
	}
	if rpm.UserData.Length != byte(2) {
		t.Errorf("Failed to decode valid RP-ACK User Data Length")
	}
	if len(rpm.UserData.TPDU) != int(rpm.UserData.Length) {
		t.Errorf("RP-ACK user data length doesn't match payload")
	}
}
