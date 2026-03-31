package event

import (
	"net/http"

	"github.com/Chuckzera1/event-source-todo-app/internal/application/dto"
	"github.com/Chuckzera1/event-source-todo-app/internal/application/usecases/event"
	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
	"github.com/gin-gonic/gin"
)

type CreateEventHandler struct {
	createEventUseCase event.CreateEventUseCase
}

func (h *CreateEventHandler) Handle(c *gin.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.createEventUseCase.Execute(c.Request.Context(), domain.Event{
		Aggregate: req.Aggregate,
		Version:   req.Version,
		Data:      req.Data,
		Timestamp: req.Timestamp,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event created successfully"})
}

func NewCreateEventHandler(createEventUseCase event.CreateEventUseCase) *CreateEventHandler {
	return &CreateEventHandler{createEventUseCase: createEventUseCase}
}
