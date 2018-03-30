package xid

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

// 从 mgo.bson 复制
type XID string

func XIDHex(s string) XID {
	d, err := hex.DecodeString(s)
	if err != nil || len(d) != 16 {
		panic(fmt.Sprintf("invalid input to XIDHex: %q", s))
	}
	return XID(d)
}

func IsXIDHex(s string) bool {
	if len(s) != 32 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

var idCounter = readRandomUint32()
var machineId = readMachineId()
var processId = readProcessId()

func readRandomUint32() uint32 {
	var b [4]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot read random xid: %v", err))
	}
	return uint32((uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24))
}

func readMachineId() []byte {
	var sum [3]byte
	id := sum[:]
	hostname, err1 := os.Hostname()
	if err1 != nil {
		_, err2 := io.ReadFull(rand.Reader, id)
		if err2 != nil {
			panic(fmt.Errorf("cannot get hostname: %v; %v", err1, err2))
		}
		return id
	}
	hw := md5.New()
	hw.Write([]byte(hostname))
	copy(id, hw.Sum(nil))
	return id
}

func readProcessId() []byte {
	var pId = os.Getpid()
	var id = make([]byte, 2)
	id[0] = byte(pId >> 8)
	id[1] = byte(pId)
	return id
}

func NewXID() XID {
	var b [16]byte
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint64(b[:], uint64(time.Now().UnixNano()))
	// Machine, first 3 bytes of md5(hostname)
	b[8] = machineId[0]
	b[9] = machineId[1]
	b[10] = machineId[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	b[11] = processId[0] // byte(processId >> 8)
	b[12] = processId[1] // byte(processId)
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&idCounter, 1)
	b[13] = byte(i >> 16)
	b[14] = byte(i >> 8)
	b[15] = byte(i)
	return XID(b[:])
}

func NewXIDWithTime(t time.Time) XID {
	var b [16]byte
	binary.BigEndian.PutUint64(b[:8], uint64(t.UnixNano()))
	return XID(string(b[:]))
}

func (id XID) String() string {
	return fmt.Sprintf(`XIDHex("%x")`, string(id))
}

func (id XID) Hex() string {
	return hex.EncodeToString([]byte(id))
}

func (id XID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%x"`, string(id))), nil
}

var nullBytes = []byte("null")

func (id *XID) UnmarshalJSON(data []byte) error {
	if len(data) == 2 && data[0] == '"' && data[1] == '"' || bytes.Equal(data, nullBytes) {
		*id = ""
		return nil
	}
	if len(data) != 34 || data[0] != '"' || data[33] != '"' {
		return errors.New(fmt.Sprintf("invalid XID in JSON: %s", string(data)))
	}
	var buf [16]byte
	_, err := hex.Decode(buf[:], data[1:33])
	if err != nil {
		return errors.New(fmt.Sprintf("invalid XID in JSON: %s (%s)", string(data), err))
	}
	*id = XID(string(buf[:]))
	return nil
}

func (id XID) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%x", string(id))), nil
}

func (id *XID) UnmarshalText(data []byte) error {
	if len(data) == 1 && data[0] == ' ' || len(data) == 0 {
		*id = ""
		return nil
	}
	if len(data) != 32 {
		return fmt.Errorf("invalid XID: %s", data)
	}
	var buf [16]byte
	_, err := hex.Decode(buf[:], data[:])
	if err != nil {
		return fmt.Errorf("invalid XID: %s (%s)", data, err)
	}
	*id = XID(string(buf[:]))
	return nil
}

func (id XID) Value() (driver.Value, error) {
	b, err := id.MarshalText()
	return string(b), err
}

func (id *XID) Scan(value interface{}) (err error) {
	switch val := value.(type) {
	case string:
		return id.UnmarshalText([]byte(val))
	case []byte:
		return id.UnmarshalText(val)
	case nil:
		return nil
	default:
		return fmt.Errorf("xid: scanning unsupported type: %T", value)
	}
}

func (id XID) Valid() bool {
	return len(id) == 16
}

func (id XID) byteSlice(start, end int) []byte {
	if len(id) != 16 {
		panic(fmt.Sprintf("invalid XID: %q", string(id)))
	}
	return []byte(string(id)[start:end])
}

func (id XID) Time() time.Time {
	secs := int64(binary.BigEndian.Uint64(id.byteSlice(0, 8)))
	return time.Unix(secs/1e9, secs%1e9)
}

func (id XID) Machine() []byte {
	return id.byteSlice(8, 11)
}

func (id XID) Pid() uint16 {
	return binary.BigEndian.Uint16(id.byteSlice(11, 13))
}

func (id XID) Counter() int32 {
	b := id.byteSlice(13, 16)
	return int32(uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]))
}
