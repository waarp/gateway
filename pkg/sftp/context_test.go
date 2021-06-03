package sftp

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"
)

func makeCerts(ctx *pipelinetest.SelfContext) []model.Cert {
	return []model.Cert{makeServerKey(ctx.Server), makePartnerKey(ctx.Partner)}
}

func makeServerKey(serv *model.LocalAgent) model.Cert {
	return model.Cert{
		OwnerType:  serv.TableName(),
		OwnerID:    serv.ID,
		Name:       "sftp_server_cert",
		PrivateKey: []byte(rsaPK),
		PublicKey:  []byte(rsaPBK),
	}
}

func makePartnerKey(part *model.RemoteAgent) model.Cert {
	return model.Cert{
		OwnerType:  part.TableName(),
		OwnerID:    part.ID,
		Name:       "sftp_partner_cert",
		PrivateKey: []byte(rsaPK),
		PublicKey:  []byte(rsaPBK),
	}
}
