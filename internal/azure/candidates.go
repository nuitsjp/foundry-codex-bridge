package azure

func CodexCandidate(format string, capabilities map[string]string) bool {
	return isCodexCandidate(format, capabilities)
}
