package flows

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/audit"
	"loginer/internal/api/respond"
	"loginer/internal/model"
	"loginer/internal/store"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger
}

// New returns a Handler.
func New(st *store.Store, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, audit: recorder, log: log}
}

// List returns every flow, the steps one can be made of, and how many
// applications each flow signs people in for.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	if _, err := h.store.DefaultLoginFlow(ctx); err != nil {
		respond.Failure(c, h.log, err, "loading the default login flow failed")
		return
	}

	flows, err := h.store.LoginFlows(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "listing login flows failed")
		return
	}

	counts, err := h.store.LoginFlowApplications(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "counting the applications on each login flow failed")
		return
	}

	c.JSON(http.StatusOK, newListResponse(flows, counts))
}

// Get returns one flow.
func (h *Handler) Get(c *gin.Context) {
	flow, ok := h.find(c)
	if !ok {
		return
	}

	h.answer(c, http.StatusOK, flow)
}

// Create adds a flow, disabled until an administrator enables it.
func (h *Handler) Create(c *gin.Context) {
	var req flowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	flow := model.DefaultLoginFlow()
	flow.IsDefault = false
	flow.IsEnabled = false
	flow.Name, flow.Slug, flow.Description = "", "", ""

	if err := req.applyTo(&flow, true); err != nil {
		respond.Failure(c, h.log, err, "validating a login flow failed")
		return
	}

	if err := h.store.CreateLoginFlow(c.Request.Context(), &flow); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			respond.Fail(c, slugTaken)
			return
		}

		respond.Failure(c, h.log, err, "creating a login flow failed")
		return
	}

	h.audit.RecordWith(c, "login_flow.created", targetType, flow.ID.String(), map[string]any{
		"flow": flow.Name,
	})

	h.answer(c, http.StatusCreated, &flow)
}

// Update changes a flow. Its identifier is not among them: an application and
// an export both name a flow by it.
func (h *Handler) Update(c *gin.Context) {
	flow, ok := h.find(c)
	if !ok {
		return
	}

	var req flowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	// A flow that is the default stays the default until another is made
	// one: unmarking it here would leave nothing to fall back to.
	wasDefault := flow.IsDefault

	if err := req.applyTo(flow, false); err != nil {
		respond.Failure(c, h.log, err, "validating a login flow failed")
		return
	}

	if wasDefault && !flow.IsDefault {
		respond.Fail(c, keepADefault)
		return
	}

	if err := h.store.SaveLoginFlow(c.Request.Context(), flow); err != nil {
		respond.Failure(c, h.log, err, "updating a login flow failed")
		return
	}

	h.audit.RecordWith(c, "login_flow.updated", targetType, flow.ID.String(), map[string]any{
		"flow": flow.Name, "is_enabled": flow.IsEnabled, "is_default": flow.IsDefault,
	})

	h.answer(c, http.StatusOK, flow)
}

// Delete removes a flow. The applications signing people in with it fall back
// to the default, which is why the default itself cannot go.
func (h *Handler) Delete(c *gin.Context) {
	flow, ok := h.find(c)
	if !ok {
		return
	}

	err := h.store.DeleteLoginFlow(c.Request.Context(), flow)
	switch {
	case errors.Is(err, store.ErrDefaultLoginFlow):
		respond.Fail(c, defaultStays)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "deleting a login flow failed")
		return
	}

	h.audit.RecordWith(c, "login_flow.deleted", targetType, flow.ID.String(), map[string]any{
		"flow": flow.Name,
	})

	c.Status(http.StatusNoContent)
}

func (h *Handler) find(c *gin.Context) (*model.LoginFlow, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, flowAbsent)
		return nil, false
	}

	flow, err := h.store.LoginFlow(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, flowAbsent)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading a login flow failed")
		return nil, false
	}

	return flow, true
}

func (h *Handler) answer(c *gin.Context, status int, flow *model.LoginFlow) {
	counts, err := h.store.LoginFlowApplications(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "counting the applications on each login flow failed")
		return
	}

	c.JSON(status, response{Flow: newFlowResponse(*flow, counts[flow.ID])})
}
