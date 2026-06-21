// Package xid 提供基于内存的 Snowflake 风格 uint64 ID 生成器。
//
// 默认布局使用 42 位存储已过毫秒数，10 位存储 worker，12 位存储同一毫秒内的序列号。
// XID 可安全并发使用，并保证同一进程、同一 worker 内生成的 ID 唯一。
package xid

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	defaultWorkerBits       = uint8(10)
	defaultSequenceBits     = uint8(12)
	defaultMaxClockBackward = time.Second
	defaultRetryDelay       = time.Millisecond
	defaultMaxBatchSize     = 200
)

var (
	// ErrInvalidConfig 表示生成器配置不合法。
	ErrInvalidConfig = errors.New("invalid config")

	// ErrClockBackward 表示本地时钟回拨超过配置的容忍范围，或当前时间早于 epoch。
	ErrClockBackward = errors.New("clock moved backward")

	// ErrTimeOverflow 表示已过时间戳无法放入配置出的时间戳位宽。
	ErrTimeOverflow = errors.New("timestamp exceeds configured bit width")
)

// Options 控制 ID 的生成和解析方式。
type Options struct {
	// Epoch 会在组装 ID 前从当前本地时间中扣除。
	Epoch time.Time

	// Worker 标识当前生成器实例。需要跨进程唯一时，调用方必须为不同进程分配不同 worker。
	Worker uint64

	// WorkerBits 是 worker 段位宽。
	WorkerBits uint8

	// SequenceBits 是序列号段位宽。
	SequenceBits uint8

	// MaxClockBackward 是逻辑时钟可吸收的最大本地时钟回拨时长。
	// 超过该值会返回 ErrClockBackward。
	MaxClockBackward time.Duration

	// RetryDelay 是序列号耗尽导致逻辑时钟领先本地时钟时的等待间隔。
	RetryDelay time.Duration

	// MaxBatchSize 限制一次 NextBatch 最多可保留的 ID 数量。
	MaxBatchSize int
}

// Option 修改 XID 配置。
type Option func(*Options)

// WithEpoch 设置生成 ID 使用的自定义起始时间。
func WithEpoch(epoch time.Time) Option {
	return func(opts *Options) {
		opts.Epoch = epoch
	}
}

// WithWorker 设置 worker 标识。
func WithWorker(worker uint64) Option {
	return func(opts *Options) {
		opts.Worker = worker
	}
}

// WithBits 设置 worker 和序列号的位宽。
func WithBits(workerBits, sequenceBits uint8) Option {
	return func(opts *Options) {
		opts.WorkerBits = workerBits
		opts.SequenceBits = sequenceBits
	}
}

// WithMaxClockBackward 设置允许本地时钟回拨的最大时长。
func WithMaxClockBackward(duration time.Duration) Option {
	return func(opts *Options) {
		opts.MaxClockBackward = duration
	}
}

// WithRetryDelay 设置等待本地时间追上逻辑时间时的重试间隔。
func WithRetryDelay(delay time.Duration) Option {
	return func(opts *Options) {
		opts.RetryDelay = delay
	}
}

// WithMaxBatchSize 设置 NextBatch 单次允许的最大数量。
func WithMaxBatchSize(n int) Option {
	return func(opts *Options) {
		opts.MaxBatchSize = n
	}
}

// XID 使用本地内存状态生成 Snowflake 风格 uint64 ID。
type XID struct {
	opts Options

	mu        sync.Mutex
	lastMs    int64
	lastSeq   uint64
	worker    uint64
	workerMax uint64

	workerShift   uint8
	timestampBits uint8
	sequenceMask  uint64
	timestampMask uint64
}

func defaultOptions() Options {
	return Options{
		Epoch:            time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Worker:           0,
		WorkerBits:       defaultWorkerBits,
		SequenceBits:     defaultSequenceBits,
		MaxClockBackward: defaultMaxClockBackward,
		RetryDelay:       defaultRetryDelay,
		MaxBatchSize:     defaultMaxBatchSize,
	}
}

// New 创建基于内存的 XID。
func New(options ...Option) (*XID, error) {
	opts := defaultOptions()
	for _, opt := range options {
		if opt != nil {
			opt(&opts)
		}
	}
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	timestampBits := uint8(64 - int(opts.WorkerBits) - int(opts.SequenceBits))
	g := &XID{
		opts:          opts,
		lastMs:        -1,
		lastSeq:       0,
		worker:        opts.Worker,
		workerMax:     bitMask(opts.WorkerBits),
		workerShift:   opts.SequenceBits,
		timestampBits: timestampBits,
		sequenceMask:  bitMask(opts.SequenceBits),
		timestampMask: bitMask(timestampBits),
	}
	return g, nil
}

func validateOptions(opts Options) error {
	if opts.Epoch.IsZero() {
		return fmt.Errorf("%w: epoch is zero", ErrInvalidConfig)
	}
	if opts.WorkerBits == 0 {
		return fmt.Errorf("%w: worker bits must be positive", ErrInvalidConfig)
	}
	if opts.SequenceBits == 0 {
		return fmt.Errorf("%w: sequence bits must be positive", ErrInvalidConfig)
	}
	if int(opts.WorkerBits)+int(opts.SequenceBits) >= 64 {
		return fmt.Errorf("%w: worker bits plus sequence bits must be less than 64", ErrInvalidConfig)
	}
	if opts.Worker > bitMask(opts.WorkerBits) {
		return fmt.Errorf("%w: worker %d exceeds max %d", ErrInvalidConfig, opts.Worker, bitMask(opts.WorkerBits))
	}
	if opts.MaxClockBackward < 0 {
		return fmt.Errorf("%w: max clock backward cannot be negative", ErrInvalidConfig)
	}
	if opts.RetryDelay <= 0 {
		return fmt.Errorf("%w: retry delay must be positive", ErrInvalidConfig)
	}
	if opts.MaxBatchSize <= 0 {
		return fmt.Errorf("%w: max batch size must be positive", ErrInvalidConfig)
	}
	return nil
}

// Next 返回一个唯一 ID。
func (g *XID) Next(ctx context.Context) (uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		elapsedMs, seq, wait, err := g.reserveRange(1)
		if err != nil {
			return 0, err
		}
		if !wait {
			return g.compose(elapsedMs, seq), nil
		}
		if err = sleep(ctx, g.opts.RetryDelay); err != nil {
			return 0, err
		}
	}
}

// NextBatch 保留 n 个 ID，并按升序返回。
func (g *XID) NextBatch(ctx context.Context, n int) ([]uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if n <= 0 {
		return nil, fmt.Errorf("%w: batch size must be positive", ErrInvalidConfig)
	}
	if n > g.opts.MaxBatchSize {
		return nil, fmt.Errorf("%w: batch size %d exceeds max %d", ErrInvalidConfig, n, g.opts.MaxBatchSize)
	}

	for {
		elapsedMs, seq, wait, err := g.reserveRange(n)
		if err != nil {
			return nil, err
		}
		if wait {
			if err = sleep(ctx, g.opts.RetryDelay); err != nil {
				return nil, err
			}
			continue
		}

		ids := make([]uint64, 0, n)
		for len(ids) < n {
			ids = append(ids, g.compose(elapsedMs, seq))
			seq++
			if seq > g.sequenceMask {
				seq = 0
				elapsedMs++
			}
		}
		return ids, nil
	}
}

func (g *XID) reserveRange(n int) (uint64, uint64, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()
	if now < g.opts.Epoch.UnixMilli() {
		return 0, 0, false, fmt.Errorf("%w: current time is before epoch", ErrClockBackward)
	}
	if g.lastMs >= 0 && now < g.lastMs {
		backward := time.Duration(g.lastMs-now) * time.Millisecond
		if backward > g.opts.MaxClockBackward {
			return 0, 0, false, fmt.Errorf("%w: now=%d last_timestamp=%d", ErrClockBackward, now, g.lastMs)
		}
		now = g.lastMs
	}

	startMs := now
	startSeq := uint64(0)
	if now == g.lastMs {
		startSeq = g.lastSeq + 1
		if startSeq > g.sequenceMask {
			startMs = g.lastMs + 1
			startSeq = 0
		}
	}

	finalOffset := startSeq + uint64(n) - 1
	finalMs := startMs + int64(finalOffset/(g.sequenceMask+1))
	finalSeq := finalOffset % (g.sequenceMask + 1)

	if finalMs > time.Now().UnixMilli()+int64(g.opts.MaxClockBackward/time.Millisecond) {
		return 0, 0, true, nil
	}

	startElapsed := startMs - g.opts.Epoch.UnixMilli()
	finalElapsed := finalMs - g.opts.Epoch.UnixMilli()
	if startElapsed < 0 {
		return 0, 0, false, fmt.Errorf("%w: current time is before epoch", ErrClockBackward)
	}
	if uint64(finalElapsed) > g.timestampMask {
		return 0, 0, false, fmt.Errorf("%w: elapsed milliseconds %d exceeds %d bits", ErrTimeOverflow, finalElapsed, g.timestampBits)
	}

	g.lastMs = finalMs
	g.lastSeq = finalSeq

	return uint64(startElapsed), startSeq, false, nil
}

func (g *XID) compose(elapsedMs, seq uint64) uint64 {
	return (elapsedMs << (g.opts.WorkerBits + g.opts.SequenceBits)) |
		(g.worker << g.workerShift) |
		seq
}

// Worker 返回配置的 worker 标识。
func (g *XID) Worker() uint64 {
	return g.worker
}

// Time 返回 id 中按当前生成器布局解析出的已过毫秒数。
// 加上 Epoch 可得到实际生成时间。
func (g *XID) Time(id uint64) int64 {
	return int64(id >> (g.opts.WorkerBits + g.opts.SequenceBits))
}

// CreatedAt 返回 id 中按当前生成器布局解析出的实际生成时间。
func (g *XID) CreatedAt(id uint64) time.Time {
	return g.opts.Epoch.Add(time.Duration(g.Time(id)) * time.Millisecond)
}

// WorkerOf 返回 id 中按当前生成器布局解析出的 worker 段。
func (g *XID) WorkerOf(id uint64) uint64 {
	return (id >> g.workerShift) & g.workerMax
}

// Sequence 返回 id 中按当前生成器布局解析出的序列号段。
func (g *XID) Sequence(id uint64) uint64 {
	return id & g.sequenceMask
}

func bitMask(bits uint8) uint64 {
	return (uint64(1) << bits) - 1
}

func sleep(ctx context.Context, delay time.Duration) error {
	var timer = time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
