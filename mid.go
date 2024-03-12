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

const (
	kMidLen    = 12
	kMidHexLen = kMidLen * 2
)

// MID 从 mgo.bson 复制
type MID string

func MIDHex(s string) MID {
	d, err := hex.DecodeString(s)
	if err != nil || len(d) != kMidLen {
		panic(fmt.Sprintf("MID: invalid input to MIDHex: %q", s))
	}
	return MID(d)
}

func IsMIDHex(s string) bool {
	if len(s) != kMidHexLen {
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
		panic(fmt.Errorf("MID: cannot read random number: %v", err))
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
			panic(fmt.Errorf("MID: cannot get hostname: %v; %v", err1, err2))
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

func NewMID() MID {
	var b [kMidLen]byte
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))
	// Machine, first 3 bytes of md5(hostname)
	b[4] = machineId[0]
	b[5] = machineId[1]
	b[6] = machineId[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	b[7] = processId[0]
	b[8] = processId[1]
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&idCounter, 1)
	b[9] = byte(i >> 16)
	b[10] = byte(i >> 8)
	b[11] = byte(i)
	return MID(b[:])
}

func (m MID) String() string {
	return fmt.Sprintf(`MIDHex("%x")`, string(m))
}

func (m MID) Hex() string {
	return hex.EncodeToString([]byte(m))
}

func (m MID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%x"`, string(m))), nil
}

var nullBytes = []byte("null")

func (m *MID) UnmarshalJSON(data []byte) error {
	if len(data) == 2 && data[0] == '"' && data[1] == '"' || bytes.Equal(data, nullBytes) {
		*m = ""
		return nil
	}
	if len(data) != kMidHexLen+2 || data[0] != '"' || data[kMidHexLen+1] != '"' {
		return errors.New(fmt.Sprintf("MID: invalid MID in JSON: %s", string(data)))
	}
	var buf [kMidLen]byte
	_, err := hex.Decode(buf[:], data[1:kMidHexLen+1])
	if err != nil {
		return errors.New(fmt.Sprintf("MID: invalid MID in JSON: %s (%s)", string(data), err))
	}
	*m = MID(string(buf[:]))
	return nil
}

func (m MID) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%x", string(m))), nil
}

func (m *MID) UnmarshalText(data []byte) error {
	if len(data) == 1 && data[0] == ' ' || len(data) == 0 {
		*m = ""
		return nil
	}
	if len(data) != kMidHexLen {
		return fmt.Errorf("MID: invalid MID: %s", data)
	}
	var buf [kMidLen]byte
	_, err := hex.Decode(buf[:], data[:])
	if err != nil {
		return fmt.Errorf("MID: invalid MID: %s (%s)", data, err)
	}
	*m = MID(string(buf[:]))
	return nil
}

func (m MID) Value() (driver.Value, error) {
	b, err := m.MarshalText()
	return string(b), err
}

func (m *MID) Scan(value interface{}) (err error) {
	switch val := value.(type) {
	case string:
		return m.UnmarshalText([]byte(val))
	case []byte:
		return m.UnmarshalText(val)
	case nil:
		return nil
	default:
		return fmt.Errorf("MID: scanning unsupported type: %T", value)
	}
}

func (m MID) Valid() bool {
	return len(m) == kMidLen
}

func (m MID) byteSlice(start, end int) []byte {
	if len(m) != kMidLen {
		panic(fmt.Sprintf("MID: invalid MID: %q", string(m)))
	}
	return []byte(string(m)[start:end])
}

func (m MID) Time() int64 {
	return int64(binary.BigEndian.Uint32(m.byteSlice(0, 4)))
}

func (m MID) Machine() []byte {
	return m.byteSlice(4, 7)
}

func (m MID) Pid() uint16 {
	return binary.BigEndian.Uint16(m.byteSlice(7, 9))
}

func (m MID) Counter() int32 {
	b := m.byteSlice(9, 12)
	return int32(uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]))
}
