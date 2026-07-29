package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	UpdateconfTargetsHeader = "Targets"
	UpdateconfRestartHeader = "Restart"
	UpdateconfResetHeader   = "Reset"
	UpdateconfDryRunHeader  = "Dry-Run"

	jsonFormat = "application/json"
	yamlFormat = "application/yaml"
	targetAll  = "all"
)

type importFile struct{ *http.Request }

func (i *importFile) Read(b []byte) (int, error) {
	return i.Body.Read(b) //nolint:wrapcheck //wrapping adds nothing here
}

func (i *importFile) Name() string {
	switch i.Header.Get("Content-Type") {
	case yamlFormat:
		return "import.yaml"
	case jsonFormat:
		return "import.json"
	default:
		return "import.json"
	}
}

func isHeaderSet(r *http.Request, header string) bool {
	return utils.EqualFold(r.Header.Get(header), "true", "yes", "1")
}

func updateconf(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file := &importFile{r}
		data, dataErr := backup.ParseData(file)
		if handleError(w, logger, dataErr) {
			return
		}

		targets := r.Header[UpdateconfTargetsHeader]
		if len(targets) == 0 {
			targets = []string{targetAll}
		}

		isDry := isHeaderSet(r, UpdateconfDryRunHeader)
		withReset := isHeaderSet(r, UpdateconfResetHeader)
		withRestart := isHeaderSet(r, UpdateconfRestartHeader)

		if err := backup.Import(db, logger, data, targets, isDry, withReset,
			withRestart); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

type exportFile struct {
	r *http.Request
	http.ResponseWriter
}

func (i *exportFile) setFormat() {
	switch i.r.Header.Get("Accept") {
	case yamlFormat:
		i.Header().Set("Content-Type", yamlFormat)
	case jsonFormat:
		i.Header().Set("Content-Type", jsonFormat)
	default:
		i.Header().Set("Content-Type", jsonFormat)
	}
}

func (i *exportFile) Name() string {
	switch i.r.Header.Get("Accept") {
	case yamlFormat:
		return "export.yaml"
	case jsonFormat:
		return "export.json"
	default:
		return "export.json"
	}
}

func exportconf(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targets := r.Header[UpdateconfTargetsHeader]
		if len(targets) == 0 {
			targets = []string{targetAll}
		}

		f := &exportFile{r, w}
		f.setFormat()

		if err := backup.Export(db, logger, f, targets); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
