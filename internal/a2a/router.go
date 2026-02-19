package a2a

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
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
	var allSkills []Skill

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

		aggregateCard := AgentCard{
			Name:         "upal",
			Description:  "Upal visual AI workflow platform",
			URL:          baseURL + "/a2a",
			Capabilities: Capabilities{Streaming: false, PushNotifications: false},
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

// MountStaticA2ARoutes registers placeholder A2A routes on the main server
// when no workflow is loaded. It provides:
//   - GET  /a2a/agent-card       — basic Upal agent card (no skills)
//   - POST /a2a/nodes/{nodeID}   — JSON-RPC error: node not registered
//   - GET  /a2a/nodes/{nodeID}/agent-card — 404
func MountStaticA2ARoutes(r chi.Router, executors map[engine.NodeType]engine.NodeExecutorInterface, baseURL string) {
	r.Route("/a2a", func(r chi.Router) {
		r.Get("/agent-card", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AgentCard{
				Name:               "upal",
				Description:        "Upal visual AI workflow platform",
				URL:                baseURL + "/a2a",
				Capabilities:       Capabilities{Streaming: false, PushNotifications: false},
				DefaultInputModes:  []string{"text/plain"},
				DefaultOutputModes: []string{"text/plain"},
			})
		})
		r.Post("/nodes/{nodeID}", func(w http.ResponseWriter, r *http.Request) {
			writeJSONRPCError(w, nil, -32601, "Node not registered. Start a workflow run first.")
		})
		r.Get("/nodes/{nodeID}/agent-card", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Node not registered", http.StatusNotFound)
		})
	})
}
