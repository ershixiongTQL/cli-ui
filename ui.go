//A simple cli-based user interface
package cliui

import (
	"github.com/ershixiongTQL/cli-ui/frontendtelnet"
	"github.com/ershixiongTQL/cli-ui/interfaces"
)

type Agent struct {
	agentType interfaces.UI_AGENT_FE_TYPE
	agent     interfaces.UIAgentInterface
}

func (agent *Agent) Start() error {
	return agent.agent.Start()
}

func (agent *Agent) Stop() {
	agent.agent.Stop()
}

func (agent *Agent) FrontEndType() interfaces.UI_AGENT_FE_TYPE {
	return agent.agentType
}

func Create(frontType string, getPrompt func() string, getBanner func() string, backendConfigPath string, listenOn string) (agent *Agent) {

	agent = new(Agent)

	if frontType == "telnet" {

		server := frontendtelnet.Server{}

		backend := backendPrepare(backendConfigPath)

		if backend == nil {
			return nil
		}

		server.Init(frontendtelnet.Config{
			GetPrompt: getPrompt,
			GetBanner: getBanner,
			Backend:   backend,
			ListenOn:  listenOn,
		})

		agent.agent = &server
		agent.agentType = interfaces.UI_AGENT_FE_TYPE_TELNET

	} else {
		return nil
	}

	return
}
