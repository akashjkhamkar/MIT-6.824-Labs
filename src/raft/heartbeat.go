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

func (rf *Raft) HeartbeatHandler(args *HeartBeatArgs, reply *HeartBeatReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.term

	if args.Term < rf.term {
		reply.Success = false
		rf.Debug(dHeartbeat, "Rejecting old beat from S%d", args.Id)
		return
	}

	rf.term = args.Term
	rf.become_follower()
	rf.reset_election_timeout()
	
	reply.Success = true
}

func (rf *Raft) sendRequestBeat(server int, args *HeartBeatArgs, reply *HeartBeatReply) bool {
	ok := rf.peers[server].Call("Raft.HeartbeatHandler", args, reply)
	return ok
}

func (rf *Raft) send_beat(term int, server int) {
	rf.mu.Lock()

	PrevLogIndex := rf.nextIndex[server] - 1
	PrevLogTerm := rf.log[PrevLogIndex - 1].Term
	Entries := rf.log[PrevLogIndex+1:]

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
	defer rf.mu.Lock()

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
		rf.nextIndex[i] = len(rf.log)
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