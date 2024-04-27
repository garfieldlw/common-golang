package unique

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	epoch          = int64(1609430400000)
	workerBits     = uint(6)
	typeBits       = uint(8)
	sequenceBits   = uint(8)
	workerMax      = int64(-1 ^ (-1 << workerBits))
	sequenceMask   = int64(-1 ^ (-1 << sequenceBits))
	sequenceShift  = uint(0)
	workerShift    = sequenceBits
	timestampShift = sequenceBits + workerBits
	typeShift      = uint(63) - typeBits
)

type Snowflake struct {
	mu        sync.Mutex
	timestamp int64
	worker    int64
	sequence  int64
	bizType   int64
}

func NewSnowflake(bizType BizType) (*Snowflake, error) {
	workerId := hostnameToInt() & workerMax
	fmt.Println(workerId)
	if workerId < 0 || workerId > workerMax {
		return nil, errors.New("invalid worker id")
	}

	return &Snowflake{
		timestamp: 0,
		worker:    workerId,
		sequence:  0,
		bizType:   int64(bizType),
	}, nil
}

// Generate creates and returns a unique snowflake ID
func (s *Snowflake) Generate() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano() / 1e6

	if s.timestamp == now {
		s.sequence = (s.sequence + 1) & sequenceMask

		if s.sequence == 0 {
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		s.sequence = 0
	}

	s.timestamp = now

	return ((now-epoch)&0x01FFFFFFFFFF)<<timestampShift | (s.worker << workerShift) | (s.sequence << sequenceShift) | s.bizType<<typeShift
}

func (s *Snowflake) GetBizType(id int64) BizType {
	bizType := (id >> typeShift) & 0xFF
	return BizType(int32(bizType))
}

func (s *Snowflake) GetTimestamp(id int64) int64 {
	ts := (id >> timestampShift) & 0x01FFFFFFFFFF
	return ts + epoch
}

func hostnameToInt() int64 {
	var hostname, err = os.Hostname()
	var hash = uint(0)
	var seed = uint(131)
	if err != nil {
		return 0
	}

	for _, k := range hostname {
		hash = hash*seed + uint(k)
	}

	return int64(hash & 0x7FFFFFFF)
}
