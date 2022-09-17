package raft

import (
	"math/rand"
	"sync/atomic"
	"time"
)

func (rf *Raft) is_election_timeout() bool {
	current_time := time.Now().UnixNano() / int64(time.Millisecond)
	diff := current_time - rf.get_last_reset_time()

	return diff > int64(rf.election_timeout)
}

func (rf *Raft) reset_election_timeout() {
	current_time := time.Now().UnixNano() / int64(time.Millisecond)
	atomic.StoreInt64(&rf.last_reset_time, current_time)
}

func (rf *Raft) get_last_reset_time() int64 {
	return atomic.LoadInt64(&rf.last_reset_time)
}

func (rf *Raft) is_candidate() bool {
	z := atomic.LoadInt32(&rf.candidate)
	return z == 1
}

func (rf *Raft) set_candidate(set bool) {
	var value int32

	if set {
		value = 1
		rf.set_leader(false)
	}

	atomic.StoreInt32(&rf.candidate, value)
}

func (rf *Raft) is_leader() bool {
	z := atomic.LoadInt32(&rf.leader)
	return z == 1
}

func (rf *Raft) set_leader(set bool) {
	var value int32

	if set {
		value = 1
		rf.set_candidate(false)
	}

	atomic.StoreInt32(&rf.leader, value)
}

func (rf *Raft) get_current_term() int {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	return rf.term
}

func (rf *Raft) become_follower() {
	rf.set_leader(false)
	rf.set_candidate(false)
}

func (rf *Raft) get_random_sleeping_time() int {
	random_time := rand.Intn(rf.random_sleep_time_range)
	total := rf.base_sleep_time + random_time
	
	return total
}

func (rf *Raft) random_sleep() {
	duration := time.Duration(rf.get_random_sleeping_time())
	total_in_miliseconds := duration * time.Millisecond
	time.Sleep(total_in_miliseconds)
}

func (rf *Raft) random_sleep_2(between int) {
	duration := time.Duration(rand.Intn(between))
	time.Sleep(duration * time.Millisecond)
}

func max(a , b int) int {
	if a > b {
		return a
	}

	return b
}

func min(a , b int) int {
	if a < b {
		return a
	}

	return b
}