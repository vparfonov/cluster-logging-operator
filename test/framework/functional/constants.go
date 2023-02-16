package functional

import (
	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"time"
)

const (
	applicationLog      = "application"
	auditLog            = "audit"
	ovnAuditLog         = "ovn"
	k8sAuditLog         = "k8s"
	oauthAuditLog       = "oauth-audit-logs"
	OpenshiftAuditLog   = "openshift-audit-logs"
	OAuthServerAuditLog = "oauth-server"
	ApplicationLogFile  = "/tmp/app-logs"
	FunctionalNodeName  = "functional-test-node"
)

var (
	maxDuration          time.Duration
	defaultRetryInterval time.Duration

	logPath = map[string]string{
		applicationLog:      "/var/log/pods",
		auditLog:            "/var/log/audit",
		ovnAuditLog:         "/var/log/ovn",
		oauthAuditLog:       "/var/log/oauth-apiserver",
		OpenshiftAuditLog:   "/var/log/openshift-apiserver",
		k8sAuditLog:         "/var/log/kube-apiserver",
		OAuthServerAuditLog: "/var/log/oauth-server",
	}
	outputLogFile = map[string]map[string]string{
		logging.OutputTypeFluentdForward: {
			applicationLog:                  ApplicationLogFile,
			auditLog:                        "/tmp/audit-logs",
			ovnAuditLog:                     "/tmp/audit-logs",
			k8sAuditLog:                     "/tmp/audit-logs",
			logging.InputNameInfrastructure: "/tmp/infra-logs",
		},
		logging.OutputTypeSyslog: {
			applicationLog:                  "/tmp/infra.log",
			auditLog:                        "/tmp/infra.log",
			k8sAuditLog:                     "/tmp/infra.log",
			ovnAuditLog:                     "/tmp/infra.log",
			logging.InputNameInfrastructure: "/tmp/infra.log",
		},
		logging.OutputTypeKafka: {
			applicationLog: "/var/log/app.log",
			auditLog:       "/var/log/infra.log",
			k8sAuditLog:    "/var/log/audit.log",
			ovnAuditLog:    "/var/log/ovnaudit.log",
		},
	}
)

func init() {
	maxDuration, _ = time.ParseDuration("5m")
	defaultRetryInterval, _ = time.ParseDuration("10s")
}
