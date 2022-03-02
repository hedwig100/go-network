package icmp

import "fmt"

/*
	ICMP message type
*/

const (
	TypeEchoReply      MessageType = 0
	TypeDestUnreach    MessageType = 3
	TypeSourceQuench   MessageType = 4
	TypeRedirect       MessageType = 5
	TypeEcho           MessageType = 8
	TypeTimeExceeded   MessageType = 11
	TypeParamProblem   MessageType = 12
	TypeTimestamp      MessageType = 13
	TypeTimestampReply MessageType = 14
	TypeInfoRequest    MessageType = 15
	TypeInfoReply      MessageType = 16
)

type MessageType uint8

func (t MessageType) String() string {
	switch t {
	case 0:
		return "TypeEchoReply"
	case 3:
		return "TypeDestUnreach"
	case 4:
		return "TypeSourceQuench"
	case 5:
		return "TypeRedirect"
	case 8:
		return "TypeEcho"
	case 11:
		return "TypeTimeExceeded"
	case 12:
		return "TypeParamProblem"
	case 13:
		return "TypeTimestamp"
	case 14:
		return "TypeTimestampReply"
	case 15:
		return "TypeInfoRequest"
	case 16:
		return "TypeInfoReply"
	default:
		return "UNKNOWN"
	}
}

/*
	ICMP code
*/

const (
	// for Unreach
	CodeNetUnreach        MessageCode = 0
	CodeHostUnreach       MessageCode = 1
	CodeProtoUnreach      MessageCode = 2
	CodePortUnreach       MessageCode = 3
	CodeFragmentNeeded    MessageCode = 4
	CodeSourceRouteFailed MessageCode = 5

	// for Redirect
	CodeRedirectNet     MessageCode = 0
	CodeRedirectHost    MessageCode = 1
	CodeRedirectTosNet  MessageCode = 2
	CodeRedirectTosHost MessageCode = 3

	// for TimeExceeded
	CodeExceededTTL      MessageCode = 0
	CodeExceededFragment MessageCode = 1
)

type MessageCode uint8

func code2string(t MessageType, c MessageCode) string {
	switch t {
	case TypeDestUnreach:
		switch c {
		case CodeNetUnreach:
			return fmt.Sprintf("CodeNetUnreach(%d)", c)
		case CodeHostUnreach:
			return fmt.Sprintf("CodeHostUnreach(%d)", c)
		case CodeProtoUnreach:
			return fmt.Sprintf("CodeProtoUnreach(%d)", c)
		case CodePortUnreach:
			return fmt.Sprintf("CodePortUnreach(%d)", c)
		case CodeFragmentNeeded:
			return fmt.Sprintf("CodeFragmentNeeded(%d)", c)
		case CodeSourceRouteFailed:
			return fmt.Sprintf("CodeSourceRouteFailed(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	case TypeRedirect:
		switch c {
		case CodeRedirectNet:
			return fmt.Sprintf("CodeRedirectNet(%d)", c)
		case CodeRedirectHost:
			return fmt.Sprintf("CodeRedirectHost(%d)", c)
		case CodeRedirectTosNet:
			return fmt.Sprintf("CodeRedirectTosNet(%d)", c)
		case CodeRedirectTosHost:
			return fmt.Sprintf("CodeRedirectTosHost(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	case TypeTimeExceeded:
		switch c {
		case CodeExceededTTL:
			return fmt.Sprintf("CodeExceededTTL(%d)", c)
		case CodeExceededFragment:
			return fmt.Sprintf("CodeExceededFragment(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	default:
		return fmt.Sprintf("UNKNOWN(%d)", c)
	}
}
