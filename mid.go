package xid

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"sync/atomic"
)

// 从 mgo.bson 复制
type MID string

func MIDHex(s string) MID {
	d, err := hex.DecodeString(s)
	if err != nil || len(d) != 7 {
		panic(fmt.Sprintf("invalid input to MIDHex: %q", s))
	}
	return MID(d)
}

func IsMIDHex(s string) bool {
	if len(s) != 14 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

var midCounter = readRandomUint32()

func NewMID() MID {
	var b = make([]byte, 7)
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))

	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&idCounter, 1)
	b[4] = byte(i >> 16)
	b[5] = byte(i >> 8)
	b[6] = byte(i)

	return MID(b[:])
}

func NewMIDWithTime(t time.Time) MID {
	var b [7]byte
	binary.BigEndian.PutUint32(b[:4], uint32(t.UnixNano()))
	return MID(string(b[:]))
}

func (id MID) String() string {
	return fmt.Sprintf(`MIDHex("%x")`, string(id))
}

func (id MID) Hex() string {
	return hex.EncodeToString([]byte(id))
}

func (id MID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%x"`, string(id))), nil
}

func (id *MID) UnmarshalJSON(data []byte) error {
	if len(data) == 2 && data[0] == '"' && data[1] == '"' || bytes.Equal(data, nullBytes) {
		*id = ""
		return nil
	}
	if len(data) != 16 || data[0] != '"' || data[15] != '"' {
		return errors.New(fmt.Sprintf("invalid MID in JSON: %s", string(data)))
	}
	var buf [7]byte
	_, err := hex.Decode(buf[:], data[1:15])
	if err != nil {
		return errors.New(fmt.Sprintf("invalid MID in JSON: %s (%s)", string(data), err))
	}
	*id = MID(string(buf[:]))
	return nil
}

func (id MID) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%x", string(id))), nil
}

func (id *MID) UnmarshalText(data []byte) error {
	if len(data) == 1 && data[0] == ' ' || len(data) == 0 {
		*id = ""
		return nil
	}
	if len(data) != 14 {
		return fmt.Errorf("invalid MID: %s", data)
	}
	var buf [7]byte
	_, err := hex.Decode(buf[:], data[:])
	if err != nil {
		return fmt.Errorf("invalid MID: %s (%s)", data, err)
	}
	*id = MID(string(buf[:]))
	return nil
}

func (id MID) Value() (driver.Value, error) {
	b, err := id.MarshalText()
	return string(b), err
}

func (id *MID) Scan(value interface{}) (err error) {
	switch val := value.(type) {
	case string:
		return id.UnmarshalText([]byte(val))
	case []byte:
		return id.UnmarshalText(val)
	case nil:
		return nil
	default:
		return fmt.Errorf("mid: scanning unsupported type: %T", value)
	}
}

func (id MID) Valid() bool {
	return len(id) == 7
}

func (id MID) byteSlice(start, end int) []byte {
	if len(id) != 7 {
		panic(fmt.Sprintf("invalid MID: %q", string(id)))
	}
	return []byte(string(id)[start:end])
}

func (id MID) Time() time.Time {
	secs := int64(binary.BigEndian.Uint32(id.byteSlice(0, 4)))
	return time.Unix(secs, 0)
}

func (id MID) Counter() int32 {
	b := id.byteSlice(4, 7)
	return int32(uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]))
}
