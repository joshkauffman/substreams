package stage

import (
	"github.com/streamingfast/substreams/block"
	pbssinternal "github.com/streamingfast/substreams/pb/sf/substreams/intern/v2"
	"github.com/streamingfast/substreams/reqctx"
)

type SegmentState int

const (
	SegmentPending SegmentState = iota // The job needs to be scheduled, no complete store exists at the end of its Range, nor any partial store for the end of this segment.
	SegmentPartialPresent
	SegmentScheduled // Means the job was scheduled for execution
	SegmentMerging   // A partial is being merged
	SegmentCompleted // End state. A store has been snapshot for this segment, and we have gone over in the per-request squasher
)

// SegmentID can be used as a key, and points to the respective indexes of
// Stages::segments[SegmentID.Segment][SegmentID.Stage]
type SegmentID struct {
	Segment int
	Stage   int
	Range   *block.Range
}

func (i SegmentID) GenRequest(req *reqctx.RequestDetails) *pbssinternal.ProcessRangeRequest {
	return &pbssinternal.ProcessRangeRequest{
		StartBlockNum: i.Range.StartBlock,
		StopBlockNum:  i.Range.ExclusiveEndBlock,
		Modules:       req.Modules,
		OutputModule:  req.OutputModule,
		Stage:         uint32(i.Stage),
	}
}

func (s SegmentState) String() string {
	switch s {
	case SegmentPending:
		return "Pending"
	case SegmentPartialPresent:
		return "PartialPresent"
	case SegmentScheduled:
		return "Scheduled"
	case SegmentMerging:
		return "Merging"
	case SegmentCompleted:
		return "Completed"
	default:
		return "Unknown"
	}
}
