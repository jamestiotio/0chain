package chain

//Stats - a struct to store various runtime stats of the chain
type Stats struct {
	MissedBlocks              int64
	RollbackCount             int64
	LongestRollbackLength     int8
	ZeroNotarizedBlocksCount  int64
	MultiNotarizedBlocksCount int64
	MaxNotarizedBlocksCount   int8
}

var NotariedBlocksCounts []int64
