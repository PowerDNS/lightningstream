package limitscanner

import (
	"bytes"
	"errors"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/lmdb-go/lmdbscan"
)

func NewLimitScanner(opt Options) (*LimitScanner, error) {
	if opt.Txn == nil {
		return nil, errors.New("limit scanner requires Options.Txn")
	}
	if opt.DBI == 0 {
		return nil, errors.New("limit scanner requires Options.DBI")
	}
	if opt.LimitDurationCheckEvery <= 0 {
		opt.LimitDurationCheckEvery = LimitDurationCheckEveryDefault
	}
	ls := &LimitScanner{
		opt: opt,
		sc:  lmdbscan.New(opt.Txn, opt.DBI),
	}
	if opt.LimitDuration > 0 {
		ls.deadline = time.Now().Add(opt.LimitDuration)
	}
	return ls, nil
}

// LimitDurationCheckEveryDefault determines every how many records we check
// if we exceeded LimitDuration.
// This default of 1000 corresponds to around 1ms of scan time on most systems,
// determining the granularity of our duration checks.
const LimitDurationCheckEveryDefault = 1000

type Options struct {
	// Txn  and DBI for the scan
	Txn *lmdb.Txn
	DBI lmdb.DBI

	// Limits imposed
	LimitRecords            int
	LimitDuration           time.Duration
	LimitDurationCheckEvery int

	// Last record previously scanned
	Last LimitCursor
}

type LimitCursor struct {
	key []byte
	val []byte
}

func (c LimitCursor) IsZero() bool {
	return c.key == nil && c.val == nil
}

// LimitScanner allows iteration over chunks of the LMDB for processing.
// The chunk size can either be given as a number or as a time limit.
type LimitScanner struct {
	opt          Options
	sc           *lmdbscan.Scanner
	count        int
	deadline     time.Time
	limitReached bool
}

func (s *LimitScanner) Scan() bool {
	if s.limitReached {
		return false
	}

	if s.count == 0 && !s.opt.Last.IsZero() {
		// Set initial position.
		// If the entry still exists, it will be the first one (basically
		// rescanning the last entry). If it is gone, we will start from the
		// next one after that.
		// s.sc.SetNext(s.opt.Last.key, s.opt.Last.val, lmdb.SetRange, lmdb.Next)
		s.sc.Set(s.opt.Last.key, s.opt.Last.val, lmdb.SetRange)
		if bytes.Equal(s.Key(), s.opt.Last.key) && bytes.Equal(s.Val(), s.opt.Last.val) {
			// Advance one
			s.sc.Set(nil, nil, lmdb.Next)
		}
	}

	// Check number-of-records limit
	if s.opt.LimitRecords > 0 && s.count >= s.opt.LimitRecords {
		s.limitReached = true
		return false
	}

	// Check time limit
	checkEvery := s.opt.LimitDurationCheckEvery
	if checkEvery > 0 && s.count > 0 && s.count%checkEvery == 0 && !s.deadline.IsZero() {
		if time.Now().After(s.deadline) {
			s.limitReached = true
			return false
		}
	}

	s.count++
	return s.sc.Scan()
}

// Last returns the LimitCursor for the next LimitScanner to use in
// (Options).Last. If this scanner didn't run into any limits Last will return a
// zero LimitCursor.
func (s *LimitScanner) Last() LimitCursor {
	if !s.limitReached {
		return LimitCursor{}
	}
	return LimitCursor{
		key: s.Key(),
		val: s.Val(),
	}
}

func (s *LimitScanner) Key() []byte {
	return s.sc.Key()
}

func (s *LimitScanner) Val() []byte {
	return s.sc.Val()
}

func (s *LimitScanner) Err() error {
	return s.sc.Err()
}

func (s *LimitScanner) Close() {
	s.sc.Close()
}
