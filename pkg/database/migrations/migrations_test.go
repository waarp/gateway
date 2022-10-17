package migrations

func testMigrations(eng *testEngine, dbType string) {
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
}
