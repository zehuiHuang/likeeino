package main

import (
	_ "likeeino/agent/agent"
	"likeeino/agent/multiagent/supervisor"
)

func main() {
	//agent.SimpleAgent()
	//agent.ReactAgent()
	//agent.ReactAgent2()
	//multi.MultiAgent()
	//multi2.MultiAgent()
	supervisor.Agent()
}
