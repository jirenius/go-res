package res

type work struct {
	s      *Service
	wid    string // Worker ID for the work queue
	single [1]func()
	queue  []func() // Callback queue
}

// startWorker starts a new resource worker that will listen for resources to
// process requests on.
func (s *Service) startWorker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.wg.Done()
	// workqueue being nil signals we the service is closing
	for s.workqueue != nil {
		for len(s.workqueue) == 0 {
			s.workcond.Wait()
			if s.workqueue == nil {
				return
			}
		}
		w := s.workqueue[0]
		if len(s.workqueue) == 1 {
			s.workqueue = s.workbuf[:0]
		} else {
			s.workqueue = s.workqueue[1:]
		}
		w.processQueue()
	}
}

func (w *work) processQueue() {
	var f func()
	idx := 0

	for len(w.queue) > idx {
		f = w.queue[idx]
		w.s.mu.Unlock()
		idx++
		f()
		w.s.mu.Lock()
	}
	// Work complete. Delete if it has a work ID.
	if w.wid != "" {
		delete(w.s.rwork, w.wid)
	}
}
