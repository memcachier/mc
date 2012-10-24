package mc

// Deal with the protocol specification of Memcached.

import (
	"errors"
)

// Errors
var (
	ErrNotFound       = errors.New("mc: not found")
	ErrKeyExists      = errors.New("mc: key exists")
	ErrValueTooLarge  = errors.New("mc: value to large")
	ErrInvalidArgs    = errors.New("mc: invalid arguments")
	ErrValueNotStored = errors.New("mc: value not stored")
	ErrNonNumeric     = errors.New("mc: incr/decr called on non-numeric value")
	ErrAuthRequired   = errors.New("mc: authentication required")
	ErrAuthContinue   = errors.New("mc: authentication continue (unsupported)")
	ErrUnknownCommand = errors.New("mc: unknown command")
	ErrOutOfMemory    = errors.New("mc: out of memory")
  // for unknown errors from client...
  ErrUnknownError   = errors.New("mc: unknown error from server")
)

var errMap = map[uint16]error{
	0:    nil,
	1:    ErrNotFound,
	2:    ErrKeyExists,
	3:    ErrValueTooLarge,
	4:    ErrInvalidArgs,
	5:    ErrValueNotStored,
	6:    ErrNonNumeric,
	0x20: ErrAuthRequired,
	// we only support PLAIN auth, no mechanism that would make use of auth
	// continue, so make it an error for now for completeness.
	0x21: ErrAuthContinue,
	0x81: ErrUnknownCommand,
	0x82: ErrOutOfMemory,
}

type opCode uint8

// Ops
const (
	OpGet opCode = opCode(iota)
	OpSet
	OpAdd
	OpReplace
	OpDelete
	OpIncrement
	OpDecrement
	OpQuit
	OpFlush
	OpGetQ
	OpNoop
	OpVersion
	OpGetK
	OpGetKQ
	OpAppend
	OpPrepend
	OpStat
	OpSetQ
	OpAddQ
	OpReplaceQ
	OpDeleteQ
	OpIncrementQ
	OpDecrementQ
	OpQuitQ
	OpFlushQ
	OpAppendQ
	OpPrependQ
	_ // Verbosity - not actually implemented in memcached
	OpTouch
	OpGAT
	OpGATQ
	OpGATK = opCode(0x23)
	OpGATKQ = opCode(0x24)
)

// Auth Ops
const (
	OpAuthList opCode = opCode(iota + 0x20)
	OpAuthStart
	OpAuthStep
)

// Magic Codes
type magicCode uint8
const (
	MagicSend magicCode = 0x80
	MagicRecv magicCode = 0x81
)

// Memcache header
type header struct {
	Magic        magicCode
	Op           opCode
	KeyLen       uint16
	ExtraLen     uint8
	DataType     uint8  // not used, memcached expects it to be 0x00.
	ResvOrStatus uint16 // for request this field is reserved / unused, for
	                    // response it indicates the status
	BodyLen      uint32
	Opaque       uint32 // copied back to you in response message (message id)
	CAS          uint64 // version really
}

// Main Memcache message structure
type msg struct {
	header                 // [0..23]
	iextras []interface{}  // [24..(m-1)] Command specific extras (In)

	// Idea of this is we can pass in pointers to values that should appear in the
	// response extras in this field and the generic send/recieve code can handle.
	oextras []interface{}  // [24..(m-1)] Command specifc extras (Out)


	key     string         // [m..(n-1)] Key (as needed, length in header)
	val     string         // [n..x] Value (as needed, length in header)
}

