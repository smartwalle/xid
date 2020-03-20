package xid

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// 34 位用于存储时间戳(秒)信息，最大可以存储的值为 17179869183，即 2514-05-30 09:53:03.
// 8 位用于存储数据节点信息，最大可以存储 255，即可以有 256 个节点.
// 21 位用于存储每一秒产生的序列号信息，最大可以存储 2097151，即每秒最大可以生产 2097152 个序列号.

const (
	kSequenceBits uint8 = 21 // 序列号占用的位数
	kDataNodeBits uint8 = 8  // 数据节点占用的位数

	kMaxSequence int64 = -1 ^ (-1 << kSequenceBits) // 序列号最大值，存储范围为 0-2097151
	kMaxDataNode int64 = -1 ^ (-1 << kDataNodeBits) // 数据节点最大值，存储范围为 0-255

	kTimeShift     = kDataNodeBits + kSequenceBits // 时间戳向左的偏移量
	kDataNodeShift = kSequenceBits                 // 数据节点向左的偏移量

	kDataNodeMask = kMaxDataNode << kSequenceBits
)

var (
	ErrDataNodeNotAllowed = errors.New(fmt.Sprintf("xid: data node can't be greater than %d or less than 0", kMaxDataNode))
)

type Option func(*XID) error

// WithDataNode 设置数据节点标识
func WithDataNode(node int64) Option {
	return func(x *XID) error {
		if node < 0 || node > kMaxDataNode {
			return ErrDataNodeNotAllowed
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
	timeOffset int64 // 时间偏移量
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

	var id = (second-this.timeOffset)<<kTimeShift | (this.node << kDataNodeShift) | (this.sequence)
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
	return s & kDataNodeMask >> kDataNodeShift
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
