package sftp

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func makeCerts(ctx *pipelinetest.SelfContext) []*model.Crypto {
	return []*model.Crypto{makeServerKey(ctx.Server), makePartnerKey(ctx.Partner)}
}

func makeServerKey(serv *model.LocalAgent) *model.Crypto {
	return &model.Crypto{
		LocalAgentID: utils.NewNullInt64(serv.ID),
		Name:         "sftp_server_cert",
		PrivateKey:   testhelpers.RSAPk,
	}
}

func makePartnerKey(part *model.RemoteAgent) *model.Crypto {
	return &model.Crypto{
		RemoteAgentID: utils.NewNullInt64(part.ID),
		Name:          "sftp_partner_cert",
		SSHPublicKey:  testhelpers.SSHPbk,
	}
}
