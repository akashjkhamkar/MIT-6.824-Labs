package raft

import "time"

type HeartBeatArgs struct {
	Term int
	Id int
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
	args := &HeartBeatArgs{
		Term: term,
		Id: rf.me,
	}

	reply := &HeartBeatReply{}
	ok := rf.sendRequestBeat(server, args, reply)

	// if ok
	// check if term is higher
	// update the nextindex and matchindex

	if !ok {
		return
	}

	rf.mu.Lock()

	if !rf.is_leader() {
		rf.mu.Unlock()
		return
	}

	if reply.Term > rf.term {
		rf.term = reply.Term
		rf.become_follower()
	} else {
		// update the indexes
	}

	rf.mu.Unlock()
}

func (rf *Raft) heartbeats(term int) {
	// Start heartbeats

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