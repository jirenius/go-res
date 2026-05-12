package res

import (
	"sync/atomic"

	nats "github.com/nats-io/nats.go"
)

// ServiceDebugStats contains a point-in-time snapshot of service queue pressure.
type ServiceDebugStats struct {
	Started bool `json:"started"`

	InChannelLen int `json:"inChannelLen"`
	InChannelCap int `json:"inChannelCap"`

	PendingLen    int  `json:"pendingLen"`
	PendingCap    int  `json:"pendingCap"`
	PendingClosed bool `json:"pendingClosed"`

	WorkQueueLen       int `json:"workQueueLen"`
	ResourceWorkQueues int `json:"resourceWorkQueues"`
	// ResourceWorkItems is the number of queued or in-flight items visible in
	// resource work queues. It is not an exact active-handler count.
	ResourceWorkItems  int `json:"resourceWorkItems"`
	ResourceWorkMaxLen int `json:"resourceWorkMaxLen"`

	Subscriptions []ServiceDebugSubscriptionStats `json:"subscriptions,omitempty"`
}

// ServiceDebugSubscriptionStats contains a point-in-time snapshot of a NATS subscription.
type ServiceDebugSubscriptionStats struct {
	Subject   string `json:"subject"`
	Queue     string `json:"queue,omitempty"`
	Delivered int64  `json:"delivered"`
	Dropped   int    `json:"dropped"`
	Error     string `json:"error,omitempty"`
}

// ServiceDebugMessage contains debug information about a pending incoming message.
type ServiceDebugMessage struct {
	Subject       string `json:"subject"`
	Reply         string `json:"reply,omitempty"`
	Bytes         int    `json:"bytes"`
	Data          []byte `json:"data,omitempty"`
	DataTruncated bool   `json:"dataTruncated,omitempty"`
}

// DebugStats returns a point-in-time snapshot of service queue pressure.
func (s *Service) DebugStats() ServiceDebugStats {
	st := ServiceDebugStats{
		Started: atomic.LoadInt32(&s.state) == stateStarted,
	}

	s.mu.Lock()
	if s.inCh != nil {
		st.InChannelLen = len(s.inCh)
		st.InChannelCap = cap(s.inCh)
	}
	st.WorkQueueLen = len(s.workqueue)
	st.ResourceWorkQueues = len(s.rwork)
	for _, w := range s.rwork {
		n := len(w.queue)
		st.ResourceWorkItems += n
		if n > st.ResourceWorkMaxLen {
			st.ResourceWorkMaxLen = n
		}
	}
	subs := append([]*nats.Subscription(nil), s.subs...)
	s.mu.Unlock()

	s.pendingMu.Lock()
	st.PendingLen = len(s.pending)
	st.PendingCap = cap(s.pending)
	st.PendingClosed = s.pendingClosed
	s.pendingMu.Unlock()

	if len(subs) > 0 {
		st.Subscriptions = make([]ServiceDebugSubscriptionStats, 0, len(subs))
		for _, sub := range subs {
			dst := ServiceDebugSubscriptionStats{
				Subject: sub.Subject,
				Queue:   sub.Queue,
			}
			if delivered, err := sub.Delivered(); err != nil {
				dst.Error = err.Error()
			} else {
				dst.Delivered = delivered
			}
			if dropped, err := sub.Dropped(); err != nil {
				if dst.Error == "" {
					dst.Error = err.Error()
				}
			} else {
				dst.Dropped = dropped
			}
			st.Subscriptions = append(st.Subscriptions, dst)
		}
	}

	return st
}

// DebugPendingMessages returns a point-in-time copy of pending incoming messages.
//
// It only inspects the service pending queue. Go channels cannot be safely
// peeked, so messages still buffered in the NATS input channel are not included.
func (s *Service) DebugPendingMessages(limit int, maxPayloadBytes int) []ServiceDebugMessage {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	if limit <= 0 || limit > len(s.pending) {
		limit = len(s.pending)
	}

	msgs := make([]ServiceDebugMessage, 0, limit)
	for _, m := range s.pending[:limit] {
		dm := ServiceDebugMessage{
			Subject: m.Subject,
			Reply:   m.Reply,
			Bytes:   len(m.Data),
		}
		switch {
		case maxPayloadBytes < 0:
			dm.Data = append([]byte(nil), m.Data...)
		case maxPayloadBytes > 0:
			n := len(m.Data)
			if n > maxPayloadBytes {
				n = maxPayloadBytes
				dm.DataTruncated = true
			}
			dm.Data = append([]byte(nil), m.Data[:n]...)
		}
		msgs = append(msgs, dm)
	}
	return msgs
}
