package mc

// Deal with the protocol specification of Memcached.

// MCError represents a MemCache error (including the status code)
type MCError struct {
	Status  uint16
	Message string
}

func (err MCError) Error() string {
	return err.Message
}

// NewMCError takes a status from the server and creates a matching MCError.
func NewMCError(status uint16) *MCError {
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

// Errors
var (
	ErrNotFound       = &MCError{1, "mc: not found"}
	ErrKeyExists      = &MCError{2, "mc: key exists"}
	ErrValueTooLarge  = &MCError{3, "mc: value to large"}
	ErrInvalidArgs    = &MCError{4, "mc: invalid arguments"}
	ErrValueNotStored = &MCError{5, "mc: value not stored"}
	ErrNonNumeric     = &MCError{6, "mc: incr/decr called on non-numeric value"}
	ErrAuthRequired   = &MCError{0x20, "mc: authentication required"}
	ErrAuthContinue   = &MCError{0x21, "mc: authentication continue (unsupported)"}
	ErrUnknownCommand = &MCError{0x81, "mc: unknown command"}
	ErrOutOfMemory    = &MCError{0x82, "mc: out of memory"}
	ErrUnknownError   = &MCError{0, "mc: unknown error from server"}
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
