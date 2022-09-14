package raft

import (
	"time"
)

type HeartBeatArgs struct {
	Term int
	Id int
	PrevLogIndex int
	PrevLogTerm int
	Entries [] LogEntry
	LeaderCommit int
}

type HeartBeatReply struct {
	Term int
	Success bool
}

func (rf *Raft) add_entries(entries [] LogEntry, index int) {
	entries_len := len(entries)
	log_len := len(rf.log)

	expected_len := index + entries_len - 1

	if expected_len > log_len {
		// append
		rf.log = rf.log[:index]
		rf.log = append(rf.log, entries...)
	}
}

func (rf *Raft) ConsistencyCheck(PrevLogIndex, PrevLogTerm int) bool {
	if PrevLogIndex == 0 {
		return true
	} else if PrevLogIndex > len(rf.log) {
		return false
	}

	entry := rf.log[PrevLogIndex - 1]

	return entry.Term == PrevLogTerm
}

func (rf *Raft) HeartbeatHandler(args *HeartBeatArgs, reply *HeartBeatReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.term

	if args.Term < rf.term {
		reply.Success = false
		rf.Debug(dHeartbeat, "Rejecting old beat from S%d", args.Id)
		return
	} else if !rf.ConsistencyCheck(args.PrevLogIndex, args.PrevLogTerm) {
		reply.Success = false
		rf.Debug(dHeartbeat, "Consistency check failed S%d", args.Id)
	} else {
		reply.Success = true
		rf.add_entries(args.Entries, args.PrevLogIndex+1)
	}

	rf.term = args.Term
	rf.become_follower()
	rf.reset_election_timeout()
	rf.Debug(dHeartbeat, "Accepted beat from S%d", args.Id)
}

func (rf *Raft) sendRequestBeat(server int, args *HeartBeatArgs, reply *HeartBeatReply) bool {
	ok := rf.peers[server].Call("Raft.HeartbeatHandler", args, reply)
	return ok
}

func (rf *Raft) send_beat(term int, server int) {
	rf.mu.Lock()

	PrevLogIndex := rf.nextIndex[server] - 1
	PrevLogTerm := 0

	var Entries [] LogEntry

	if PrevLogIndex != 0 {
		PrevLogTerm = rf.log[PrevLogIndex - 1].Term
		Entries = rf.log[PrevLogIndex :]
	}

	TopIndex := len(rf.log)

	args := &HeartBeatArgs{
		Term: term,
		Id: rf.me,
		PrevLogIndex: PrevLogIndex,
		PrevLogTerm: PrevLogTerm,
		Entries: Entries,
		LeaderCommit: rf.commitIndex,
	}

	rf.mu.Unlock()

	reply := &HeartBeatReply{}
	
	ok := rf.sendRequestBeat(server, args, reply)

	if !ok {
		return
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.is_leader() || rf.term != term {
		return
	} else if reply.Term > rf.term {
		rf.term = reply.Term
		rf.become_follower()
	} else if len(Entries) == 0 {
		return
	} else if reply.Success {
		// update
		rf.nextIndex[server] = max(TopIndex, rf.nextIndex[server])
		rf.matchIndex[server] = max(TopIndex - 1, rf.matchIndex[server])
	} else {
		// decrement
		rf.nextIndex[server]--
	}
}

func (rf *Raft) heartbeats(term int) {
	// Start heartbeats
	rf.nextIndex = make([] int, len(rf.peers))
	for i := range rf.nextIndex {
		rf.nextIndex[i] = len(rf.log) + 1
	}

	rf.matchIndex = make([] int, len(rf.peers))

	for !rf.killed() && rf.is_leader() {
		// sending beats
		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}

			go rf.send_beat(term, i)
		}

		rf.Debug(dElection, "Beat completed")
		time.Sleep(100*time.Millisecond)
	}
}