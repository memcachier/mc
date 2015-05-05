package mc

// Deal with the protocol specification of Memcached.

// Error represents a MemCache error (including the status code). All function
// in mc return error values of this type, despite the functions using the plain
// error type. You can safely cast all error types returned by mc to *Error. If
// needed, we take an underlying error value (such as a network error) and wrap
// it in Error, storing the underlying value in WrappedError.
//
// error is used as the return typed instead of Error directly due to the
// limitation in Go where error(nil) != *Error(nil).
type Error struct {
	Status       uint16
	Message      string
	WrappedError error
}

func (err Error) Error() string {
	return err.Message
}

// newError takes a status from the server and creates a matching Error.
func newError(status uint16) *Error {
	switch status {
	case StatusOK:
		return nil
	case StatusNotFound:
		return ErrNotFound
	case StatusKeyExists:
		return ErrKeyExists
	case StatusValueTooLarge:
		return ErrValueTooLarge
	case StatusInvalidArgs:
		return ErrInvalidArgs
	case StatusValueNotStored:
		return ErrValueNotStored
	case StatusNonNumeric:
		return ErrNonNumeric
	case StatusAuthRequired:
		return ErrAuthRequired

	// we only support PLAIN auth, no mechanism that would make use of auth
	// continue, so make it an error for now for completeness.
	case StatusAuthContinue:
		return ErrAuthContinue
	case StatusUnknownCommand:
		return ErrUnknownCommand
	case StatusOutOfMemory:
		return ErrOutOfMemory
	}
	return ErrUnknownError
}

// Errors mc may return. Some errors aren't represented here as the message is
// dynamically generated. Status Code however captures all possible values for
// Error.Status.
var (
	ErrNotFound       = &Error{StatusNotFound, "mc: not found", nil}
	ErrKeyExists      = &Error{StatusKeyExists, "mc: key exists", nil}
	ErrValueTooLarge  = &Error{StatusValueNotStored, "mc: value to large", nil}
	ErrInvalidArgs    = &Error{StatusInvalidArgs, "mc: invalid arguments", nil}
	ErrValueNotStored = &Error{StatusValueNotStored, "mc: value not stored", nil}
	ErrNonNumeric     = &Error{StatusNonNumeric, "mc: incr/decr called on non-numeric value", nil}
	ErrAuthRequired   = &Error{StatusAuthRequired, "mc: authentication required", nil}
	ErrAuthContinue   = &Error{StatusAuthContinue, "mc: authentication continue (unsupported)", nil}
	ErrUnknownCommand = &Error{StatusUnknownCommand, "mc: unknown command", nil}
	ErrOutOfMemory    = &Error{StatusOutOfMemory, "mc: out of memory", nil}
	ErrUnknownError   = &Error{StatusUnknownError, "mc: unknown error from server", nil}
)

// Status Codes that may be returned (usually as part of an Error).
const (
	StatusOK             = 0
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
