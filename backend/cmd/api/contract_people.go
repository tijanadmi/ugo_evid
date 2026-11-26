package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
)

func (s *Server) ContractPeople(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	user, ok := s.documentAccess(c, id, c.Request.Method != "GET")
	if !ok {
		return
	}
	store, ok := s.store.(repository.ContractPeopleStore)
	if !ok {
		c.JSON(503, gin.H{"error": "Angažovanja nisu dostupna."})
		return
	}
	if c.Request.Method == "GET" {
		items, err := store.ListContractPeople(c.Request.Context(), id)
		if err != nil {
			documentError(c, err)
			return
		}
		candidates, err := store.ContractPeopleCandidates(c.Request.Context(), id)
		if err != nil {
			documentError(c, err)
			return
		}
		c.JSON(200, gin.H{"items": items, "candidates": candidates})
		return
	}
	var req models.ContractPersonChange
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "Proverite izabrano lice, ulogu i verziju."})
		return
	}
	if err := store.ChangeContractPerson(c.Request.Context(), id, user, req, c.Request.Method == "DELETE"); err != nil {
		switch {
		case errors.Is(err, repository.ErrContractPeople):
			c.JSON(409, gin.H{"error": err.Error()})
		case errors.Is(err, repository.ErrContactLocked):
			contactError(c, err)
		case errors.Is(err, repository.ErrContractUnavailable):
			c.JSON(403, gin.H{"error": err.Error()})
		default:
			documentError(c, err)
		}
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func (s *Server) ContactEngagements(c *gin.Context) {
	person, _, ok := s.accessibleContact(c)
	if !ok {
		return
	}
	store, ok := s.store.(repository.ContractPeopleStore)
	if !ok {
		c.JSON(503, gin.H{"error": "Angažovanja nisu dostupna."})
		return
	}
	items, err := store.ContactEngagements(c.Request.Context(), person.ID)
	if err != nil {
		contactError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items})
}
