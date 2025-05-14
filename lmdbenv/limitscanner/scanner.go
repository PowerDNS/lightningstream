package limitscanner

import (
	"bytes"
	"errors"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/lmdb-go/lmdbscan"
)

// ErrLimitReached is returned when the LimitScanner reaches the configured limit
var ErrLimitReached = errors.New("limit reached")

func NewLimitScanner(opt Options) (*LimitScanner, error) {
	if opt.Txn == nil {
		panic("limit scanner requires Options.Txn")
	}
	if opt.DBI == 0 {
		panic("limit scanner requires Options.DBI")
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
	opt      Options
	sc       *lmdbscan.Scanner
	count    int
	deadline time.Time
	err      error
}

func (s *LimitScanner) Scan() bool {
	if s.err != nil {
		return false
	}

	if s.count == 0 && !s.opt.Last.IsZero() {
		// Set initial position.
		// If the entry still exists, it will be the first one (basically
		// rescanning the last entry). If it is gone, we will start from the
		// next one after that.
		//s.sc.SetNext(s.opt.Last.key, s.opt.Last.val, lmdb.SetRange, lmdb.Next)
		s.sc.Set(s.opt.Last.key, s.opt.Last.val, lmdb.SetRange)
		if bytes.Equal(s.Key(), s.opt.Last.key) && bytes.Equal(s.Val(), s.opt.Last.val) {
			// Advance one
			s.sc.Set(nil, nil, lmdb.Next)
		}
	}

	// Check number-of-records limit
	if s.opt.LimitRecords > 0 && s.count >= s.opt.LimitRecords {
		s.err = ErrLimitReached
		return false
	}

	// Check time limit
	checkEvery := s.opt.LimitDurationCheckEvery
	if checkEvery > 0 && s.count > 0 && s.count%checkEvery == 0 && !s.deadline.IsZero() {
		if time.Now().After(s.deadline) {
			s.err = ErrLimitReached
			return false
		}
	}

	s.count++
	return s.sc.Scan()
}

func (s *LimitScanner) Last() LimitCursor {
	return LimitCursor{
		key: s.Key(),
		val: s.Val(),
	}
}

func (s *LimitScanner) Cursor() *lmdb.Cursor {
	return s.sc.Cursor()
}

func (s *LimitScanner) Key() []byte {
	return s.sc.Key()
}

func (s *LimitScanner) Val() []byte {
	return s.sc.Val()
}

func (s *LimitScanner) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.sc.Err()
}

func (s *LimitScanner) Close() {
	s.sc.Close()
}
