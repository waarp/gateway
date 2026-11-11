package analytics

import (
	"context"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	ServiceName = "Analytics"

	// Bound of the transfers and of the history entries read at start-up to
	// seed the directories seen by the transfers (seedLimit).
	seedLimit = 1000
)

//nolint:gochecknoglobals //global var is better for simplicity
var GlobalService *Service

type Service struct {
	analytics

	DB *database.DB

	logger       *log.Logger
	state        utils.State
	stopSampling context.CancelFunc
}

func (s *Service) Name() string { return ServiceName }

func (s *Service) Start() error {
	if s.state.IsRunning() {
		return nil
	}

	if err := s.start(); err != nil {
		s.logger.Errorf("Failed to start service: %v", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *Service) start() error {
	if s.logger == nil {
		s.logger = logging.NewLogger(ServiceName)
	}

	now := time.Now()
	s.StartTime.Store(&now)

	s.seedSeen(now)

	ctx, cancel := context.WithCancel(context.Background())
	s.stopSampling = cancel

	settings := s.DiskSettings()

	go s.sampleEvery(ctx, SamplePeriod, settings.MeasurePeriod, settings.SurveyPeriod)

	return nil
}

// sampleEvery samples the counters now, then every period; measures the
// volumes now, then every diskPeriod; and surveys the directories now, then
// every surveyPeriod, each cycle in its own routine so that a listing never
// delays a measure; until ctx ends.
func (s *Service) sampleEvery(ctx context.Context, period, diskPeriod, surveyPeriod time.Duration) {
	s.Sample(time.Now())
	s.measureDisks(ctx, time.Now())

	go s.Survey(ctx, time.Now())

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	disks := time.NewTicker(diskPeriod)
	defer disks.Stop()

	survey := time.NewTicker(surveyPeriod)
	defer survey.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.Sample(now)
		case now := <-disks.C:
			s.measureDisks(ctx, now)
		case now := <-survey.C:
			go s.Survey(ctx, now)
		}
	}
}

// measureDisks measures, in the background, the volumes of every local
// directory the transfers use.
func (s *Service) measureDisks(ctx context.Context, now time.Time) {
	s.Measure(ctx, s.localDirs(now), now)
}

// localDirs lists the directories to measure: the gateway's own (given by
// the daemon), those of the servers and the rules and the destinations of the
// tasks (read in the database), then the directories the transfers used
// within the window; the measure keeps each path once.
func (s *Service) localDirs(now time.Time) []LocalDir {
	var list []LocalDir

	for _, d := range s.Dirs() {
		list = append(list, LocalDir{Path: d.Path, Role: d.Name, Origin: OriginConfigured})
	}

	if s.DB != nil {
		list = append(list, s.configuredDirs()...)
	}

	return append(list, s.seenDirs(now)...)
}

// configuredDirs reads in the database the directories of the servers (root,
// receive, send, temporary), of the rules (local, temporary receive) and the
// fixed part of the destinations of the tasks, resolved as the transfers
// resolve them, each with the name and the identifier of its server or of
// its rule; a reading that fails is logged and leaves its part out.
func (s *Service) configuredDirs() []LocalDir {
	var (
		servers model.LocalAgents
		rules   model.Rules
		tasks   model.Tasks
		out     []LocalDir
	)

	if err := s.DB.Select(&servers).Run(); err != nil {
		s.logger.Warningf("Failed to read the servers for the disk measure: %v", err)
	}

	for _, srv := range servers {
		root := s.Resolve(srv.RootDir)
		for _, d := range []struct{ dir, role string }{
			{root, RoleServerRoot},
			{s.Resolve(srv.ReceiveDir, root), RoleServerReceive},
			{s.Resolve(srv.SendDir, root), RoleServerSend},
			{s.Resolve(srv.TmpReceiveDir, root), RoleServerTmp},
		} {
			if d.dir != "" {
				out = append(out, LocalDir{Path: d.dir, Role: d.role, Owner: srv.Name, OwnerID: srv.ID, Origin: OriginConfigured})
			}
		}
	}

	if err := s.DB.Select(&rules).Run(); err != nil {
		s.logger.Warningf("Failed to read the rules for the disk measure: %v", err)
	}

	names := map[int64]string{}

	for _, rule := range rules {
		names[rule.ID] = rule.Name

		for _, d := range []struct{ dir, role string }{{rule.LocalDir, RoleRuleLocal}, {rule.TmpLocalRcvDir, RoleRuleTmp}} {
			if resolved := s.Resolve(d.dir); resolved != "" {
				out = append(out, LocalDir{
					Path: resolved, Role: d.role, Owner: rule.Name, OwnerID: rule.ID, Origin: OriginConfigured,
				})
			}
		}
	}

	if err := s.DB.Select(&tasks).Run(); err != nil {
		s.logger.Warningf("Failed to read the tasks for the disk measure: %v", err)
	}

	for _, task := range tasks {
		dest, known := TaskDestinations[task.Type]
		if !known {
			continue
		}

		dir, cloud := DestinationDir(task.Args[dest.Arg], dest.Dir)
		if dir == "" {
			continue
		}

		if !cloud {
			dir = s.Resolve(dir)
		}

		out = append(out, LocalDir{
			Path: dir, Role: RoleTask, Owner: names[task.RuleID], OwnerID: task.RuleID, Origin: OriginTask,
		})
	}

	return out
}

// seedSeen records the directories of the transfers of the last SeenWindow,
// current and finished, a bounded number of each, so that the volumes they
// use are measured from the start.
func (s *Service) seedSeen(now time.Time) {
	if s.DB == nil {
		return
	}

	since := now.Add(-SeenWindow)

	var transfers model.Transfers
	if err := s.DB.Select(&transfers).Where("start > ?", since).OrderBy("start", false).
		Limit(seedLimit, 0).Run(); err != nil {
		s.logger.Warningf("Failed to read the transfers for the disk measure: %v", err)
	}

	for _, t := range transfers {
		s.Seen(t.LocalPath, t.Start)
	}

	var history model.HistoryEntries
	if err := s.DB.Select(&history).Where("start > ?", since).OrderBy("start", false).
		Limit(seedLimit, 0).Run(); err != nil {
		s.logger.Warningf("Failed to read the history for the disk measure: %v", err)
	}

	for _, h := range history {
		s.Seen(h.LocalPath, h.Start)
	}
}

func (s *Service) Stop(context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := s.stop(); err != nil {
		s.logger.Errorf("Failed to stop service: %v", err)
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *Service) stop() error {
	if s.stopSampling != nil {
		s.stopSampling()
	}

	return nil
}

func (s *Service) State() (utils.StateCode, string) { return s.state.Get() }
