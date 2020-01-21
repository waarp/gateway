package pipeline

import (
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

type pipeline struct {
	Info   model.InTransferInfo
	Db     *database.Db
	Logger *log.Logger
	Root   string

	signals chan model.Signal
	proc    *tasks.Processor
}

func (p *pipeline) createTransfer() model.TransferError {
	err := p.Db.Create(&p.Info.Transfer)
	if err != nil {
		p.Logger.Criticalf("Failed to create transfer entry: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}
	return model.TransferError{}
}

func (p *pipeline) makeFile() (*os.File, model.TransferError) {
	if p.Info.Rule.IsSend {
		path := filepath.Clean(filepath.Join(p.Root, p.Info.Rule.Path, p.Info.Transfer.SourcePath))
		file, err := os.Open(path)
		if err != nil {
			p.Logger.Errorf("Failed to open source file: %s", err)
			t := &model.Transfer{Error: model.NewTransferError(model.TeFileNotFound, err.Error())}
			if dbErr := p.Db.Update(t, p.Info.Transfer.ID, false); dbErr != nil {
				return nil, model.NewTransferError(model.TeInternal, dbErr.Error())
			}
			return nil, t.Error
		}
		return file, model.TransferError{}
	}

	path := filepath.Clean(filepath.Join(p.Root, p.Info.Rule.Path, p.Info.Transfer.DestPath))
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		p.Logger.Errorf("Failed to create destination file: %s", err)
		t := &model.Transfer{Error: model.NewTransferError(model.TeForbidden, err.Error())}
		if dbErr := p.Db.Update(t, p.Info.Transfer.ID, false); dbErr != nil {
			return nil, model.NewTransferError(model.TeInternal, dbErr.Error())
		}
		return nil, t.Error
	}
	return file, model.TransferError{}
}

func (p *pipeline) prologue() (*os.File, model.TransferError) {
	if err := p.createTransfer(); err.Code != model.TeOk {
		return nil, err
	}

	p.signals = make(chan model.Signal)
	Signals.Store(p.Info.Transfer.ID, p.signals)

	p.proc = &tasks.Processor{
		Db:       p.Db,
		Logger:   p.Logger,
		Rule:     &p.Info.Rule,
		Transfer: &p.Info.Transfer,
		Signals:  p.signals,
	}

	file, err := p.makeFile()
	if err.Code != model.TeOk {
		return nil, err
	}

	if err := PreTasks(p.Db, p.Logger, p.proc); err.Code != model.TeOk {
		return nil, err
	}

	stat := &model.Transfer{Status: model.StatusPreTasks}
	if err := p.Db.Update(stat, p.Info.Transfer.ID, false); err != nil {
		p.Logger.Criticalf("Failed to update transfer status: %s", err)
		return nil, model.NewTransferError(model.TeInternal, err.Error())
	}

	return file, model.TransferError{}
}

func (p *pipeline) epilogue() model.TransferError {
	if err := PostTasks(p.Db, p.Logger, p.proc); err.Code != model.TeOk {
		return err
	}

	ToHistory(p.Db, p.Logger, p.Info.Transfer.ID, false)
	return model.TransferError{}
}

func (p *pipeline) handleError(err model.TransferError) {
	t := &model.Transfer{Error: err}
	if err := p.Db.Update(t, p.Info.Transfer.ID, false); err != nil {
		p.Logger.Criticalf("Failed to update transfer status: %s", err)
		return
	}

	_ = ErrorTasks(p.Db, p.Logger, p.proc, err)
	ToHistory(p.Db, p.Logger, p.Info.Transfer.ID, true)
}
