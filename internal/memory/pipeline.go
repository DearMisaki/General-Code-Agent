package memory

import "context"

type MemoryWritePipelineOptions struct {
	Store   MemoryStore
	Policy  *MemoryPolicyFilter
	Deduper *Deduper
	Audit   *MemoryAuditWriter
	AppID   string
}

type MemoryWritePipeline struct {
	opts MemoryWritePipelineOptions
}

type MemoryWriteResult struct {
	Added     int
	Updated   int
	Skipped   int
	Conflicts int
	Filtered  int
}

func NewMemoryWritePipeline(opts MemoryWritePipelineOptions) *MemoryWritePipeline {
	if opts.Policy == nil {
		opts.Policy = NewMemoryPolicyFilter(PolicyOptions{})
	}
	if opts.Store != nil && opts.Deduper == nil {
		opts.Deduper = NewDeduper(opts.Store, DedupeOptions{})
	}
	return &MemoryWritePipeline{opts: opts}
}

func (p *MemoryWritePipeline) WriteCandidates(ctx context.Context, candidates []MemoryCandidate) (MemoryWriteResult, error) {
	var result MemoryWriteResult
	if p == nil || p.opts.Store == nil {
		return result, nil
	}
	for _, candidate := range candidates {
		if candidate.AppID == "" {
			candidate.AppID = p.opts.AppID
		}
		decision := p.opts.Policy.Filter(candidate)
		if !decision.Accepted {
			result.Filtered++
			_ = p.opts.Audit.Append(MemoryAuditRecord{
				Event:   EventMemoryCandidateFiltered,
				Summary: string(decision.Reason),
			})
			continue
		}
		candidate = decision.Candidate
		_ = p.opts.Audit.Append(MemoryAuditRecord{Event: EventMemoryCandidateAccepted, Summary: candidate.Memory})
		dedupe := DedupeDecision{Action: DedupeAdd, Memory: candidate}
		var err error
		if p.opts.Deduper != nil {
			dedupe, err = p.opts.Deduper.Decide(ctx, candidate)
			if err != nil {
				return result, err
			}
		}
		switch dedupe.Action {
		case DedupeSkip:
			result.Skipped++
			id := ""
			if dedupe.Match != nil {
				id = dedupe.Match.ID
			}
			_ = p.opts.Audit.Append(MemoryAuditRecord{Event: EventMemoryWriteSkippedDuplicate, MemoryID: id, Summary: dedupe.Reason})
		case DedupeUpdateMetadata:
			if dedupe.Match == nil {
				result.Skipped++
				continue
			}
			if err := p.opts.Store.Update(ctx, dedupe.Match.ID, MemoryPatch{Metadata: candidate.Metadata}); err != nil {
				return result, err
			}
			result.Updated++
			_ = p.opts.Audit.Append(MemoryAuditRecord{Event: EventMemoryWriteUpdated, MemoryID: dedupe.Match.ID, Summary: dedupe.Reason})
		case DedupeConflict:
			result.Conflicts++
			id := ""
			if dedupe.Match != nil {
				id = dedupe.Match.ID
			}
			_ = p.opts.Audit.Append(MemoryAuditRecord{Event: EventMemoryWriteConflictDetected, MemoryID: id, Summary: dedupe.Reason})
		default:
			added, err := p.opts.Store.Add(ctx, AddMemoryRequest{Candidates: []MemoryCandidate{candidate}})
			if err != nil {
				return result, err
			}
			result.Added += len(added)
			for _, mem := range added {
				_ = p.opts.Audit.Append(MemoryAuditRecord{Event: EventMemoryWriteAdded, MemoryID: mem.ID, Summary: mem.Memory})
			}
		}
	}
	return result, nil
}
