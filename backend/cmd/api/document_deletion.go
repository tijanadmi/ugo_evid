package api

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/repository"
)

func (s *Server) finishDocumentDeletion(ctx context.Context, id int) (result error) {
	defer func() {
		if reporter, ok := s.store.(repository.DeletionAttemptStore); ok {
			reportCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			message := ""
			if result != nil {
				message = result.Error()
			}
			if err := reporter.RecordDocumentDeletionAttempt(reportCtx, id, message); err != nil {
				log.Error().Err(err).Int("document_id", id).Msg("cannot record deletion attempt")
			}
		}
	}()
	store, ok := s.store.(repository.DocumentDeletionStore)
	if !ok {
		return errors.New("document deletion store unavailable")
	}
	// Do not create a missing/misconfigured document root during deletion.
	root, err := os.OpenRoot(s.config.DocumentRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	return store.FinishDocumentDeletion(ctx, id, root)
}

// Called once per backend process. Oracle row locks serialize concurrent
// workers and the immediate API attempt; no in-memory ownership is required.
func (s *Server) RunDocumentDeletionWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		s.retryDocumentDeletions(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) retryDocumentDeletions(ctx context.Context) {
	store, ok := s.store.(repository.DocumentDeletionStore)
	if !ok {
		return
	}
	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	ids, err := store.PendingDocumentDeletions(scanCtx)
	cancel()
	if err != nil {
		log.Error().Err(err).Msg("cannot list pending document deletions")
		return
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		attempt, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := s.finishDocumentDeletion(attempt, id)
		cancel()
		if err != nil {
			log.Error().Err(err).Int("document_id", id).Msg("document deletion pending retry")
		}
	}
}
