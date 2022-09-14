package raft

type RequestVoteArgs struct {
	Term int
	Server int
	LastLogIndex int
	LastLogTerm int
}

type RequestVoteReply struct {
	Term int
	Vote bool
}

func (rf *Raft) RequestVoteHandler(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.term

	candidate_log_size := args.LastLogIndex
	voter_log_size := len(rf.log)

	is_log_upto_date := candidate_log_size >= voter_log_size

	if args.Term < rf.term || !is_log_upto_date {
		rf.Debug(dTicker, "No vote for S%d because old term / log (%d).", args.Server, args.Term)
		reply.Vote = false
		return
	}

	if args.Term > rf.term {
		rf.Debug(dTicker, "Voting to a higher term candidate S%d", args.Server)
		rf.term = args.Term
		rf.voted = -1
		rf.become_follower()
	}

	if rf.voted == -1 || rf.voted == args.Server {
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

func (rf *Raft) request_vote(term int, server int, vote_result chan RequestVoteReply) {
	LastIndex := len(rf.log)
	LastTerm := 0
	
	if LastIndex != 0 {
		LastTerm = rf.log[LastIndex - 1].Term
	}

	args := &RequestVoteArgs{
		Term: term,
		Server: rf.me,
		LastLogIndex: LastIndex,
		LastLogTerm: LastTerm,
	}
	
	reply := &RequestVoteReply{}

	for rf.is_candidate() && !rf.killed() {
		ok := rf.sendRequestVote(server, args, reply)

		if !ok {
			continue
		}

		break
	}

	vote_result <- *reply
}

func (rf *Raft) send_vote_requests(term int) {
	result_ch := make(chan RequestVoteReply, len(rf.peers) - 1)
	
	// Spawn the workers to get the votes 
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}

		go rf.request_vote(term, i, result_ch)
	}

	// Receive the votes
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
		
		if result.Term > term {
			rf.term = result.Term
			rf.mu.Unlock()
			break
		}

		if votes == rf.majority {
			// Become leader and end the election
			rf.Debug(dElection, "Won the election")
			rf.set_leader(true)
			rf.voted = -1
			go rf.heartbeats(rf.term)
			
			rf.mu.Unlock()
			return
		}

		rf.mu.Unlock()
	}

	rf.Debug(dElection, "Lost the election")
}

func (rf *Raft) election() {
	// Get the term
	// Start the main routine
	
	rf.term++
	rf.voted = rf.me
	go rf.send_vote_requests(rf.term)
}