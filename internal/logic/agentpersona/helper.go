package agentpersona

import (
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func personIDFromInput(input *request.UriAgentPersonaIDRequest) (uuid.UUID, error) {
	if input == nil {
		return uuid.Nil, xerr.BadRequest("智能体角色 ID 无效")
	}
	id, err := uuid.Parse(input.PersonaID)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("智能体角色 ID 无效")
	}
	return id, nil
}
