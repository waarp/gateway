package migrations

func testMigrations(eng *testEngine, dbType string) {
	// 0.4.0
	testVer0_4_0InitDatabase(eng, dbType)

	// 0.4.2
	testVer0_4_2RemoveHistoryRemoteIDUnique(eng, dbType)

	// 0.5.0
	testVer0_5_0RemoveRulePathSlash(eng, dbType)
	testVer0_5_0CheckRulePathAncestor(eng, dbType)
	testVer0_5_0LocalAgentChangePaths(eng)
	testVer0_5_0LocalAgentsPathsRename(eng)
	testVer0_5_0LocalAgentDisallowReservedNames(eng)
	testVer0_5_0RuleNewPathCols(eng)
	testVer0_5_0RulePathChanges(eng)
	testVer0_5_0AddFilesize(eng)
	testVer0_5_0TransferChangePaths(eng)
	testVer0_5_0TransferFormatLocalPath(eng)
	testVer0_5_0HistoryChangePaths(eng)
	testVer0_5_0LocalAccountsPasswordDecode(eng)
	testVer0_5_0UserPasswordChange(eng, dbType)

	// 0.5.2
	testVer0_5_2FillRemoteTransferID(eng)

	// 0.6.0
	testVer0_6_0AddTransferInfoIsHistory(eng)

	// 0.7.0
	testVer0_7_0AddLocalAgentEnabled(eng)
	testVer0_7_0RevampUsersTable(eng, dbType)
	testVer0_7_0RevampLocalAgentTable(eng)
	testVer0_7_0RevampRemoteAgentTable(eng)
	testVer0_7_0RevampLocalAccountsTable(eng, dbType)
	testVer0_7_0RevampRemoteAccountsTable(eng)
	testVer0_7_0RevampRulesTable(eng)
	testVer0_7_0RevampTasksTable(eng, dbType)
	testVer0_7_0RevampHistoryTable(eng)
	testVer0_7_0RevampTransfersTable(eng)
	testVer0_7_0RevampTransferInfoTable(eng)
	testVer0_7_0RevampCryptoTable(eng)
	testVer0_7_0RevampRuleAccessTable(eng)
	testVer0_7_0AddLocalAgentsAddressUnique(eng)
	testVer0_7_0AddNormalizedTransfersView(eng)

	// 0.7.5
	testVer0_7_5SplitR66TLS(eng)
}
