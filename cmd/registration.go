package cmd

import "sync"

var registerCommandsOnce sync.Once

// Cobra commands are package globals, so their relationships must be wired once
// even though NewRootCmd can be called repeatedly by tests.
func registerCommands() {
	registerCommandsOnce.Do(func() {
		registerAgentCommands()
		registerAPICommands()
		registerAuditCommands()
		registerAuditSessionsCommands()
		registerAuthCommands()
		registerAuthProviderCommands()
		registerDeployTokenCommands()
		registerGistCommands()
		registerHookCommands()
		registerIssueCommands()
		registerIssueMilestoneCommands()
		registerKGCommands()
		registerKGExtendedCommands()
		registerLabelCommands()
		registerMCPCommands()
		registerNamespaceCommands()
		registerNamespaceProfileCommands()
		registerNotificationCommands()
		registerOAuthClientsCommands()
		registerOrgInvitationCommands()
		registerOrgMemberCommands()
		registerPRCommands()
		registerPRCollabCommands()
		registerProjectCommands()
		registerProvenanceCommands()
		registerReleaseCommands()
		registerRepoCommands()
		registerRepoBranchCommands()
		registerRepoBrowseCommands()
		registerRepoCommitCommands()
		registerRepoGitCommands()
		registerRepoInsightsCommands()
		registerRepoTagCommands()
		registerRepoTopicsCommands()
		registerSearchCommands()
		registerSessionCommands()
		registerWorkloadCommands()
		registerSelfHostCommands()
		registerSelfHostLicenseCommands()
		registerSSHKeyCommands()
		registerTokenCommands()
		registerWebhookCommands()
	})
}
