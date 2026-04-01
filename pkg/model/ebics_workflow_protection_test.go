package model

import (
	"hash/fnv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestCredentialBeforeDeleteRejectsActiveEbicsLifecycleReference(t *testing.T) {
	db := dbtest.TestDatabase(t)
	credential := insertProtectedCredentialTestCredential(t, db, "cred-delete-active")
	subscriber := insertProtectedCredentialTestSubscriber(t, db, "HOST-CRED-DEL", "PARTNER-CRED-DEL", "USER-CRED-DEL")

	lifecycle := &EbicsKeyLifecycle{
		EbicsSubscriberID:    subscriber.ID,
		KeyUsage:             EbicsKeyUsageAuthenticationForRuntime(),
		RotationType:         EbicsRotationTypeRotationForRuntime(),
		Status:               "DRAFT",
		CurrentCredentialID:  credential.ID,
	}
	require.NoError(t, db.Insert(lifecycle).Run())

	err := credential.BeforeDelete(db)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot be deleted")
}

func TestCredentialBeforeWriteRejectsProtectedMutationWhenLifecycleIsActive(t *testing.T) {
	db := dbtest.TestDatabase(t)
	credential := insertProtectedCredentialTestCredential(t, db, "cred-mutation-active")
	subscriber := insertProtectedCredentialTestSubscriber(t, db, "HOST-CRED-UPD", "PARTNER-CRED-UPD", "USER-CRED-UPD")

	lifecycle := &EbicsKeyLifecycle{
		EbicsSubscriberID:    subscriber.ID,
		KeyUsage:             EbicsKeyUsageAuthenticationForRuntime(),
		RotationType:         EbicsRotationTypeRotationForRuntime(),
		Status:               "DRAFT",
		CurrentCredentialID:  credential.ID,
	}
	require.NoError(t, db.Insert(lifecycle).Run())

	credential.Value = "updated-secret"
	err := credential.BeforeWrite(db)
	require.Error(t, err)
	require.ErrorContains(t, err, "protected material cannot be modified")
}

func TestEbicsKeyLifecycleBeforeDeleteRejectsActiveLifecycle(t *testing.T) {
	db := dbtest.TestDatabase(t)
	credential := insertProtectedCredentialTestCredential(t, db, "cred-lifecycle-active")
	nextCredential := insertProtectedCredentialTestCredential(t, db, "cred-lifecycle-active-next")
	subscriber := insertProtectedCredentialTestSubscriber(t, db, "HOST-LIFE-ACT", "PARTNER-LIFE-ACT", "USER-LIFE-ACT")

	lifecycle := &EbicsKeyLifecycle{
		EbicsSubscriberID:   subscriber.ID,
		KeyUsage:            EbicsKeyUsageAuthenticationForRuntime(),
		RotationType:        EbicsRotationTypeRotationForRuntime(),
		Status:              EbicsKeyLifecycleStatusOrderSentForRuntime(),
		CurrentCredentialID: credential.ID,
		NextCredentialID:    utils.NewNullInt64(nextCredential.ID),
		RequestedAt:         mustUTC("2026-04-01T10:00:00Z"),
		SentAt:              mustUTC("2026-04-01T10:05:00Z"),
	}
	require.NoError(t, db.Insert(lifecycle).Run())

	err := lifecycle.BeforeDelete(db)
	require.Error(t, err)
	require.ErrorContains(t, err, "still active")
}

func TestEbicsKeyLifecycleBeforeDeleteAllowsTerminalLifecycle(t *testing.T) {
	db := dbtest.TestDatabase(t)
	credential := insertProtectedCredentialTestCredential(t, db, "cred-lifecycle-terminal")
	subscriber := insertProtectedCredentialTestSubscriber(t, db, "HOST-LIFE-END", "PARTNER-LIFE-END", "USER-LIFE-END")

	lifecycle := &EbicsKeyLifecycle{
		EbicsSubscriberID:   subscriber.ID,
		KeyUsage:            EbicsKeyUsageAuthenticationForRuntime(),
		RotationType:        EbicsRotationTypeRotationForRuntime(),
		Status:              EbicsKeyLifecycleStatusCancelledForRuntime(),
		CurrentCredentialID: credential.ID,
	}
	require.NoError(t, db.Insert(lifecycle).Run())

	require.NoError(t, lifecycle.BeforeDelete(db))
}

func TestEbicsInitializationBeforeDeleteRejectsActiveWorkflow(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertProtectedCredentialTestSubscriber(t, db, "HOST-INIT-ACT", "PARTNER-INIT-ACT", "USER-INIT-ACT")

	workflow := &EbicsInitializationWorkflow{
		EbicsSubscriberID: subscriber.ID,
		Status:            "WAITING_BANK_ACTIVATION",
		CurrentStep:       "WAITING_BANK_ACTIVATION",
	}
	require.NoError(t, db.Insert(workflow).Run())

	err := workflow.BeforeDelete(db)
	require.Error(t, err)
	require.ErrorContains(t, err, "still active")
}

func TestEbicsInitializationBeforeDeleteAllowsCancelledWorkflow(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertProtectedCredentialTestSubscriber(t, db, "HOST-INIT-END", "PARTNER-INIT-END", "USER-INIT-END")

	workflow := &EbicsInitializationWorkflow{
		EbicsSubscriberID: subscriber.ID,
		Status:            "CANCELLED",
		CurrentStep:       "CANCELLED",
	}
	require.NoError(t, db.Insert(workflow).Run())

	require.NoError(t, workflow.BeforeDelete(db))
}

func insertProtectedCredentialTestCredential(t *testing.T, db database.Access, name string) *Credential {
	t.Helper()

	agent := &LocalAgent{
		Name:     name + "-agent",
		Protocol: testProtocol,
		Address:  types.Addr("localhost", uniqueTestPort(name)),
	}
	require.NoError(t, db.Insert(agent).Run())

	credential := &Credential{
		LocalAgentID: utils.NewNullInt64(agent.ID),
		Name:         name,
		Type:         testExternalAuth,
		Value:        "secret",
	}
	require.NoError(t, db.Insert(credential).Run())

	return credential
}

func insertProtectedCredentialTestSubscriber(
	t *testing.T,
	db database.Access,
	hostID, partnerID, userID string,
) *EbicsSubscriber {
	t.Helper()

	host := &EbicsHost{
		Name:            hostID,
		HostID:          hostID,
		Enabled:         true,
		IsServer:        true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &EbicsSubscriber{
		Name:        partnerID + ":" + userID,
		EbicsHostID: host.ID,
		PartnerID:   partnerID,
		UserID:      userID,
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	return subscriber
}

func mustUTC(raw string) (out time.Time) {
	out, _ = time.Parse(time.RFC3339, raw)
	return out.UTC()
}

func uniqueTestPort(seed string) uint16 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(seed))
	return uint16(10000 + (hasher.Sum32() % 50000))
}
