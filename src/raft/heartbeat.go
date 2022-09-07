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

func (rf *Raft) send_beat(term int, server int, result_ch chan HeartBeatReply) {
	args := &HeartBeatArgs{
		Term: term,
		Id: rf.me,
	}

	reply := &HeartBeatReply{}

	for rf.is_leader() && !rf.killed() {
		ok := rf.sendRequestBeat(server, args, reply)

		if !ok {
			continue
		}

		break
	}

	result_ch <- *reply
}

func (rf *Raft) heartbeats(term int) {
	// Start heartbeats

	for !rf.killed() && rf.is_leader() {
		// sending beats
		result_ch := make(chan HeartBeatReply, len(rf.peers) - 1)

		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}

			go rf.send_beat(term, i, result_ch)
		}

		success := 1
		for i := 0; i < len(rf.peers) - 1; i++ {
			result := <- result_ch
	
			if result.Success {
				success++
			}
	
			rf.mu.Lock()
	
			if !rf.is_leader() {
				rf.mu.Unlock()
				return
			}
			
			if result.Term > term {
				rf.term = result.Term
				rf.become_follower()
				rf.mu.Unlock()
				break
			}
	
			if success == rf.majority {
				// Become leader and end the election
				rf.Debug(dElection, "Beat completed")
				rf.mu.Unlock()
				break
			}
	
			rf.mu.Unlock()
		}

		time.Sleep(100*time.Millisecond)
	}
}