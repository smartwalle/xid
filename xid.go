package xid

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	kSequenceBits uint8 = 22 // 序列号占用的位数
	kNodeBits     uint8 = 8  // 数据节点占用的位数

	kMaxSequence int64 = -1 ^ (-1 << kSequenceBits) // 序列号最大值，用于防止溢出
	kMaxNode     int64 = -1 ^ (-1 << kNodeBits)     // 数据节点最大值，用于防止溢出 0-255

	kTimeShift = kNodeBits + kSequenceBits // 时间戳向左的偏移量
	kNodeShift = kSequenceBits             // 数据节点向左的偏移量

	kNodeMask = kMaxNode << kSequenceBits
)

var (
	ErrNodeNotAllowed = errors.New(fmt.Sprintf("xid: node can't be greater than %d or less than 0", kMaxNode))
)

type Option func(*XID) error

// WithNode 设置数据节点标识
func WithNode(node int64) Option {
	return func(x *XID) error {
		if node < 0 || node > kMaxNode {
			return ErrNodeNotAllowed
		}
		x.node = node
		return nil
	}
}

// WithTimeOffset 设置时间偏移量
func WithTimeOffset(t time.Time) Option {
	return func(x *XID) error {
		if t.IsZero() {
			return nil
		}
		x.timeOffset = t.Unix()
		return nil
	}
}

type XID struct {
	mu         sync.Mutex
	second     int64 // 上一次生成 id 的时间戳（秒）
	node       int64 // 数据节点
	sequence   int64 // 当前秒已经生成的 id 序列号 (从0开始累加)
	timeOffset int64
}

func New(opts ...Option) (*XID, error) {
	var x = &XID{}
	x.second = 0
	x.node = 0
	x.sequence = 0
	x.timeOffset = 0

	var err error
	for _, opt := range opts {
		if err = opt(x); err != nil {
			return nil, err
		}
	}

	return x, nil
}

func (this *XID) Next() int64 {
	this.mu.Lock()
	defer this.mu.Unlock()

	var second = time.Now().Unix()
	if second < this.second {
		return -1
	}

	if this.second == second {
		this.sequence = (this.sequence + 1) & kMaxSequence
		if this.sequence == 0 {
			second = this.getNextSecond()
		}
	} else {
		this.sequence = 0
	}
	this.second = second

	var id = (second-this.timeOffset)<<kTimeShift | (this.node << kNodeShift) | (this.sequence)
	return id
}

func (this *XID) getNextSecond() int64 {
	var second = time.Now().Unix()
	for second < this.second {
		second = time.Now().Unix()
	}
	return second
}

// Time 获取 id 的时间，单位是 second
func Time(s int64) int64 {
	return s >> kTimeShift
}

// Node 获取 id 的数据节点标识
func Node(s int64) int64 {
	return s & kNodeMask >> kNodeShift
}

//  Sequence 获取 id 的序列号
func Sequence(s int64) int64 {
	return s & kMaxSequence
}

var defaultXID *XID
var once sync.Once

func Next() int64 {
	once.Do(func() {
		defaultXID, _ = New()
	})
	return defaultXID.Next()
}

func Init(opts ...Option) (err error) {
	once.Do(func() {
		defaultXID, err = New(opts...)
	})

	if err != nil {
		once = sync.Once{}
	}

	return err
}
