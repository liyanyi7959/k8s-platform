package provisioning

import (
	model "k8s-platform-backend/internal/provisioning/domain"
	sshtransport "k8s-platform-backend/internal/transport/ssh"
)

func deploymentSSHConfig(server model.DeployServer) sshtransport.Config {
	return sshtransport.Config{Host: server.IP, Port: server.SSHPort, User: server.User, AuthType: server.AuthType}
}
