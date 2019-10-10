package main

import (
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type transferCommand struct {
	Add transferAddCommand `command:"add" description:"Add a new transfer to be executed"`
}

// ######################## ADD ##########################

type transferAddCommand struct {
	File      string `required:"true" short:"f" long:"file" description:"The file to transfer"`
	ServerID  uint64 `required:"true" short:"s" long:"server_id" description:"The remote server with which perform the transfer"`
	AccountID uint64 `required:"true" short:"a" long:"account_id" description:"The account used to connect on the server"`
	RuleID    uint64 `required:"true" short:"r" long:"rule" description:"The rule to use for the transfer"`
}

func (t *transferAddCommand) Execute(_ []string) error {
	newTransfer := model.Transfer{
		RemoteID:    t.ServerID,
		AccountID:   t.AccountID,
		Source:      t.File,
		RuleID:      t.RuleID,
		Destination: t.File,
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.TransfersPath

	_, err = addCommand(newTransfer, conn)
	if err != nil {
		return err
	}

	return nil
}
