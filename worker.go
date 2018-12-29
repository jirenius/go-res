package res

type work struct {
	s     *Service
	wid   string   // Worker ID for the work queue
	queue []func() // Callback queue
}

// startWorker starts a new resource worker that will listen for resources to
// process requests on.
func (s *Service) startWorker(ch chan *work) {
	for w := range ch {
		w.processQueue()
	}
	s.wg.Done()
}

func (w *work) processQueue() {
	var f func()
	idx := 0

	w.s.mu.Lock()
	for len(w.queue) > idx {
		f = w.queue[idx]
		w.s.mu.Unlock()
		idx++
		f()
		w.s.mu.Lock()
	}
	// Work complete
	delete(w.s.rwork, w.wid)
	w.s.mu.Unlock()
}
