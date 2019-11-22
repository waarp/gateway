package sftp

import (
	"io"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type progress struct {
	ID     uint64
	Bytes  uint64
	Status string
}

type downloadStream struct {
	File   *os.File
	ID     uint64
	Report chan<- progress

	fail  error
	bytes uint64
}

func (d *downloadStream) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = d.File.WriteAt(p, off)
	if err != nil {
		d.fail = err
	}
	d.bytes += uint64(n)
	d.Report <- progress{ID: d.ID, Bytes: d.bytes, Status: string(model.StatusTransfer)}

	return
}

func (d *downloadStream) TransferError(err error) {
	d.fail = err
}

func (d *downloadStream) Close() error {
	if d.fail == nil {
		d.Report <- progress{ID: d.ID, Bytes: d.bytes, Status: string(model.StatusDone)}
	} else {
		d.Report <- progress{ID: d.ID, Bytes: d.bytes, Status: string(model.StatusError)}
	}

	return nil
}

type uploadStream struct {
	File   *os.File
	ID     uint64
	Report chan<- progress

	fail  error
	bytes uint64
}

func (u *uploadStream) ReadAt(p []byte, off int64) (n int, err error) {
	u.Report <- progress{ID: u.ID, Bytes: u.bytes, Status: string(model.StatusTransfer)}

	n, err = u.File.ReadAt(p, off)
	if err != nil && err != io.EOF {
		u.fail = err
	}
	u.bytes += uint64(n)

	return
}

func (u *uploadStream) TransferError(err error) {
	u.fail = err
}

func (u *uploadStream) Close() error {
	if u.fail == nil {
		u.Report <- progress{ID: u.ID, Bytes: u.bytes, Status: string(model.StatusDone)}
	} else {
		u.Report <- progress{ID: u.ID, Bytes: u.bytes, Status: string(model.StatusError)}
	}

	return nil
}
