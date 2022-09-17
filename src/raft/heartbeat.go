package raft

import (
	"sort"
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
	JumpIndex int
}

func (rf *Raft) set_commit_index(LeaderCommit int) {
	if LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(LeaderCommit, len(rf.log))
	}
}

func (rf *Raft) add_entries(entries [] LogEntry, index int) {
	expected_length := index - 1 + len(entries)
	
	if expected_length < len(rf.log) {
		// swap the middle, keep the rest
		top := rf.log[index - 1 + len(entries):]
		bottom := rf.log[:index - 1]

		var final [] LogEntry
		final = append(final, bottom...)
		final = append(final, entries...)
		final = append(final, top...)

		rf.log = final
		return
	}

	rf.log = rf.log[:index - 1]
	rf.log = append(rf.log, entries...)
}

func (rf *Raft) consistency_jump_index(PrevLogIndex int) int {
	index := min(PrevLogIndex, len(rf.log)) - 1
	for_term := rf.log[index].Term

	for i := index - 1; i >= 0; i-- {
		term := rf.log[i].Term

		if term != for_term {
			return i + 2
		}
	}

	return 1
}

func (rf *Raft) ConsistencyCheck(PrevLogIndex, PrevLogTerm int) bool {
	if PrevLogIndex == 0 {
		return true
	} else if PrevLogIndex > len(rf.log) {
		return false
	}

	entry := rf.log[PrevLogIndex - 1]

	if entry.Term != PrevLogTerm {
		// deleting all successesive entries
		rf.log = rf.log[:PrevLogIndex - 1]
		return false
	}

	return true
}

func (rf *Raft) HeartbeatHandler(args *HeartBeatArgs, reply *HeartBeatReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.term
	
	rf.Debug(dHeartbeat, "Beat - ", args)
	if args.Term < rf.term {
		reply.Success = false
		rf.Debug(dHeartbeat, "Rejecting old beat from S%d", args.Id)
		return
	} else if !rf.ConsistencyCheck(args.PrevLogIndex, args.PrevLogTerm) {
		reply.Success = false
		reply.JumpIndex = rf.consistency_jump_index(args.PrevLogIndex)
		rf.Debug(dHeartbeat, "Consistency check failed S%d", args.Id)
	} else {
		reply.Success = true
		rf.add_entries(args.Entries, args.PrevLogIndex+1)
		rf.set_commit_index(args.LeaderCommit)
	}

	rf.executer()
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
	
	if PrevLogIndex != 0 {
		PrevLogTerm = rf.log[PrevLogIndex - 1].Term
	}

	var Entries [] LogEntry

	if len(rf.log) != 0 {
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
	} else if reply.Success && len(Entries) != 0 {
		// update
		rf.nextIndex[server] = max(TopIndex + 1, rf.nextIndex[server])
		rf.matchIndex[server] = max(TopIndex, rf.matchIndex[server])
	} else if !reply.Success {
		// decrement
		rf.nextIndex[server] = reply.JumpIndex
	}
}

func (rf *Raft) commiter() {
	// sort and return the majorith number
	// making a copy of array because otherwise it will swap server infos
	var sorted_matchindexes [] int
	
	sorted_matchindexes = append(sorted_matchindexes, rf.matchIndex...)
	sorted_matchindexes[rf.me] = len(rf.log)

	sort.Slice(sorted_matchindexes, func(i, j int) bool {
		return sorted_matchindexes[i] > sorted_matchindexes[j]
	})

	new_index := sorted_matchindexes[rf.majority - 1]

	if new_index == 0 || len(rf.log) == 0 {
		return
	}

	if rf.log[new_index - 1].Term == rf.term {
		rf.commitIndex = new_index
	}

	rf.executer()

	time.Sleep(10*time.Millisecond)
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
		rf.mu.Lock()

		if term != rf.term {
			rf.mu.Unlock()
			break
		}

		rf.mu.Unlock()
		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}

			go rf.send_beat(term, i)
		}

		rf.commiter()
		time.Sleep(100*time.Millisecond)
	}
}