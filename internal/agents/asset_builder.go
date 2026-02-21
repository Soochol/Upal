package agents

import (
	"fmt"
	"iter"

	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// AssetNodeBuilder creates agents that read an uploaded file's extracted
// text from storage and inject it into session state.
type AssetNodeBuilder struct {
	storage storage.Storage
}

// NewAssetNodeBuilder creates an AssetNodeBuilder backed by the given storage.
func NewAssetNodeBuilder(s storage.Storage) *AssetNodeBuilder {
	return &AssetNodeBuilder{storage: s}
}

func (b *AssetNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeAsset }

func (b *AssetNodeBuilder) Build(nd *upal.NodeDefinition, _ BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	fileID, _ := nd.Config["file_id"].(string)
	if fileID == "" {
		return nil, fmt.Errorf("asset node %q: missing file_id in config", nodeID)
	}

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Asset node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				info, rc, err := b.storage.Get(ctx, fileID)
				if err != nil {
					yield(nil, fmt.Errorf("asset node %q: file %q not found: %w", nodeID, fileID, err))
					return
				}
				if rc != nil {
					rc.Close()
				}

				val := info.ExtractedText
				if val == "" {
					val = fmt.Sprintf("[file: %s]", info.Filename)
				}

				state := ctx.Session().State()
				_ = state.Set(nodeID, val)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(fmt.Sprintf("[asset: %s]", info.Filename))},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = val
				yield(event, nil)
			}
		},
	})
}
