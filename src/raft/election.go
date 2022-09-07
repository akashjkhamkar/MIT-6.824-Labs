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

func (rf *Raft) RequestVoteHandler(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.term

	if (args.Term < rf.term) || (args.Term == rf.term && rf.is_leader()) {
		rf.Debug(dTicker, "No vote for S%d because old term (%d).", args.Server, args.Term)
		reply.Vote = false
		return
	}

	if args.Term > rf.term {
		rf.Debug(dTicker, "Voting to a higher term candidate S%d", args.Server)
		rf.term = args.Term
		rf.voted = -1
		rf.become_follower()
	}

	if (rf.voted == -1 || rf.voted == args.Server) {
		// Grant the vote and convert to follower
		rf.Debug(dTicker, "Voted to S%d", args.Server)
		rf.voted = args.Server
		reply.Term = rf.term
		reply.Vote = true
		rf.reset_election_timeout()
		return
	}

	rf.Debug(dTicker, "No vote for S%d beacause already voted S%d", args.Server, rf.voted)
	reply.Vote = false
}

func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVoteHandler", args, reply)
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
	result_ch := make(chan VoteResult, len(rf.peers) - 1)
	
	// Spawn the workers to get the votes 
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}

		go rf.request_vote(term, i, result_ch)
	}

	// receive the votes
	votes := 1
	for i := 0; i < len(rf.peers) - 1; i++ {
		result := <- result_ch

		if result.Vote {
			votes++
		}

		rf.mu.Lock()

		if !rf.is_candidate() {
			rf.mu.Unlock()
			break
		}
		
		if result.Server_term > term {
			rf.term = result.Server_term
			rf.mu.Unlock()
			break
		}

		if votes == rf.majority {
			// Become leader and end the election
			rf.Debug(dElection, "Won the election")
			rf.set_leader(true)
			// rf.heartsbeats()
			rf.mu.Unlock()
			return
		}

		rf.mu.Unlock()
	}

	rf.Debug(dElection, "Lost the election")
}

func (rf *Raft) election() {
	// get the term
	// start the main routine
	
	rf.term++
	rf.voted = rf.me
	go rf.send_vote_requests(rf.term)
}