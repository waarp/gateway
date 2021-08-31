package sftp

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"
)

func makeCerts(ctx *pipelinetest.SelfContext) []model.Crypto {
	return []model.Crypto{makeServerKey(ctx.Server), makePartnerKey(ctx.Partner)}
}

func makeServerKey(serv *model.LocalAgent) model.Crypto {
	return model.Crypto{
		OwnerType:  serv.TableName(),
		OwnerID:    serv.ID,
		Name:       "sftp_server_cert",
		PrivateKey: rsaPK,
	}
}

func makePartnerKey(part *model.RemoteAgent) model.Crypto {
	return model.Crypto{
		OwnerType:    part.TableName(),
		OwnerID:      part.ID,
		Name:         "sftp_partner_cert",
		SSHPublicKey: rsaPBK,
	}
}
