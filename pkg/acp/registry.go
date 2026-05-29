package acp

// Registry manages ACP agent registration and trajectory sharing.
type Registry interface {
	Register(agent ACPAgent) error
	Discover(capability string) ([]ACPAgent, error)
	ShareTrajectory(t Trajectory) error
}
