package a2a

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

// MountA2ARoutes registers per-node A2A endpoints on a Chi router.
// For each node it mounts:
//   - POST /a2a/nodes/{nodeID}         — NodeHandler (JSON-RPC)
//   - GET  /a2a/nodes/{nodeID}/agent-card — AgentCardHandler
//
// It also mounts an aggregate agent card at GET /a2a/agent-card that
// combines skills from all registered nodes.
func MountA2ARoutes(r chi.Router, nodes []*engine.NodeDefinition, executors map[engine.NodeType]engine.NodeExecutorInterface, baseURL string) {
	var allSkills []a2atypes.Skill

	r.Route("/a2a", func(r chi.Router) {
		for _, nodeDef := range nodes {
			nodeDef := nodeDef
			executor, ok := executors[nodeDef.Type]
			if !ok {
				continue
			}

			handler := NewNodeHandler(executor, nodeDef)
			cardHandler := NewAgentCardHandler(nodeDef, baseURL)

			r.Post("/nodes/"+nodeDef.ID, handler.ServeHTTP)
			r.Get("/nodes/"+nodeDef.ID+"/agent-card", cardHandler.ServeHTTP)

			card := AgentCardFromNodeDef(nodeDef, baseURL)
			allSkills = append(allSkills, card.Skills...)
		}

		aggregateCard := a2atypes.AgentCard{
			Name:         "upal",
			Description:  "Upal visual AI workflow platform",
			URL:          baseURL + "/a2a",
			Capabilities: a2atypes.Capabilities{Streaming: false, PushNotifications: false},
			Skills:             allSkills,
			DefaultInputModes:  []string{"text/plain"},
			DefaultOutputModes: []string{"text/plain"},
		}
		r.Get("/agent-card", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(aggregateCard)
		})
	})
}

