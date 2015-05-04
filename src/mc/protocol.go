package mc

// Deal with the protocol specification of Memcached.

type MCError struct {
  Status uint16
  Message string
}

func NewMCError(status uint16) *MCError {
  switch status {
    case 1:
      return &MCError{Status:status, Message: ErrNotFound}
    case 2:
      return &MCError{Status:status, Message: ErrKeyExists}
    case 3:
      return &MCError{Status:status, Message: ErrValueTooLarge}
    case 4:
      return &MCError{Status:status, Message: ErrInvalidArgs}
    case 5:
      return &MCError{Status:status, Message: ErrValueNotStored}
    case 6:
      return &MCError{Status:status, Message: ErrNonNumeric}
    case 0x20:
      return &MCError{Status:status, Message: ErrAuthRequired}

    // we only support PLAIN auth, no mechanism that would make use of auth
    // continue, so make it an error for now for completeness.
    case 0x21:
      return &MCError{Status:status, Message: ErrAuthContinue}
    case 0x81:
      return &MCError{Status:status, Message: ErrUnknownCommand}
    case 0x82:
      return &MCError{Status:status, Message: ErrOutOfMemory}
    default:
      return nil
  }
}

func (err MCError) Error() string {
  return err.Message
}

// Errors
var (
  ErrNotFound       = "mc: not found"
  ErrKeyExists      = "mc: key exists"
  ErrValueTooLarge  = "mc: value to large"
  ErrInvalidArgs    = "mc: invalid arguments"
  ErrValueNotStored = "mc: value not stored"
  ErrNonNumeric     = "mc: incr/decr called on non-numeric value"
  ErrAuthRequired   = "mc: authentication required"
  ErrAuthContinue   = "mc: authentication continue (unsupported)"
  ErrUnknownCommand = "mc: unknown command"
  ErrOutOfMemory    = "mc: out of memory"
  // for unknown errors from client...
  ErrUnknownError   = "mc: unknown error from server"
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

