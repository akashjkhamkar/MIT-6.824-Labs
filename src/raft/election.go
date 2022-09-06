package raft

type RequestVoteArgs struct {
	Term int
	Server int
}

type RequestVoteReply struct {
	Term int
	Vote bool
}

type VoteResult struct {
	Term int
	Vote bool
	Server_term int
}

func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) request_vote(term int, server int, vote_result chan VoteResult) {
	args := &RequestVoteArgs{
		Term: term,
		Server: rf.me,
	}
	
	reply := &RequestVoteReply{}

	result := VoteResult{
		Term: term,
		Server_term: -1,
		Vote: false,
	}

	for rf.is_candidate() && !rf.killed() {
		ok := rf.sendRequestVote(server, args, reply)

		if !ok {
			continue
		}

		result.Vote = reply.Vote
		result.Server_term = reply.Term

		break
	}

	vote_result <- result
}

func (rf *Raft) send_vote_requests(term int) {
	result_ch := make(chan VoteResult)
	
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}

		go rf.request_vote(term, i, result_ch)
	}

	for i := 0; i < len(rf.peers) - 1; i++ {
		// receive
	}
}

func (rf *Raft) election() {
	// get the term
	// start the main routine
	
	rf.term++
	rf.voted = rf.me
	go rf.send_vote_requests(rf.term)
}