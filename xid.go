package xid

import (
	"encoding/binary"
	"encoding/hex"
	"crypto/md5"
	"sync/atomic"
	"fmt"
	"os"
	"io"
	"crypto/rand"
	"time"
	"bytes"
	"errors"
)

// 从 mgo.bson 复制

var objectIdCounter uint32 = 0
var machineId = readMachineId()
var nullBytes = []byte("null")

type XID string

func XIDHex(s string) XID {
	d, err := hex.DecodeString(s)
	if err != nil || len(d) != 12 {
		panic(fmt.Sprintf("Invalid input to ObjectIdHex: %q", s))
	}
	return XID(d)
}

func NewXID() XID {
	var b [12]byte
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))
	// Machine, first 3 bytes of md5(hostname)
	b[4] = machineId[0]
	b[5] = machineId[1]
	b[6] = machineId[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	pid := os.Getpid()
	b[7] = byte(pid >> 8)
	b[8] = byte(pid)
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&objectIdCounter, 1)
	b[9] = byte(i >> 16)
	b[10] = byte(i >> 8)
	b[11] = byte(i)
	return XID(b[:])
}

func NewObjectIdWithTime(t time.Time) XID {
	var b [12]byte
	binary.BigEndian.PutUint32(b[:4], uint32(t.Unix()))
	return XID(string(b[:]))
}

func IsXIDHex(s string) bool {
	if len(s) != 24 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func (id XID) String() string {
	return fmt.Sprintf(`XID("%x")`, string(id))
}

func (id XID) Hex() string {
	return hex.EncodeToString([]byte(id))
}

func (id XID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%x"`, string(id))), nil
}

func (id *XID) UnmarshalJSON(data []byte) error {
	if len(data) == 2 && data[0] == '"' && data[1] == '"' || bytes.Equal(data, nullBytes) {
		*id = ""
		return nil
	}
	if len(data) != 26 || data[0] != '"' || data[25] != '"' {
		return errors.New(fmt.Sprintf("Invalid XID in JSON: %s", string(data)))
	}
	var buf [12]byte
	_, err := hex.Decode(buf[:], data[1:25])
	if err != nil {
		return errors.New(fmt.Sprintf("Invalid XID in JSON: %s (%s)", string(data), err))
	}
	*id = XID(string(buf[:]))
	return nil
}

func (id XID) Valid() bool {
	return len(id) == 12
}

func (id XID) byteSlice(start, end int) []byte {
	if len(id) != 12 {
		panic(fmt.Sprintf("Invalid XID: %q", string(id)))
	}
	return []byte(string(id)[start:end])
}

func (id XID) Time() time.Time {
	secs := int64(binary.BigEndian.Uint32(id.byteSlice(0, 4)))
	return time.Unix(secs, 0)
}

func (id XID) Machine() []byte {
	return id.byteSlice(4, 7)
}

func (id XID) Pid() uint16 {
	return binary.BigEndian.Uint16(id.byteSlice(7, 9))
}

func (id XID) Counter() int32 {
	b := id.byteSlice(9, 12)
	return int32(uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]))
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