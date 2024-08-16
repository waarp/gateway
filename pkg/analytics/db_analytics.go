package analytics

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var ErrNoTransfer = errors.New("no transfer found")

func (s *Service) GetLastTransfer() (*model.NormalizedTransferView, error) {
	var trans model.NormalizedTransfers
	if err := s.DB.Select(&trans).Where("owner=?", conf.GlobalConfig.GatewayName).
		Limit(1, 0).OrderBy("start", false).Run(); err != nil {
		s.logger.Error("Failed to retrieve last transfer: %v", err)

		return nil, fmt.Errorf("failed to retrieve last transfer: %w", err)
	}

	if len(trans) == 0 {
		return nil, ErrNoTransfer
	}

	return trans[0], nil
}

func (s *Service) CountTransferWithStatus(status types.TransferStatus) (uint64, error) {
	query := s.DB.Count(&model.NormalizedTransferView{}).Where("owner = ?",
		conf.GlobalConfig.GatewayName)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	count, err := query.Run()
	if err != nil {
		s.logger.Error("Failed to count transfers: %v", err)

		return 0, fmt.Errorf("failed to count transfers: %w", err)
	}

	return count, nil
}

func (s *Service) CountTransfersWithErrorCode(code types.TransferErrorCode) (uint64, error) {
	count, err := s.DB.Count(&model.NormalizedTransferView{}).Where("owner = ?",
		conf.GlobalConfig.GatewayName).Where("error_code = ?", code).Run()
	if err != nil {
		s.logger.Error("Failed to count transfers: %v", err)

		return 0, fmt.Errorf("failed to count transfers: %w", err)
	}

	return count, nil
}
