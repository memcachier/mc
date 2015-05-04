package mc

// Deal with the protocol specification of Memcached.

// Error represents a MemCache error (including the status code)
type Error struct {
	Status  uint16
	Message string
}

func (err Error) Error() string {
	return err.Message
}

// NewError takes a status from the server and creates a matching Error.
func NewError(status uint16) *Error {
	switch status {
	case 0:
		return nil
	case 1:
		return ErrNotFound
	case 2:
		return ErrKeyExists
	case 3:
		return ErrValueTooLarge
	case 4:
		return ErrInvalidArgs
	case 5:
		return ErrValueNotStored
	case 6:
		return ErrNonNumeric
	case 0x20:
		return ErrAuthRequired

	// we only support PLAIN auth, no mechanism that would make use of auth
	// continue, so make it an error for now for completeness.
	case 0x21:
		return ErrAuthContinue
	case 0x81:
		return ErrUnknownCommand
	case 0x82:
		return ErrOutOfMemory
	}
	return ErrUnknownError
}

// Errors mc may return. Some errors aren't represented here as the message is
// dynamically generated. Status Code however captures all possible values for
// Error.Status.
var (
	ErrNotFound       = &Error{StatusNotFound, "mc: not found"}
	ErrKeyExists      = &Error{StatusKeyExists, "mc: key exists"}
	ErrValueTooLarge  = &Error{StatusValueNotStored, "mc: value to large"}
	ErrInvalidArgs    = &Error{StatusInvalidArgs, "mc: invalid arguments"}
	ErrValueNotStored = &Error{StatusValueNotStored, "mc: value not stored"}
	ErrNonNumeric     = &Error{StatusNonNumeric, "mc: incr/decr called on non-numeric value"}
	ErrAuthRequired   = &Error{StatusAuthRequired, "mc: authentication required"}
	ErrAuthContinue   = &Error{StatusAuthContinue, "mc: authentication continue (unsupported)"}
	ErrUnknownCommand = &Error{StatusUnknownCommand, "mc: unknown command"}
	ErrOutOfMemory    = &Error{StatusOutOfMemory, "mc: out of memory"}
	ErrUnknownError   = &Error{StatusUnknownError, "mc: unknown error from server"}
)

// Status Codes that may be returned (usually as part of an Error).
const (
	StatusNotFound       = 1
	StatusKeyExists      = 2
	StatusValueTooLarge  = 3
	StatusInvalidArgs    = 4
	StatusValueNotStored = 5
	StatusNonNumeric     = 6
	StatusAuthRequired   = 0x20
	StatusAuthContinue   = 0x21
	StatusUnknownCommand = 0x81
	StatusOutOfMemory    = 0x82
	StatusAuthUnknown    = 0xfff0
	StatusNetworkError   = 0xfff1
	StatusUnknownError   = 0xffff
)

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
	OpGATK  = opCode(0x23)
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
	magicSend magicCode = 0x80
	magicRecv magicCode = 0x81
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
	BodyLen uint32
	Opaque  uint32 // copied back to you in response message (message id)
	CAS     uint64 // version really
}

// Main Memcache message structure
type msg struct {
	header                // [0..23]
	iextras []interface{} // [24..(m-1)] Command specific extras (In)

	// Idea of this is we can pass in pointers to values that should appear in the
	// response extras in this field and the generic send/recieve code can handle.
	oextras []interface{} // [24..(m-1)] Command specifc extras (Out)

	key string // [m..(n-1)] Key (as needed, length in header)
	val string // [n..x] Value (as needed, length in header)
}
